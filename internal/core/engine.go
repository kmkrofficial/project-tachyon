package core

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/cavaliergopher/grab/v3"
	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type TachyonEngine struct {
	logger *slog.Logger
	ctx    context.Context
	// ActiveDownloads maps DownloadID (string) to *grab.Response
	activeDownloads sync.Map
}

func NewEngine(logger *slog.Logger) *TachyonEngine {
	return &TachyonEngine{
		logger: logger,
	}
}

func (e *TachyonEngine) SetContext(ctx context.Context) {
	e.ctx = ctx
}

func (e *TachyonEngine) StartDownload(urlStr string, destPath string) (string, error) {
	// 1. Validation
	if urlStr == "" {
		return "", fmt.Errorf("empty URL")
	}

	// 2. Client Initialization
	client := grab.NewClient()
	// Set UserAgent to avoid 403s
	client.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	req, err := grab.NewRequest(destPath, urlStr)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// 3. Optimization: "Hydra" Mode (32 workers)
	// req.NoOfWorkers = 32 // FIXME: Verify grab v3 API for this.

	// Generate Download ID
	downloadID := uuid.New().String()

	// 4. Async Execution
	go e.processDownload(client, req, downloadID)

	return downloadID, nil
}

func (e *TachyonEngine) processDownload(client *grab.Client, req *grab.Request, id string) {
	e.logger.Info("Download Started", "id", id, "url", req.URL().String(), "dest", req.Filename)

	resp := client.Do(req)
	e.activeDownloads.Store(id, resp)
	defer e.activeDownloads.Delete(id)

	// Ticker for progress monitoring
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

Loop:
	for {
		select {
		case <-ticker.C:
			// Calculate standard metrics
			progress := resp.Progress() * 100
			speedMBs := float64(resp.BytesPerSecond()) / 1024 / 1024

			// Broadcast progress via Wails Events
			if e.ctx != nil {
				runtime.EventsEmit(e.ctx, "download:progress", map[string]interface{}{
					"id":        id,
					"progress":  progress,
					"speed_MBs": speedMBs,
					"eta":       resp.ETA().String(),
					"filename":  resp.Filename,
				})
			}

			// Also log it
			e.logger.Info("Download Progress",
				"id", id,
				"progress", fmt.Sprintf("%.2f%%", progress),
				"speed_MBs", speedMBs,
			)

			if resp.IsComplete() {
				break Loop
			}
		case <-resp.Done:
			// Channel closed when download finishes or cancels
			break Loop
		}
	}

	// Final status check
	if err := resp.Err(); err != nil {
		e.logger.Error("Download Failed", "id", id, "error", err.Error())
		if e.ctx != nil {
			runtime.EventsEmit(e.ctx, "download:failed", map[string]interface{}{
				"id":    id,
				"error": err.Error(),
			})
		}
	} else {
		e.logger.Info("Download Completed", "id", id, "path", resp.Filename)
		if e.ctx != nil {
			runtime.EventsEmit(e.ctx, "download:completed", map[string]interface{}{
				"id":   id,
				"path": resp.Filename,
			})
		}
	}
}
