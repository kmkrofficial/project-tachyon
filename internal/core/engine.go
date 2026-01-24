package core

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"project-tachyon/internal/storage"
	"sync"
	"time"

	"github.com/cavaliergopher/grab/v3"
	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type TachyonEngine struct {
	logger          *slog.Logger
	storage         *storage.Storage
	ctx             context.Context
	activeDownloads sync.Map
	queue           *DownloadQueue
	semaphore       chan struct{}
	stats           *StatsManager
}

func NewEngine(logger *slog.Logger, storage *storage.Storage) *TachyonEngine {
	e := &TachyonEngine{
		logger:    logger,
		storage:   storage,
		queue:     NewDownloadQueue(),
		semaphore: make(chan struct{}, 3),
		stats:     NewStatsManager(storage),
	}
	go e.queueWorker()
	return e
}

func (e *TachyonEngine) SetContext(ctx context.Context) {
	e.ctx = ctx
}

func (e *TachyonEngine) GetHistory() ([]storage.Task, error) {
	return e.storage.GetAllTasks()
}

func (e *TachyonEngine) GetTask(id string) (storage.Task, error) {
	return e.storage.GetTask(id)
}

// GetStats returns the stats manager for analytics
func (e *TachyonEngine) GetStats() *StatsManager {
	return e.stats
}

func (e *TachyonEngine) SetMaxConcurrent(n int) {
	if n < 1 {
		return
	}
	// Resizing channel hard at runtime, easier to just update a specialized semaphore or use weighted semaphore.
	// For Phase 3.75 simple semaphore channel replacement or just logging TODO.
	// Ideally we use golang.org/x/sync/semaphore but sticking to stdlib.
	// We will stick to fixed 3 for now or implement dynamic adjustment later.
}

func (e *TachyonEngine) queueWorker() {
	for {
		task := e.queue.Pop() // Blocks until available
		if task == nil {
			continue
		}

		e.semaphore <- struct{}{} // Acquire token (blocks if full)

		go func(t *storage.Task) {
			defer func() { <-e.semaphore }() // Release token

			// Re-create req logic here or call internal helper
			// We need to re-initialize client/req because they weren't stored in Queue, only the Task model.
			// This means we might need to store URL in Task (we do).
			e.executeTask(t)
		}(task)
	}
}

func (e *TachyonEngine) executeTask(task *storage.DownloadTask) {
	// Client Initialization
	client := grab.NewClient()
	client.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	req, err := grab.NewRequest(task.SavePath, task.URL)
	if err != nil {
		e.logger.Error("Failed to create request for queued task", "id", task.ID, "error", err)
		task.Status = "error"
		e.storage.SaveTask(*task)
		return
	}

	e.processDownload(client, req, task.ID, *task)
}

func (e *TachyonEngine) StartDownload(urlStr string, destPath string) (string, error) {
	// 1. Validation
	if urlStr == "" {
		return "", fmt.Errorf("empty URL")
	}

	// Generate Download ID
	downloadID := uuid.New().String()

	// Categorization
	// Guess filename from URL for categorization
	guessedFilename := filepath.Base(urlStr) // Simple extraction, robust enough for category
	if guessedFilename == "" || guessedFilename == "." {
		guessedFilename = "unknown"
	}

	finalPath, _ := GetOrganizedPath(destPath, guessedFilename)
	category := GetCategory(guessedFilename)

	// Create Initial Task Record (using new SQLite model)
	task := storage.DownloadTask{
		ID:        downloadID,
		URL:       urlStr,
		Filename:  filepath.Base(finalPath),
		SavePath:  finalPath,
		Status:    "pending",
		Category:  category,
		Priority:  1, // Default Normal
		Progress:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := e.storage.SaveTask(task); err != nil {
		e.logger.Error("Failed to save initial task", "error", err)
	}

	// 4. Enqueue
	e.queue.Push(&task)

	// Notify UI of "Pending" state immediately
	if e.ctx != nil {
		// Emit event or rely on GetTasks polling if any?
		// Best to emit "download:queued" or just let the "progress" logic pick it up when it starts.
		// But UI expects "download:progress" to see it.
		// Let's emit an initial packet.
		runtime.EventsEmit(e.ctx, "download:progress", map[string]interface{}{
			"id":       downloadID,
			"progress": 0,
			"status":   "pending",
			"filename": task.Filename,
		})
	}

	return downloadID, nil
}

func (e *TachyonEngine) processDownload(client *grab.Client, req *grab.Request, id string, task storage.DownloadTask) {
	e.logger.Info("Download Started", "id", id, "url", req.URL().String(), "dest", req.Filename)

	resp := client.Do(req)
	e.activeDownloads.Store(id, resp)

	// Track bytes for stats - we'll track delta between ticks
	var lastBytesCompleted int64 = 0
	defer e.activeDownloads.Delete(id)

	// Update Filename if grab resolved a better one (e.g. from Content-Disposition)
	if resp.Filename != "" {
		task.Filename = filepath.Base(resp.Filename)
		task.SavePath = resp.Filename
	}

	// Ticker for progress monitoring (UI updates)
	uiTicker := time.NewTicker(200 * time.Millisecond)
	defer uiTicker.Stop()

	// Ticker for DB persistence (Slower to save I/O)
	dbTicker := time.NewTicker(2 * time.Second)
	defer dbTicker.Stop()

Loop:
	for {
		select {
		case <-uiTicker.C:
			// Calculate metrics
			progress := resp.Progress() * 100
			speedMBs := float64(resp.BytesPerSecond()) / 1024 / 1024

			// Broadcast to UI
			if e.ctx != nil {
				runtime.EventsEmit(e.ctx, "download:progress", map[string]interface{}{
					"id":        id,
					"progress":  progress,
					"speed_MBs": speedMBs,
					"eta":       resp.ETA().String(),
					"filename":  task.Filename,
				})
			}

			// Log occasionally/debug level usually, keeping Info for now based on req
			// e.logger.Info(...)

			if resp.IsComplete() {
				break Loop
			}

		case <-dbTicker.C:
			// Checkpoint: Persist state to SQLite (every 2 seconds)
			task.Progress = resp.Progress() * 100
			task.TotalSize = resp.Size()
			task.Downloaded = resp.BytesComplete()
			task.Speed = float64(resp.BytesPerSecond())
			task.Status = "downloading"
			if err := e.storage.SaveTask(task); err != nil {
				e.logger.Error("Failed to save task state", "id", id, "error", err)
			}

			// Track bytes downloaded for lifetime stats (debounced every 2 seconds)
			currentBytes := resp.BytesComplete()
			deltaBytes := currentBytes - lastBytesCompleted
			if deltaBytes > 0 {
				e.stats.TrackDownloadBytes(deltaBytes)
				lastBytesCompleted = currentBytes
			}

		case <-resp.Done:
			// Channel closed when download finishes or cancels
			break Loop
		}
	}

	// Final status check & Persistence
	if err := resp.Err(); err != nil {
		task.Status = "error"
		task.Progress = resp.Progress() * 100 // Save progress where it failed
		e.logger.Error("Download Failed", "id", id, "error", err.Error())

		if e.ctx != nil {
			runtime.EventsEmit(e.ctx, "download:failed", map[string]interface{}{
				"id":    id,
				"error": err.Error(),
			})
		}
	} else {
		task.Status = "completed"
		task.Progress = 100
		e.logger.Info("Download Completed", "id", id, "path", resp.Filename)

		// Track any remaining bytes not captured by the ticker
		finalBytes := resp.BytesComplete()
		deltaBytes := finalBytes - lastBytesCompleted
		if deltaBytes > 0 {
			e.stats.TrackDownloadBytes(deltaBytes)
		}

		// Track completed file for analytics
		e.stats.TrackFileCompleted()

		if e.ctx != nil {
			runtime.EventsEmit(e.ctx, "download:completed", map[string]interface{}{
				"id":   id,
				"path": resp.Filename,
			})
		}
	}

	// Final Save
	if err := e.storage.SaveTask(task); err != nil {
		e.logger.Error("Failed to save final task state", "id", id, "error", err)
	}
}
