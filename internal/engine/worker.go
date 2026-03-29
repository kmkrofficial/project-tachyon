package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"project-tachyon/internal/storage"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// DownloadPart represents a single unit of work
type DownloadPart struct {
	ID          int   // Sequence ID
	StartOffset int64 // Byte Start (Inclusive)
	EndOffset   int64 // Byte End (Inclusive)
	Attempts    int   // Retry count
}

// downloadWorker consumes parts and downloads them to individual temp files.
func (e *TachyonEngine) downloadWorker(ctx context.Context, taskID string, urlStr string, host string, tempDir string, partCh <-chan DownloadPart, retryCh chan DownloadPart, partDoneCh chan<- int, errCh chan<- error, downloadedBytes *int64, errorCount *atomic.Int32, headersStr string, cookiesStr string, strictRanges bool, inflight *inflightTracker, nextStealID *atomic.Int32) {
	partChOpen := true
	for {
		if ctx.Err() != nil {
			return
		}

		// Phase 1: consume from primary channel and retries
		if partChOpen {
			select {
			case <-ctx.Done():
				return
			case part, ok := <-retryCh:
				if ok {
					e.processDownloadPart(ctx, taskID, urlStr, host, tempDir, part, retryCh, partDoneCh, errCh, downloadedBytes, errorCount, headersStr, cookiesStr, strictRanges, inflight)
					continue
				}
			case part, ok := <-partCh:
				if !ok {
					partChOpen = false
					continue // switch to phase 2
				}
				e.processDownloadPart(ctx, taskID, urlStr, host, tempDir, part, retryCh, partDoneCh, errCh, downloadedBytes, errorCount, headersStr, cookiesStr, strictRanges, inflight)
				continue
			}
		}

		// Phase 2: primary channel drained — drain retries, then try stealing
		select {
		case <-ctx.Done():
			return
		case rp := <-retryCh:
			e.processDownloadPart(ctx, taskID, urlStr, host, tempDir, rp, retryCh, partDoneCh, errCh, downloadedBytes, errorCount, headersStr, cookiesStr, strictRanges, inflight)
			continue
		case <-time.After(50 * time.Millisecond):
			// Brief wait for pending retries before trying to steal or exit
		}

		if strictRanges {
			stolen, _ := inflight.StealLargest(int(nextStealID.Add(1) - 1))
			if stolen != nil {
				e.processDownloadPart(ctx, taskID, urlStr, host, tempDir, *stolen, retryCh, partDoneCh, errCh, downloadedBytes, errorCount, headersStr, cookiesStr, strictRanges, inflight)
				continue
			}
		}
		return
	}
}

// processDownloadPart handles downloading a single part with retry logic
func (e *TachyonEngine) processDownloadPart(ctx context.Context, taskID string, urlStr string, host string, tempDir string, part DownloadPart, retryCh chan DownloadPart, partDoneCh chan<- int, errCh chan<- error, downloadedBytes *int64, errorCount *atomic.Int32, headersStr string, cookiesStr string, strictRanges bool, inflight *inflightTracker) {
	inflight.Start(part)
	defer inflight.Complete(part.ID)

	if err := e.breaker.Allow(host); err != nil {
		if part.Attempts < 3 {
			part.Attempts++
			select {
			case retryCh <- part:
			default:
				errCh <- fmt.Errorf("breaker open, retry buffer full for part %d", part.ID)
			}
		} else {
			errCh <- fmt.Errorf("breaker open for host %s, part %d exhausted retries", host, part.ID)
		}
		return
	}

	startedAt := time.Now()
	err := e.downloadPart(ctx, taskID, urlStr, tempDir, part, BufferSize, headersStr, cookiesStr, strictRanges, downloadedBytes, inflight)
	e.congestion.RecordOutcome(host, time.Since(startedAt), err)

	if err != nil {
		e.breaker.RecordFailure(host)
		errorCount.Add(1)

		if errors.Is(err, ErrRangeIgnored) {
			errCh <- ErrRangeIgnored
			return
		}

		if err == ErrLinkExpired {
			e.logger.Warn("Link expired (403), task needs URL refresh", "id", taskID)
			errCh <- ErrLinkExpired
			return
		}

		if errors.Is(err, ErrStallTimeout) {
			e.logger.Error("Download stalled (30s timeout)", "id", taskID, "part", part.ID)
			errCh <- ErrStallTimeout
			return
		}

		if part.Attempts < 3 {
			part.Attempts++
			e.logger.Warn("Retrying part", "id", part.ID, "attempt", part.Attempts)
			select {
			case retryCh <- part:
			default:
				e.logger.Error("Retry buffer full, dropping part (critical)", "id", part.ID)
				errCh <- fmt.Errorf("Retry buffer full")
				return
			}
		} else {
			e.logger.Error("Part exceeded max retries", "id", part.ID)
			errCh <- fmt.Errorf("Part %d run out of attempts", part.ID)
			return
		}
	} else {
		e.breaker.RecordSuccess(host)
		partDoneCh <- part.ID
	}
}

// ErrStallTimeout is returned when a download stalls for too long without receiving data.
var ErrStallTimeout = fmt.Errorf("download stalled: no data received")

const (
	minStallTimeout = 5 * time.Second
	maxStallTimeout = 30 * time.Second
)

func adaptiveStallTimeout(recentBytesPerSec float64, bufSize int) time.Duration {
	if recentBytesPerSec <= 0 {
		return maxStallTimeout
	}
	expected := float64(bufSize) / recentBytesPerSec
	timeout := time.Duration(expected*3) * time.Second
	if timeout < minStallTimeout {
		return minStallTimeout
	}
	if timeout > maxStallTimeout {
		return maxStallTimeout
	}
	return timeout
}

// downloadPart downloads a single part into its own temp file.
func (e *TachyonEngine) downloadPart(ctx context.Context, taskID string, urlStr string, tempDir string, part DownloadPart, chunkSize int, headersStr string, cookiesStr string, strictRanges bool, downloadedBytes *int64, inflight *inflightTracker) error {
	req, err := e.newRequest("GET", urlStr, headersStr, cookiesStr)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	if part.EndOffset != StreamEndOffset {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", part.StartOffset, part.EndOffset))
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if strictRanges && part.EndOffset != StreamEndOffset && resp.StatusCode == http.StatusOK {
		return ErrRangeIgnored
	}

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden {
			return ErrLinkExpired
		}
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Create temp file for this part
	pw, err := newPartWriter(tempDir, taskID, part.StartOffset, downloadedBytes)
	if err != nil {
		return err
	}
	defer pw.Close()

	bufPtr := e.bufferPool.Get().(*[]byte)
	defer e.bufferPool.Put(bufPtr)
	buf := *bufPtr

	totalBytesToRead := part.EndOffset - part.StartOffset + 1
	if part.EndOffset == StreamEndOffset {
		totalBytesToRead = StreamEndOffset
	}
	bytesReadTotal := int64(0)

	// Adaptive stall timeout state
	var recentSpeed float64
	lastSpeedCheck := time.Now()
	lastSpeedBytes := int64(0)

	// Deadline-based stall detection using a timer
	stallTimer := time.NewTimer(maxStallTimeout)
	defer stallTimer.Stop()

	type readResult struct {
		n   int
		err error
	}
	readCh := make(chan readResult, 1)

	for bytesReadTotal < totalBytesToRead {
		stall := adaptiveStallTimeout(recentSpeed, chunkSize)

		go func() {
			n, readErr := resp.Body.Read(buf)
			readCh <- readResult{n, readErr}
		}()

		if !stallTimer.Stop() {
			select {
			case <-stallTimer.C:
			default:
			}
		}
		stallTimer.Reset(stall)

		var rr readResult
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-stallTimer.C:
			return ErrStallTimeout
		case rr = <-readCh:
		}

		if rr.n > 0 {
			if err := e.bandwidthManager.Wait(ctx, taskID, rr.n); err != nil {
				return err
			}

			// Check if this part was stolen (EndOffset reduced by work-stealing).
			// Only write up to the adjusted boundary to avoid overlap.
			writeN := rr.n
			if adj := inflight.AdjustedEnd(part.ID); adj >= 0 {
				newLimit := adj - part.StartOffset + 1
				if bytesReadTotal+int64(writeN) > newLimit {
					writeN = int(newLimit - bytesReadTotal)
					if writeN <= 0 {
						break // Nothing more to write for this part
					}
				}
				totalBytesToRead = newLimit
			}

			if writeErr := pw.Write(buf[:writeN]); writeErr != nil {
				return writeErr
			}
			bytesReadTotal += int64(writeN)

			lastSpeedBytes += int64(writeN)
			elapsed := time.Since(lastSpeedCheck).Seconds()
			if elapsed >= 1.0 {
				recentSpeed = float64(lastSpeedBytes) / elapsed
				lastSpeedBytes = 0
				lastSpeedCheck = time.Now()
			}
		}
		if rr.err != nil {
			if rr.err == io.EOF {
				break
			}
			return rr.err
		}
	}

	return nil
}

// failTask marks a task as failed
func (e *TachyonEngine) failTask(task *storage.DownloadTask, reason string) {
	e.logger.Error("Task Failed", "id", task.ID, "reason", reason)
	task.Status = "error"
	e.storage.SaveTaskAtomic(task.ID, func(t *storage.DownloadTask) {
		t.Status = "error"
	})
	if e.ctx != nil {
		runtime.EventsEmit(e.ctx, "download:error", map[string]interface{}{
			"id":    task.ID,
			"error": reason,
		})
	}
}

// loadState deserializes download state from MetaJSON
func (e *TachyonEngine) loadState(metaJSON string) (*storage.ResumeState, error) {
	return e.stateManager.Load(metaJSON)
}

// serializeState serializes download state to MetaJSON
func (e *TachyonEngine) serializeState(task *storage.DownloadTask, completedParts map[int]bool, partPlan map[int]DownloadPart) string {
	state := &storage.ResumeState{
		Version:      1,
		ETag:         "",
		LastModified: "",
		TotalSize:    task.TotalSize,
		Parts:        make(map[int]storage.PartState),
	}

	// Track completed parts
	for id, done := range completedParts {
		if done {
			part, ok := partPlan[id]
			if !ok {
				continue
			}
			state.Parts[id] = storage.PartState{
				Start:    part.StartOffset,
				End:      part.EndOffset,
				Complete: true,
			}
		}
	}

	str, err := e.stateManager.Serialize(state)
	if err != nil {
		e.logger.Error("Failed to serialize state", "error", err)
		return ""
	}
	return str
}
