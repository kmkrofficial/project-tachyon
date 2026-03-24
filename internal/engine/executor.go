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

	u, _ := url.Parse(task.URL)
	host := u.Hostname()
	if e.isHostSingleStream(host) {
		probe.AcceptRanges = false
	}

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
	parts := e.planDownloadParts(probe.Size, probe.AcceptRanges)
	numParts := len(parts)
	if !probe.AcceptRanges {
		e.logger.Info("Server does not support ranges, switching to single-threaded mode", "id", task.ID)
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
	partPlan := make(map[int]DownloadPart, len(parts))
	for _, part := range parts {
		partPlan[part.ID] = part
	}
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
		for _, part := range parts {
			if completedParts[part.ID] {
				continue
			}
			partCh <- part
		}
		close(partCh)
	}()

	// 5. Worker Swarm (Consumers)
	var errorCount atomic.Int32

	wg := &sync.WaitGroup{}

	// Initialize downloadedBytes based on completed parts
	var initialBytes int64
	for id, done := range completedParts {
		if !done {
			continue
		}
		part, ok := partPlan[id]
		if !ok {
			continue
		}
		if part.EndOffset == StreamEndOffset {
			continue
		}
		initialBytes += (part.EndOffset - part.StartOffset + 1)
	}

	var downloadedBytes int64 = initialBytes

	// Spawn initial workers via global pool (dynamic scaling adjusts this later)
	workerCount := e.selectWorkerCount(host, numParts, probe.AcceptRanges)
	strictRanges := probe.AcceptRanges && workerCount > 1
	var activeWorkers atomic.Int32
	activeWorkers.Store(int32(workerCount))
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		e.workerPool.Submit(func() {
			defer wg.Done()
			e.downloadWorker(ctx, task.ID, task.URL, host, file, partCh, retryCh, partDoneCh, errCh, &downloadedBytes, &errorCount, task.Headers, task.Cookies, strictRanges)
		})
	}

	// 6. Monitor Progress & Waits
	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Dynamic scaling ticker — re-evaluate every 5 seconds
	scaleTicker := time.NewTicker(5 * time.Second)
	defer scaleTicker.Stop()

	// Speed calculation variables
	var lastDownloadedBytes int64 = atomic.LoadInt64(&downloadedBytes)
	lastTick := time.Now()

	// Initial Status Update
	task.Status = "downloading"
	e.storage.SaveTask(*task)

	// Notify frontend of status transition
	if e.ctx != nil {
		runtime.EventsEmit(e.ctx, "download:progress", map[string]interface{}{
			"id":       task.ID,
			"status":   "downloading",
			"filename": task.Filename,
			"total":    task.TotalSize,
		})
	}

	// Update Bandwidth Manager priority
	e.bandwidthManager.SetTaskPriority(task.ID, task.Priority)

Loop:
	for {
		select {
		case <-ctx.Done():
			// Cancelled by user — atomic status+meta update
			metaSnap := e.serializeState(task, completedParts, partPlan)
			e.storage.SaveTaskAtomic(task.ID, func(t *storage.DownloadTask) {
				t.Status = "paused"
				t.MetaJSON = metaSnap
			})
			task.Status = "paused"
			e.logger.Info("Download Cancelled/Paused", "id", task.ID)
			if e.ctx != nil {
				runtime.EventsEmit(e.ctx, "download:paused", map[string]interface{}{
					"id": task.ID,
				})
			}
			break Loop

		case err := <-errCh:
			if errors.Is(err, ErrRangeIgnored) {
				e.logger.Warn("Range ignored by host, downgrading to single-stream mode", "id", task.ID, "host", host)
				e.markHostSingleStream(host)
				if saveErr := e.storage.SaveTaskAtomic(task.ID, func(t *storage.DownloadTask) {
					t.Status = "pending"
					t.MetaJSON = ""
					t.Progress = 0
					t.Downloaded = 0
					t.Speed = 0
					t.TimeRemaining = ""
				}); saveErr != nil {
					e.failTask(task, fmt.Sprintf("Failed to save fallback state: %v", saveErr))
					cancel()
					return
				}
				task.Status = "pending"
				task.MetaJSON = ""
				task.Progress = 0
				task.Downloaded = 0
				task.Speed = 0
				task.TimeRemaining = ""
				e.queue.Push(task)
				cancel()
				return
			}

			// Check for link expiration (403 Forbidden)
			if errors.Is(err, ErrLinkExpired) {
				e.logger.Warn("Link expired - pausing for URL refresh", "id", task.ID)
				metaSnap := e.serializeState(task, completedParts, partPlan)
				e.storage.SaveTaskAtomic(task.ID, func(t *storage.DownloadTask) {
					t.Status = StatusNeedsAuth
					t.MetaJSON = metaSnap
				})
				task.Status = StatusNeedsAuth
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
					"status":     task.Status,
					"progress":   task.Progress,
					"speed":      task.Speed,
					"eta":        task.TimeRemaining,
					"downloaded": task.Downloaded,
					"total":      task.TotalSize,
				})
			}

		case <-scaleTicker.C:
			// Dynamic worker scaling: query congestion controller for ideal count
			if strictRanges {
				ideal := int32(e.selectWorkerCount(host, numParts-len(completedParts), true))
				current := activeWorkers.Load()
				if ideal > current {
					toSpawn := ideal - current
					activeWorkers.Store(ideal)
					for i := int32(0); i < toSpawn; i++ {
						wg.Add(1)
						e.workerPool.Submit(func() {
							defer wg.Done()
							e.downloadWorker(ctx, task.ID, task.URL, host, file, partCh, retryCh, partDoneCh, errCh, &downloadedBytes, &errorCount, task.Headers, task.Cookies, strictRanges)
						})
					}
					e.logger.Info("Scaled up workers", "id", task.ID, "from", current, "to", ideal)
				}
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
		e.storage.SaveTaskAtomic(task.ID, func(t *storage.DownloadTask) {
			t.Status = "completed"
			t.Progress = 100
			t.Downloaded = task.TotalSize
		})
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
