package engine

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"project-tachyon/internal/analytics"
	"project-tachyon/internal/filesystem"
	"project-tachyon/internal/integrity"
	"project-tachyon/internal/network"
	"project-tachyon/internal/queue"
	"project-tachyon/internal/security"
	"project-tachyon/internal/storage"
)

// Configurable constants
const (
	DownloadChunkSize = 1 * 1024 * 1024 // 1MB Part Size
	BufferSize        = 32 * 1024       // 32KB Buffer for CopyBuffer
	GenericUserAgent  = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36"

	// Status for tasks needing URL refresh (403 received)
	StatusNeedsAuth = "needs_auth"
)

// TachyonEngine is the core download orchestrator
type TachyonEngine struct {
	logger          *slog.Logger
	storage         *storage.Storage
	ctx             context.Context
	queue           *queue.DownloadQueue
	scheduler       *queue.SmartScheduler
	activeDownloads sync.Map // map[string]*activeDownloadInfo
	bufferPool      *sync.Pool
	httpClient      *http.Client
	stats           *analytics.StatsManager

	// Concurrency Control
	maxConcurrent    int
	runningDownloads int
	workerCond       *sync.Cond
	workerMutex      sync.Mutex

	// Bandwidth & Traffic
	bandwidthManager *network.BandwidthManager

	// integrity
	allocator *filesystem.Allocator
	verifier  *integrity.FileVerifier

	// utilities
	organizer *filesystem.SmartOrganizer

	// Phase 7 Components
	stateManager         *StateManager
	congestionController *network.CongestionController

	// Security
	scanner security.Scanner

	// Custom User-Agent (thread-safe)
	userAgentMu sync.RWMutex
	userAgent   string
}

// NewEngine creates a new TachyonEngine instance
func NewEngine(logger *slog.Logger, storage *storage.Storage) *TachyonEngine {
	// Custom Transport for Connection Reuse
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100, // Global pool size
		MaxIdleConnsPerHost:   32,  // Allow high concurrency per host
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true, // We want raw bytes
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   0, // No timeout for the client itself, request contexts handles it
	}

	q := queue.NewDownloadQueue()
	s := queue.NewSmartScheduler(logger, q)

	e := &TachyonEngine{
		logger:          logger,
		storage:         storage,
		queue:           q,
		scheduler:       s,
		activeDownloads: sync.Map{},
		bufferPool: &sync.Pool{
			New: func() interface{} {
				// Allocate 32KB buffer
				b := make([]byte, BufferSize)
				return &b
			},
		},
		httpClient:           client,
		stats:                analytics.NewStatsManager(storage, filesystem.GetDefaultDownloadPath),
		maxConcurrent:        5, // System wide limit of downloads
		runningDownloads:     0,
		bandwidthManager:     network.NewBandwidthManager(),
		allocator:            filesystem.NewAllocator(),
		verifier:             integrity.NewFileVerifier(),
		organizer:            filesystem.NewSmartOrganizer(),
		stateManager:         NewStateManager(),
		congestionController: network.NewCongestionController(1, 32),
		scanner:              security.NewScanner(logger),
	}
	e.workerCond = sync.NewCond(&e.workerMutex)

	go e.queueWorker()
	return e
}

// SetContext sets the Wails context for event emission
func (e *TachyonEngine) SetContext(ctx context.Context) {
	e.ctx = ctx
	// Recover any downloads that were interrupted by app close
	e.RecoverInterruptedDownloads()
}

// Shutdown gracefully stops the engine
func (e *TachyonEngine) Shutdown() error {
	e.logger.Info("Engine shutting down...")

	// 1. Cancel all active downloads
	var shutdownWg sync.WaitGroup
	e.activeDownloads.Range(func(key, value interface{}) bool {
		if info, ok := value.(*activeDownloadInfo); ok {
			if info.Cancel != nil {
				info.Cancel()
			}
			shutdownWg.Add(1)
			go func() {
				shutdownWg.Done()
			}()
		}
		return true
	})

	// Wait for workers to cleanup (max 2 seconds)
	deadline := time.Now().Add(2 * time.Second)
	for {
		e.workerMutex.Lock()
		count := e.runningDownloads
		e.workerMutex.Unlock()
		if count == 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 2. Force Checkpoint
	if err := e.storage.Checkpoint(); err != nil {
		e.logger.Error("Failed to checkpoint DB", "error", err)
		return err
	}
	e.logger.Info("Engine shutdown complete")
	return nil
}

// RecoverInterruptedDownloads finds downloads stuck in "downloading" or "pending" status
// and moves them to "paused" so they can be manually resumed
func (e *TachyonEngine) RecoverInterruptedDownloads() {
	tasks, err := e.storage.GetAllTasks()
	if err != nil {
		e.logger.Error("Failed to recover interrupted downloads", "error", err)
		return
	}

	for _, task := range tasks {
		if task.Status == "downloading" || task.Status == "pending" {
			// Move to paused state
			task.Status = "paused"
			if err := e.storage.SaveTask(task); err != nil {
				e.logger.Error("Failed to pause interrupted download", "id", task.ID, "error", err)
				continue
			}
			e.logger.Info("Recovered interrupted download", "id", task.ID, "filename", task.Filename)
		}
	}
}

// GetStorage returns the storage instance
func (e *TachyonEngine) GetStorage() *storage.Storage {
	return e.storage
}

// GetUserAgent returns the current custom User-Agent (thread-safe)
func (e *TachyonEngine) GetUserAgent() string {
	e.userAgentMu.RLock()
	defer e.userAgentMu.RUnlock()
	return e.userAgent
}

// SetUserAgent sets a custom User-Agent for all requests (thread-safe)
func (e *TachyonEngine) SetUserAgent(ua string) {
	e.userAgentMu.Lock()
	defer e.userAgentMu.Unlock()
	e.userAgent = ua
	e.logger.Info("User-Agent updated", "user_agent", ua)
}

// GetStats returns the stats manager
func (e *TachyonEngine) GetStats() *analytics.StatsManager {
	return e.stats
}

// SetMaxConcurrent sets the maximum number of concurrent downloads
func (e *TachyonEngine) SetMaxConcurrent(n int) {
	e.workerMutex.Lock()
	defer e.workerMutex.Unlock()

	if n < 1 {
		n = 1
	}
	if n > 10 {
		n = 10
	}
	e.maxConcurrent = n
	// Signal to check if more can be started
	e.workerCond.Signal()
}

// SetGlobalLimit sets the global download speed limit
func (e *TachyonEngine) SetGlobalLimit(bytesPerSec int) {
	e.bandwidthManager.SetLimit(bytesPerSec)
}

// SetHostLimit sets the per-host connection limit
func (e *TachyonEngine) SetHostLimit(domain string, limit int) {
	e.scheduler.SetHostLimit(domain, limit)
}

// GetHostLimit returns the per-host connection limit
func (e *TachyonEngine) GetHostLimit(domain string) int {
	return e.scheduler.GetHostLimit(domain)
}
