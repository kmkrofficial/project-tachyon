package engine

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"project-tachyon/internal/storage"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// queueWorker is the background worker that dispatches tasks from the queue
func (e *TachyonEngine) queueWorker() {
	for {
		// Update Concurrency Snapshot
		e.workerMutex.Lock()
		active := e.runningDownloads
		max := e.maxConcurrent
		e.workerMutex.Unlock()

		// Smart Schedule Dispatch
		task := e.scheduler.GetNextTask(active, max)

		if task == nil {
			// Wait for new tasks or slot availability
			e.queue.Wait()
			continue
		}

		// Reserve Slot
		e.workerMutex.Lock()
		e.runningDownloads++
		e.workerMutex.Unlock()

		// Notify Scheduler
		e.scheduler.OnTaskStarted(task)

		go func(t *storage.DownloadTask) {
			defer func() {
				// Panic Recovery
				if r := recover(); r != nil {
					e.logger.Error("Worker Panic Recovered", "id", t.ID, "panic", r)
					e.failTask(t, fmt.Sprintf("Internal Worker Error: %v", r))
				}

				e.workerMutex.Lock()
				e.runningDownloads--
				e.workerMutex.Unlock()

				// Notify Scheduler & Wake Queue
				e.scheduler.OnTaskCompleted(t)
			}()
			e.executeTask(t)
		}(task)
	}
}

// executeTask is the core download orchestration function
func (e *TachyonEngine) executeTask(task *storage.DownloadTask) {
	e.logger.Info("Starting Hyper-Engine Execution", "id", task.ID, "url", task.URL)

	// 1. Setup Context for Cancellation
	parentCtx := e.ctx
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx, cancel := context.WithCancel(parentCtx)
	e.activeDownloads.Store(task.ID, &activeDownloadInfo{
		Cancel: cancel,
		Wait:   &sync.WaitGroup{},
	})
	defer e.activeDownloads.Delete(task.ID)

	// 2. Probe & Validate
	probe, err := e.ProbeURL(task.URL, task.Headers, task.Cookies)
	if err != nil {
		e.failTask(task, fmt.Sprintf("Probe failed: %v", err))
		return
	}
	task.TotalSize = probe.Size

	// 3. Prepare Disk (Allocator)
	if probe.Size > 0 {
		if err := e.allocator.AllocateFile(task.SavePath, probe.Size); err != nil {
			e.failTask(task, fmt.Sprintf("Allocation failed: %v", err))
			return
		}
	} else {
		f, err := os.OpenFile(task.SavePath, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			e.failTask(task, fmt.Sprintf("File creation failed: %v", err))
			return
		}
		f.Close()
	}

	// Re-open file for sharing with workers
	file, err := os.OpenFile(task.SavePath, os.O_RDWR, 0666)
	if err != nil {
		e.failTask(task, fmt.Sprintf("File open failed: %v", err))
		return
	}
	defer file.Close()

	// 4. Job Producer (Generate Parts)
	numParts := int((probe.Size + DownloadChunkSize - 1) / DownloadChunkSize)
	if numParts == 0 {
		numParts = 1
	}

	// Fallback to Single-Threaded if Ranges not supported
	if !probe.AcceptRanges {
		e.logger.Info("Server does not support ranges, switching to single-threaded mode", "id", task.ID)
		numParts = 1
	}

	// Load Resume State
	resumeState, err := e.loadState(task.MetaJSON)
	if err != nil {
		e.logger.Warn("Failed to parse resume state", "error", err)
		resumeState = nil
	}

	// Validate Resume
	validationHeaders := map[string]string{
		"ETag":          probe.ETag,
		"Last-Modified": probe.LastModified,
	}

	if !e.stateManager.Validate(resumeState, validationHeaders) {
		e.logger.Info("Resume state invalid/mismatch, starting fresh", "id", task.ID)
		resumeState = nil
		task.Downloaded = 0
		task.Progress = 0
	} else if resumeState != nil {
		e.logger.Info("Resuming download", "id", task.ID, "parts_done", len(resumeState.Parts))
	}

	// Hydrate completed parts
	completedParts := make(map[int]bool)
	if resumeState != nil {
		for id, part := range resumeState.Parts {
			if part.Complete {
				completedParts[id] = true
			}
		}
	}

	// Channels
	partCh := make(chan DownloadPart, numParts)
	retryCh := make(chan DownloadPart, numParts)
	partDoneCh := make(chan int, numParts)
	errCh := make(chan error, numParts*2)

	// Generate parts (Skip completed)
	go func() {
		for i := 0; i < numParts; i++ {
			if completedParts[i] {
				continue
			}
			start := int64(i) * DownloadChunkSize
			end := start + DownloadChunkSize - 1
			if end >= probe.Size {
				end = probe.Size - 1
			}
			partCh <- DownloadPart{ID: i, StartOffset: start, EndOffset: end, Attempts: 0}
		}
		close(partCh)
	}()

	// 5. Worker Swarm (Consumers)
	currentConcurrency := 1
	maxTaskConcurrency := 32
	u, _ := url.Parse(task.URL)
	host := u.Host

	activeWorkers := 0

	// Channels for dynamic scaling
	scaleUpCh := make(chan struct{}, maxTaskConcurrency)
	_ = scaleUpCh // Avoid unused warning

	// Error tracking for Congestion Control
	var errorCount atomic.Int32

	wg := &sync.WaitGroup{}

	// Initialize downloadedBytes based on completed parts
	var initialBytes int64
	for id := range completedParts {
		start := int64(id) * DownloadChunkSize
		end := start + DownloadChunkSize - 1
		if end >= probe.Size {
			end = probe.Size - 1
		}
		initialBytes += (end - start + 1)
	}

	var downloadedBytes int64 = initialBytes

	// Helper to spawn workers
	spawnWorker := func() {
		wg.Add(1)
		activeWorkers++
		go func() {
			defer wg.Done()
			e.downloadWorker(ctx, task.ID, task.URL, host, file, partCh, retryCh, partDoneCh, errCh, &downloadedBytes, &errorCount, task.Headers, task.Cookies)
		}()
	}

	// Initial Spawn
	for i := 0; i < currentConcurrency; i++ {
		spawnWorker()
	}

	// 6. Monitor Progress & Waits
	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	ticker := time.NewTicker(200 * time.Millisecond)
	congestionTicker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	defer congestionTicker.Stop()

	// Speed calculation variables
	var lastDownloadedBytes int64 = atomic.LoadInt64(&downloadedBytes)
	lastTick := time.Now()

	// Initial Status Update
	task.Status = "downloading"
	e.storage.SaveTask(*task)

	// Update Bandwidth Manager priority
	e.bandwidthManager.SetTaskPriority(task.ID, task.Priority)

Loop:
	for {
		select {
		case <-ctx.Done():
			// Cancelled by user
			task.Status = "paused"
			task.MetaJSON = e.serializeState(task, completedParts)
			e.storage.SaveTask(*task)
			e.logger.Info("Download Cancelled/Paused", "id", task.ID)
			if e.ctx != nil {
				runtime.EventsEmit(e.ctx, "download:paused", map[string]interface{}{
					"id": task.ID,
				})
			}
			break Loop

		case err := <-errCh:
			// Check for link expiration (403 Forbidden)
			if errors.Is(err, ErrLinkExpired) {
				e.logger.Warn("Link expired - pausing for URL refresh", "id", task.ID)
				task.Status = StatusNeedsAuth
				task.MetaJSON = e.serializeState(task, completedParts)
				e.storage.SaveTask(*task)
				cancel()
				if e.ctx != nil {
					runtime.EventsEmit(e.ctx, "download:needs_auth", map[string]interface{}{
						"id":     task.ID,
						"reason": "Link expired (HTTP 403)",
					})
				}
				return
			}
			// Critical error reported
			e.failTask(task, fmt.Sprintf("Critical error: %v", err))
			cancel()
			return

		case id := <-partDoneCh:
			completedParts[id] = true
			if len(completedParts) == numParts {
				break Loop
			}

		case <-congestionTicker.C:
			// Congestion Control / Auto-Tuning
			ideal := e.congestionController.GetIdealConcurrency(host)
			maxTaskConcurrency = ideal

			// Scale Up Check
			if activeWorkers < maxTaskConcurrency {
				toAdd := maxTaskConcurrency - activeWorkers
				if toAdd > 2 {
					toAdd = 2
				}
				for i := 0; i < toAdd; i++ {
					spawnWorker()
				}
			}

		case <-ticker.C:
			// Update Stats
			current := atomic.LoadInt64(&downloadedBytes)
			task.Downloaded = current
			task.Progress = (float64(current) / float64(task.TotalSize)) * 100

			// Calculate Speed & ETA
			now := time.Now()
			duration := now.Sub(lastTick).Seconds()
			if duration > 0 {
				bytesDiff := current - lastDownloadedBytes
				speed := float64(bytesDiff) / duration
				task.Speed = speed

				// Update global stats
				e.stats.UpdateDownloadSpeed(int64(speed))

				lastDownloadedBytes = current
				lastTick = now

				if speed > 0 {
					remainingBytes := task.TotalSize - current
					etaSeconds := float64(remainingBytes) / speed
					task.TimeRemaining = fmt.Sprintf("%.0fs", etaSeconds)
				}
			}

			// Emit live progress event
			if e.ctx != nil {
				runtime.EventsEmit(e.ctx, "download:progress", map[string]interface{}{
					"id":         task.ID,
					"progress":   task.Progress,
					"speed":      task.Speed,
					"eta":        task.TimeRemaining,
					"downloaded": task.Downloaded,
					"total":      task.TotalSize,
				})
			}
		}
	}

	// Completion
	if len(completedParts) == numParts {
		// 7. Verify Integrity
		task.Status = "verifying"
		e.storage.SaveTask(*task)
		if e.ctx != nil {
			runtime.EventsEmit(e.ctx, "download:verifying", map[string]interface{}{"id": task.ID})
		}

		// Close file handle before verification
		file.Close()

		// Check if verification is enabled
		enabled := true
		s, err := e.storage.GetString("enable_integrity_check")
		if err == nil && s == "false" {
			enabled = false
		}

		if enabled && task.ExpectedHash != "" {
			e.logger.Info("Verifying integrity", "id", task.ID, "hash", task.ExpectedHash)
			if err := e.verifier.Verify(task.SavePath, task.HashAlgorithm, task.ExpectedHash); err != nil {
				e.failTask(task, fmt.Sprintf("Integrity Check Failed: %v", err))
				corruptedPath := task.SavePath + ".corrupted"
				os.Rename(task.SavePath, corruptedPath)
				return
			}
		}

		task.Status = "completed"
		task.Progress = 100
		task.Downloaded = task.TotalSize
		e.storage.SaveTask(*task)
		e.logger.Info("Download Completed", "id", task.ID)

		// Trigger native AV scan (non-blocking warning)
		if scanErr := e.scanner.ScanFile(ctx, task.SavePath); scanErr != nil {
			e.logger.Warn("AV scan warning", "id", task.ID, "error", scanErr)
			if e.ctx != nil {
				runtime.EventsEmit(e.ctx, "download:av_warning", map[string]interface{}{
					"id":      task.ID,
					"path":    task.SavePath,
					"warning": scanErr.Error(),
				})
			}
		}

		// Update stats
		e.stats.TrackFileCompleted()
		e.stats.TrackDownloadBytes(task.TotalSize)

		if e.ctx != nil {
			runtime.EventsEmit(e.ctx, "download:completed", map[string]interface{}{
				"id":   task.ID,
				"path": task.SavePath,
			})
		}
	}
}
