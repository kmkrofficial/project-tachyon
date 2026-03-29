package engine

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
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
	DownloadChunkSize = 4 * 1024 * 1024 // 4MB Part Size — fewer HTTP requests, better TCP ramp-up
	BufferSize        = 1 * 1024 * 1024 // 1MB Buffer — fewer read loop iterations on fast links
	MaxWorkersPerTask = 24              // Aggressive upper bound; dynamic tuning chooses active count
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
	allowLoopback   bool     // allow 127.0.0.1 downloads (testing only)
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
	congestion       *network.CongestionController
	breaker          *network.CircuitBreaker
	hostSingleStream sync.Map // map[string]bool

	// Download tuning knobs
	maxWorkersPerTask int
	baseChunkSize     int64

	// integrity
	allocator *filesystem.Allocator
	verifier  *integrity.FileVerifier

	// utilities
	organizer *filesystem.SmartOrganizer

	// Phase 7 Components
	stateManager *StateManager

	// Security
	scanner security.Scanner

	// Global goroutine pool for download workers
	workerPool *WorkerPool

	// Probe cache — reuses recent probes to skip redundant network calls
	probes *probeCache

	// Custom User-Agent (thread-safe)
	userAgentMu sync.RWMutex
	userAgent   string
}

// NewEngine creates a new TachyonEngine instance
func NewEngine(logger *slog.Logger, storage *storage.Storage) *TachyonEngine {
	// DNS cache reduces lookup latency on multi-part downloads to the same host
	dnsCache := network.NewDNSCache(5 * time.Minute)

	// Custom Transport for Connection Reuse + HTTP/2 multiplexing
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dnsCache.DialContext(30*time.Second, 30*time.Second),
		MaxIdleConns:          100, // Global pool size
		MaxIdleConnsPerHost:   32,  // Allow high concurrency per host
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second, // Bound header wait to detect dead connections
		DisableCompression:    true,             // We want raw bytes
		ForceAttemptHTTP2:     true,             // Enable HTTP/2 multiplexing
		ReadBufferSize:        128 * 1024,       // 128KB — reduces syscalls on fast links
		WriteBufferSize:       32 * 1024,        // 32KB — sufficient for request headers
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
				b := make([]byte, BufferSize)
				return &b
			},
		},
		httpClient:        client,
		stats:             analytics.NewStatsManager(storage, filesystem.GetDefaultDownloadPath),
		maxConcurrent:     5, // System wide limit of downloads
		runningDownloads:  0,
		bandwidthManager:  network.NewBandwidthManager(),
		congestion:        network.NewCongestionController(4, MaxWorkersPerTask),
		breaker:           network.NewCircuitBreaker(5, 30*time.Second),
		maxWorkersPerTask: MaxWorkersPerTask,
		baseChunkSize:     0,
		allocator:         filesystem.NewAllocator(),
		verifier:          integrity.NewFileVerifier(),
		organizer:         filesystem.NewSmartOrganizer(),
		stateManager:      NewStateManager(),
		scanner:           security.NewScanner(logger),
		workerPool:        NewWorkerPool(64), // Global pool — covers all concurrent download workers
		probes:            newProbeCache(),
	}
	e.workerCond = sync.NewCond(&e.workerMutex)

	go e.queueWorker()
	return e
}

// SetDownloadTuning overrides worker and chunk tuning at runtime.
func (e *TachyonEngine) SetDownloadTuning(maxWorkers int, baseChunkBytes int64) {
	e.workerMutex.Lock()
	if maxWorkers < 1 {
		maxWorkers = 1
	}
	if maxWorkers > 64 {
		maxWorkers = 64
	}
	e.maxWorkersPerTask = maxWorkers
	e.workerMutex.Unlock()

	if baseChunkBytes < 0 {
		baseChunkBytes = 0
	}
	e.baseChunkSize = baseChunkBytes

	// Keep congestion controller bounds aligned with worker caps.
	e.congestion = network.NewCongestionController(4, maxWorkers)
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

	// 1. Record IDs of actively running downloads so they can auto-resume on restart
	var activeIDs []string
	e.activeDownloads.Range(func(key, value interface{}) bool {
		activeIDs = append(activeIDs, key.(string))
		return true
	})
	// Also include pending tasks (queued but not yet started)
	if tasks, err := e.storage.GetAllTasks(); err == nil {
		for _, t := range tasks {
			if t.Status == "pending" || t.Status == "probing" {
				activeIDs = append(activeIDs, t.ID)
			}
		}
	}
	if len(activeIDs) > 0 {
		if err := e.storage.SetString("auto_resume_ids", joinIDs(activeIDs)); err != nil {
			e.logger.Error("Failed to save auto-resume IDs", "error", err)
		}
	} else {
		_ = e.storage.SetString("auto_resume_ids", "")
	}

	// 2. Cancel all active downloads
	e.activeDownloads.Range(func(key, value interface{}) bool {
		if info, ok := value.(*activeDownloadInfo); ok {
			if info.Cancel != nil {
				info.Cancel()
			}
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

	// 3. Force Checkpoint
	if err := e.storage.Checkpoint(); err != nil {
		e.logger.Error("Failed to checkpoint DB", "error", err)
		return err
	}

	// 4. Drain global worker pool
	e.workerPool.Close()

	e.logger.Info("Engine shutdown complete")
	return nil
}

// RecoverInterruptedDownloads finds downloads that were actively running when the
// app last closed and auto-resumes them.  Downloads that were manually paused,
// stopped, or in error are left untouched.
func (e *TachyonEngine) RecoverInterruptedDownloads() {
	tasks, err := e.storage.GetAllTasks()
	if err != nil {
		e.logger.Error("Failed to recover interrupted downloads", "error", err)
		return
	}

	// Build set of IDs that were actively running during the last graceful shutdown
	autoResumeSet := make(map[string]bool)
	if raw, err := e.storage.GetString("auto_resume_ids"); err == nil && raw != "" {
		for _, id := range splitIDs(raw) {
			autoResumeSet[id] = true
		}
	}
	// Clear the marker so it doesn't persist across future restarts
	_ = e.storage.SetString("auto_resume_ids", "")

	var toResume []string

	for _, task := range tasks {
		switch task.Status {
		case "downloading", "pending", "probing", "merging":
			// Active at close — always auto-resume regardless of whether
			// shutdown was graceful or abrupt.
			task.Status = "paused"
			if err := e.storage.SaveTask(task); err != nil {
				e.logger.Error("Failed to pause interrupted download", "id", task.ID, "error", err)
				continue
			}
			toResume = append(toResume, task.ID)
			e.logger.Info("Recovered interrupted download (will auto-resume)", "id", task.ID, "filename", task.Filename)

		case "paused":
			// Graceful shutdown — only auto-resume if it was active before shutdown
			if autoResumeSet[task.ID] {
				toResume = append(toResume, task.ID)
				e.logger.Info("Auto-resuming gracefully-paused download", "id", task.ID, "filename", task.Filename)
			}
		}
	}

	// Auto-resume after a short delay to let the UI initialise
	if len(toResume) > 0 {
		go func() {
			time.Sleep(2 * time.Second)
			for _, id := range toResume {
				if err := e.ResumeDownload(id); err != nil {
					e.logger.Error("Failed to auto-resume download", "id", id, "error", err)
				}
			}
		}()
	}
}

// GetStorage returns the storage instance
func (e *TachyonEngine) GetStorage() *storage.Storage {
	return e.storage
}

// joinIDs concatenates non-empty IDs with a comma separator.
func joinIDs(ids []string) string { return strings.Join(ids, ",") }

// splitIDs splits a comma-separated string back into individual IDs.
func splitIDs(raw string) []string {
	if raw == "" {
		return nil
	}
	return strings.Split(raw, ",")
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
