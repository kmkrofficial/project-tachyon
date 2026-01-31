package engine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"

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

// activeDownloadInfo stores control structures for a running download
type activeDownloadInfo struct {
	Cancel context.CancelFunc
	Wait   *sync.WaitGroup
}

// downloadWorker consumes parts and downloads them
func (e *TachyonEngine) downloadWorker(ctx context.Context, taskID string, urlStr string, host string, file *os.File, partCh <-chan DownloadPart, retryCh chan DownloadPart, partDoneCh chan<- int, errCh chan<- error, downloadedBytes *int64, errorCount *atomic.Int32, headersStr string, cookiesStr string) {
	for {
		select {
		case <-ctx.Done():
			return
		case part, ok := <-partCh:
			if !ok {
				return
			}
			e.processDownloadPart(ctx, taskID, urlStr, host, file, part, retryCh, partDoneCh, errCh, downloadedBytes, errorCount, headersStr, cookiesStr)
		}
	}
}

// processDownloadPart handles downloading a single part with retry logic
func (e *TachyonEngine) processDownloadPart(ctx context.Context, taskID string, urlStr string, host string, file *os.File, part DownloadPart, retryCh chan DownloadPart, partDoneCh chan<- int, errCh chan<- error, downloadedBytes *int64, errorCount *atomic.Int32, headersStr string, cookiesStr string) {
	err := e.downloadPart(ctx, taskID, urlStr, file, part, BufferSize, headersStr, cookiesStr)
	if err != nil {
		// Record error for congestion control
		e.congestionController.RecordOutcome(host, 0, err)
		errorCount.Add(1)

		// Check for ErrLinkExpired (403)
		if err == ErrLinkExpired {
			e.logger.Warn("Link expired (403), task needs URL refresh", "id", taskID)
			errCh <- ErrLinkExpired
			return
		}

		// Retry Logic
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
			// Fatal
			e.logger.Error("Part exceeded max retries", "id", part.ID)
			errCh <- fmt.Errorf("Part %d run out of attempts", part.ID)
			return
		}
	} else {
		// Success
		e.congestionController.RecordOutcome(host, 0, nil)
		atomic.AddInt64(downloadedBytes, part.EndOffset-part.StartOffset+1)
		partDoneCh <- part.ID
	}
}

// downloadPart downloads a single part of the file
func (e *TachyonEngine) downloadPart(ctx context.Context, taskID string, urlStr string, file *os.File, part DownloadPart, chunkSize int, headersStr string, cookiesStr string) error {
	req, err := e.newRequest("GET", urlStr, headersStr, cookiesStr)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", part.StartOffset, part.EndOffset))

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		// Check for 403 Forbidden - indicates expired/invalid link
		if resp.StatusCode == http.StatusForbidden {
			return ErrLinkExpired
		}
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	bufPtr := e.bufferPool.Get().(*[]byte)
	defer e.bufferPool.Put(bufPtr)
	buf := *bufPtr

	currentOffset := part.StartOffset
	totalBytesToRead := part.EndOffset - part.StartOffset + 1
	bytesReadTotal := int64(0)

	for bytesReadTotal < totalBytesToRead {
		// 1. Traffic Shaping
		if err := e.bandwidthManager.Wait(ctx, taskID, chunkSize); err != nil {
			return err
		}

		// 2. Network Read
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := file.WriteAt(buf[:n], currentOffset)
			if writeErr != nil {
				return writeErr
			}
			currentOffset += int64(n)
			bytesReadTotal += int64(n)
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return readErr
		}
	}

	return nil
}

// failTask marks a task as failed
func (e *TachyonEngine) failTask(task *storage.DownloadTask, reason string) {
	e.logger.Error("Task Failed", "id", task.ID, "reason", reason)
	task.Status = "error"
	e.storage.SaveTask(*task)
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
func (e *TachyonEngine) serializeState(task *storage.DownloadTask, completedParts map[int]bool) string {
	// Construct ResumeState from current execution status
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
			state.Parts[id] = storage.PartState{
				Start:    int64(id) * DownloadChunkSize,
				End:      int64(id)*DownloadChunkSize + DownloadChunkSize - 1,
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
