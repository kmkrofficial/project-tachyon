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
	logger  *slog.Logger
	storage *storage.Storage
	ctx     context.Context
	// ActiveDownloads maps DownloadID (string) to *grab.Response
	activeDownloads sync.Map
}

func NewEngine(logger *slog.Logger, storage *storage.Storage) *TachyonEngine {
	return &TachyonEngine{
		logger:  logger,
		storage: storage,
	}
}

func (e *TachyonEngine) SetContext(ctx context.Context) {
	e.ctx = ctx
}

func (e *TachyonEngine) GetHistory() ([]storage.Task, error) {
	return e.storage.GetAllTasks()
}

func (e *TachyonEngine) StartDownload(urlStr string, destPath string) (string, error) {
	// 1. Validation
	if urlStr == "" {
		return "", fmt.Errorf("empty URL")
	}

	// 2. Client Initialization
	client := grab.NewClient()
	client.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	req, err := grab.NewRequest(destPath, urlStr)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// 3. Optimization: "Hydra" Mode
	// req.NoOfWorkers = 32 // FIXME: Verify grab v3 API for this.

	// Generate Download ID
	downloadID := uuid.New().String()

	// Create Initial Task Record
	task := storage.Task{
		ID:        downloadID,
		URL:       urlStr,
		Filename:  filepath.Base(destPath), // Initial guess, grab might update it
		Path:      destPath,
		Status:    "downloading",
		Progress:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := e.storage.SaveTask(task); err != nil {
		e.logger.Error("Failed to save initial task", "error", err)
		// Proceed anyway? Ideally yes, but let's log error.
	}

	// 4. Async Execution
	go e.processDownload(client, req, downloadID, task)

	return downloadID, nil
}

func (e *TachyonEngine) processDownload(client *grab.Client, req *grab.Request, id string, task storage.Task) {
	e.logger.Info("Download Started", "id", id, "url", req.URL().String(), "dest", req.Filename)

	resp := client.Do(req)
	e.activeDownloads.Store(id, resp)
	defer e.activeDownloads.Delete(id)

	// Update Filename if grab resolved a better one (e.g. from Content-Disposition)
	if resp.Filename != "" {
		task.Filename = filepath.Base(resp.Filename)
		task.Path = resp.Filename
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
			// Persist state
			task.Progress = resp.Progress() * 100
			task.Size = resp.Size()
			task.Status = "downloading"
			if err := e.storage.SaveTask(task); err != nil {
				e.logger.Error("Failed to save task state", "id", id, "error", err)
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
