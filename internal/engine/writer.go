package engine

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
)

const partFileBufferSize = 1 * 1024 * 1024 // 1MB write buffer per part file

// partWriter owns a single temp file for one download part.
// Each worker writes sequentially to its own file — zero contention.
type partWriter struct {
	file       *os.File
	bw         *bufio.Writer
	path       string
	written    int64
	downloaded *int64 // shared atomic counter for progress tracking
}

// newPartWriter creates a temp file for the given part under tempDir.
// Format: <taskID>.part.<startOffset>
func newPartWriter(tempDir, taskID string, startOffset int64, downloadedBytes *int64) (*partWriter, error) {
	name := fmt.Sprintf("%s.part.%d", taskID, startOffset)
	path := filepath.Join(tempDir, name)

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to create part file %s: %w", path, err)
	}

	return &partWriter{
		file:       f,
		bw:         bufio.NewWriterSize(f, partFileBufferSize),
		path:       path,
		downloaded: downloadedBytes,
	}, nil
}

// openPartWriter opens an existing part file for append (resume).
func openPartWriter(tempDir, taskID string, startOffset int64, downloadedBytes *int64) (*partWriter, error) {
	name := fmt.Sprintf("%s.part.%d", taskID, startOffset)
	path := filepath.Join(tempDir, name)

	f, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open part file %s: %w", path, err)
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	return &partWriter{
		file:       f,
		bw:         bufio.NewWriterSize(f, partFileBufferSize),
		path:       path,
		written:    info.Size(),
		downloaded: downloadedBytes,
	}, nil
}

// Write appends data to the part file. Non-blocking — sequential I/O only.
func (pw *partWriter) Write(data []byte) error {
	n, err := pw.bw.Write(data)
	if err != nil {
		return err
	}
	pw.written += int64(n)
	atomic.AddInt64(pw.downloaded, int64(n))
	return nil
}

// Close flushes the buffer and closes the underlying file.
func (pw *partWriter) Close() error {
	if err := pw.bw.Flush(); err != nil {
		pw.file.Close()
		return err
	}
	return pw.file.Close()
}

// Path returns the temp file path.
func (pw *partWriter) Path() string {
	return pw.path
}

// Written returns bytes written so far.
func (pw *partWriter) Written() int64 {
	return pw.written
}

// mergePartFiles assembles all part temp files into the final destination.
// Parts are identified by <taskID>.part.<startOffset> — each file is written
// at its byte offset in the destination. This handles work-stealing overlaps
// naturally because later writes to the same region simply overwrite.
func mergePartFiles(tempDir, taskID, destPath string) error {
	pattern := filepath.Join(tempDir, taskID+".part.*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob part files: %w", err)
	}
	if len(matches) == 0 {
		return fmt.Errorf("no part files found for task %s", taskID)
	}

	sort.Slice(matches, func(i, j int) bool {
		return extractPartID(matches[i]) < extractPartID(matches[j])
	})

	dest, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("failed to open destination: %w", err)
	}
	defer dest.Close()

	buf := make([]byte, 4*1024*1024) // 4MB copy buffer

	for _, partPath := range matches {
		offset := int64(extractPartID(partPath))

		pf, err := os.Open(partPath)
		if err != nil {
			return fmt.Errorf("failed to open part %s: %w", partPath, err)
		}
		if _, err := dest.Seek(offset, io.SeekStart); err != nil {
			pf.Close()
			return fmt.Errorf("failed to seek to offset %d: %w", offset, err)
		}
		if _, err := io.CopyBuffer(dest, pf, buf); err != nil {
			pf.Close()
			return fmt.Errorf("failed to copy part %s: %w", partPath, err)
		}
		pf.Close()
		os.Remove(partPath)
	}

	return nil
}

// cleanupPartFiles removes all temp part files for a task.
func cleanupPartFiles(tempDir, taskID string) {
	pattern := filepath.Join(tempDir, taskID+".part.*")
	matches, _ := filepath.Glob(pattern)
	for _, m := range matches {
		os.Remove(m)
	}
}

// cleanupOrphanedParts removes part files whose startOffset
// is not in the planned set (e.g. leftover from work-stealing).
func cleanupOrphanedParts(tempDir, taskID string, plannedOffsets map[int64]bool) {
	pattern := filepath.Join(tempDir, taskID+".part.*")
	matches, _ := filepath.Glob(pattern)
	for _, m := range matches {
		offset := int64(extractPartID(m))
		if !plannedOffsets[offset] {
			os.Remove(m)
		}
	}
}

// partFileExists checks if a completed part file exists with expected size.
func partFileExists(tempDir, taskID string, startOffset int64, expectedSize int64) bool {
	name := fmt.Sprintf("%s.part.%d", taskID, startOffset)
	path := filepath.Join(tempDir, name)
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Size() == expectedSize
}

// extractPartID parses the numeric part ID from a filename like "abc.part.7"
func extractPartID(path string) int {
	base := filepath.Base(path)
	idx := strings.LastIndex(base, ".")
	if idx < 0 {
		return 0
	}
	var id int
	fmt.Sscanf(base[idx+1:], "%d", &id)
	return id
}

// tempDirForTask returns the temp directory for a task's part files.
func tempDirForTask(savePath string) string {
	return filepath.Join(filepath.Dir(savePath), ".tachyon_parts")
}
