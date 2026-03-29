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

// activeDownloadInfo stores control structures for a running download
type activeDownloadInfo struct {
	Cancel context.CancelFunc
	Wait   *sync.WaitGroup
}

// queueWorker is the background worker that dispatches tasks from the queue
func (e *TachyonEngine) queueWorker() {
	for {
		e.workerMutex.Lock()
		active := e.runningDownloads
		max := e.maxConcurrent
		e.workerMutex.Unlock()

		task := e.scheduler.GetNextTask(active, max)

		if task == nil {
			e.queue.Wait()
			continue
		}

		e.workerMutex.Lock()
		e.runningDownloads++
		e.workerMutex.Unlock()

		e.scheduler.OnTaskStarted(task)

		go func(t *storage.DownloadTask) {
			defer func() {
				if r := recover(); r != nil {
					e.logger.Error("Worker Panic Recovered", "id", t.ID, "panic", r)
					e.failTask(t, fmt.Sprintf("Internal Worker Error: %v", r))
				}

				e.workerMutex.Lock()
				e.runningDownloads--
				e.workerMutex.Unlock()

				e.scheduler.OnTaskCompleted(t)
			}()
			e.executeTask(t)
		}(task)
	}
}

// executeTask is the core download orchestration function
func (e *TachyonEngine) executeTask(task *storage.DownloadTask) {
	e.logger.Info("Starting Hyper-Engine Execution", "id", task.ID, "url", task.URL)
	startedAt := time.Now()

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
	task.Status = "probing"
	if e.ctx != nil {
		runtime.EventsEmit(e.ctx, "download:progress", map[string]interface{}{
			"id":       task.ID,
			"status":   "probing",
			"filename": task.Filename,
		})
	}
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

	isH2 := probe.IsHTTP2

	// 3. Prepare temp directory for part files
	tempDir := tempDirForTask(task.SavePath)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		e.failTask(task, fmt.Sprintf("Failed to create temp dir: %v", err))
		return
	}

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

	validationHeaders := map[string]string{
		"ETag":          probe.ETag,
		"Last-Modified": probe.LastModified,
	}

	if !e.stateManager.Validate(resumeState, validationHeaders) {
		e.logger.Info("Resume state invalid/mismatch, starting fresh", "id", task.ID)
		resumeState = nil
		task.Downloaded = 0
		task.Progress = 0
		cleanupPartFiles(tempDir, task.ID)
	} else if resumeState != nil {
		e.logger.Info("Resuming download", "id", task.ID, "parts_done", len(resumeState.Parts))
	}

	// Hydrate completed parts — validate against temp files on disk
	completedParts := make(map[int]bool)
	partPlan := make(map[int]DownloadPart, len(parts))
	for _, part := range parts {
		partPlan[part.ID] = part
	}
	if resumeState != nil {
		for id, ps := range resumeState.Parts {
			if !ps.Complete {
				continue
			}
			part, ok := partPlan[id]
			if !ok {
				continue
			}
			expectedSize := part.EndOffset - part.StartOffset + 1
			if part.EndOffset == StreamEndOffset {
				// Can't validate size for streaming parts
				completedParts[id] = true
			} else if partFileExists(tempDir, task.ID, id, expectedSize) {
				completedParts[id] = true
			}
		}
	}

	// Channels
	partCh := make(chan DownloadPart, numParts)
	retryCh := make(chan DownloadPart, numParts)
	partDoneCh := make(chan int, numParts)
	errCh := make(chan error, numParts*2)

	inflight := newInflightTracker()
	var nextStealID atomic.Int32
	nextStealID.Store(int32(numParts))

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

	workerCount := e.selectWorkerCountH2(host, numParts, probe.AcceptRanges, isH2)
	strictRanges := probe.AcceptRanges && workerCount > 1

	if workerCount > 1 {
		go e.WarmUpHost(host, workerCount/2)
	}

	var activeWorkers atomic.Int32
	activeWorkers.Store(int32(workerCount))
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		e.workerPool.Submit(func() {
			defer wg.Done()
			e.downloadWorker(ctx, task.ID, task.URL, host, tempDir, partCh, retryCh, partDoneCh, errCh, &downloadedBytes, &errorCount, task.Headers, task.Cookies, strictRanges, inflight, &nextStealID)
		})
	}

	// 6. Monitor Progress
	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	scaleTicker := time.NewTicker(5 * time.Second)
	defer scaleTicker.Stop()

	var lastDownloadedBytes int64 = atomic.LoadInt64(&downloadedBytes)
	lastTick := time.Now()
	var ewmaSpeed float64

	// Initial Status Update — save once at start
	task.Status = "downloading"
	e.storage.SaveTask(*task)

	if e.ctx != nil {
		runtime.EventsEmit(e.ctx, "download:progress", map[string]interface{}{
			"id":            task.ID,
			"status":        "downloading",
			"filename":      task.Filename,
			"total":         task.TotalSize,
			"accept_ranges": probe.AcceptRanges,
			"category":      task.Category,
			"started_at":    startedAt.Format(time.RFC3339),
			"path":          task.SavePath,
		})
	}

	e.bandwidthManager.SetTaskPriority(task.ID, task.Priority)
	e.bandwidthManager.MarkActive(task.ID)
	defer e.bandwidthManager.MarkInactive(task.ID)

Loop:
	for {
		select {
		case <-ctx.Done():
			metaSnap := e.serializeState(task, completedParts, partPlan)
			e.storage.SaveTaskAtomic(task.ID, func(t *storage.DownloadTask) {
				t.Status = "paused"
				t.MetaJSON = metaSnap
				t.Downloaded = atomic.LoadInt64(&downloadedBytes)
				t.Speed = 0
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
				cleanupPartFiles(tempDir, task.ID)
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

			if errors.Is(err, ErrStallTimeout) {
				metaSnap := e.serializeState(task, completedParts, partPlan)
				e.storage.SaveTaskAtomic(task.ID, func(t *storage.DownloadTask) {
					t.Status = "error"
					t.MetaJSON = metaSnap
				})
				e.failTask(task, "Download timed out: server not responding for 30 seconds")
				cancel()
				if e.ctx != nil {
					runtime.EventsEmit(e.ctx, "download:timeout", map[string]interface{}{
						"id":     task.ID,
						"reason": "Server not responding for 30 seconds",
					})
				}
				return
			}

			e.failTask(task, fmt.Sprintf("Critical error: %v", err))
			cancel()
			return

		case id := <-partDoneCh:
			completedParts[id] = true
			if len(completedParts) == numParts {
				break Loop
			}

		case <-ticker.C:
			// Update in-memory stats only — no DB save
			current := atomic.LoadInt64(&downloadedBytes)
			task.Downloaded = current
			if task.TotalSize > 0 {
				task.Progress = (float64(current) / float64(task.TotalSize)) * 100
			}

			now := time.Now()
			duration := now.Sub(lastTick).Seconds()
			if duration > 0 {
				bytesDiff := current - lastDownloadedBytes
				instantSpeed := float64(bytesDiff) / duration

				if ewmaSpeed == 0 {
					ewmaSpeed = instantSpeed
				} else {
					ewmaSpeed = 0.7*ewmaSpeed + 0.3*instantSpeed
				}
				task.Speed = ewmaSpeed

				e.stats.UpdateDownloadSpeed(int64(ewmaSpeed))

				lastDownloadedBytes = current
				lastTick = now

				if ewmaSpeed > 0 {
					remainingBytes := task.TotalSize - current
					etaSeconds := float64(remainingBytes) / ewmaSpeed
					task.TimeRemaining = fmt.Sprintf("%.0fs", etaSeconds)
				}
			}

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
			if strictRanges {
				ideal := int32(e.selectWorkerCountH2(host, numParts-len(completedParts), true, isH2))
				current := activeWorkers.Load()
				if ideal > current {
					toSpawn := ideal - current
					activeWorkers.Store(ideal)
					for i := int32(0); i < toSpawn; i++ {
						wg.Add(1)
						e.workerPool.Submit(func() {
							defer wg.Done()
							e.downloadWorker(ctx, task.ID, task.URL, host, tempDir, partCh, retryCh, partDoneCh, errCh, &downloadedBytes, &errorCount, task.Headers, task.Cookies, strictRanges, inflight, &nextStealID)
						})
					}
					e.logger.Info("Scaled up workers", "id", task.ID, "from", current, "to", ideal)
				} else if ideal < current && ideal >= 1 {
					activeWorkers.Store(ideal)
					e.logger.Info("Scaled down workers target", "id", task.ID, "from", current, "to", ideal)
				}
			}
		}
	}

	// 7. Merge & Verify
	if len(completedParts) == numParts {
		task.Status = "merging"
		if e.ctx != nil {
			runtime.EventsEmit(e.ctx, "download:progress", map[string]interface{}{
				"id":     task.ID,
				"status": "merging",
			})
		}

		e.logger.Info("Merging part files", "id", task.ID, "parts", numParts)
		if err := mergePartFiles(tempDir, task.ID, task.SavePath); err != nil {
			e.failTask(task, fmt.Sprintf("Merge failed: %v", err))
			return
		}

		// Clean up temp dir if empty
		os.Remove(tempDir)

		task.Status = "verifying"
		e.storage.SaveTask(*task)
		if e.ctx != nil {
			runtime.EventsEmit(e.ctx, "download:progress", map[string]interface{}{
				"id":     task.ID,
				"status": "verifying",
			})
		}

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

		e.stats.TrackFileCompleted()
		e.stats.TrackDownloadBytes(task.TotalSize)

		completedAt := time.Now()
		elapsed := completedAt.Sub(startedAt).Seconds()
		var avgSpeed float64
		if elapsed > 0 {
			avgSpeed = float64(task.TotalSize) / elapsed
		}

		if e.ctx != nil {
			runtime.EventsEmit(e.ctx, "download:completed", map[string]interface{}{
				"id":           task.ID,
				"path":         task.SavePath,
				"completed_at": completedAt.Format(time.RFC3339),
				"started_at":   startedAt.Format(time.RFC3339),
				"elapsed":      elapsed,
				"avg_speed":    avgSpeed,
			})
		}
	}
}
