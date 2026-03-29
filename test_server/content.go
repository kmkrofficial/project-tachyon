package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// fillPattern fills buf with deterministic bytes based on offset.
// Uses prime modulus 251 for non-repeating, verifiable content.
func fillPattern(buf []byte, offset int64) {
	for i := range buf {
		buf[i] = byte((offset + int64(i)) % 251)
	}
}

// writePatternData writes deterministic content from offset 0.
func writePatternData(w io.Writer, size int64) int64 {
	return writePatternDataFrom(w, 0, size)
}

// writePatternDataFrom writes deterministic content from a given offset.
func writePatternDataFrom(w io.Writer, offset, size int64) int64 {
	const chunkSize = 32 * 1024
	buf := make([]byte, chunkSize)
	var written int64

	for written < size {
		remaining := size - written
		if remaining < chunkSize {
			buf = buf[:remaining]
		}
		fillPattern(buf, offset+written)
		n, err := w.Write(buf)
		written += int64(n)
		if err != nil {
			break
		}
	}
	return written
}

// serveRangeContent serves content with HTTP Range request support.
func (ts *TestServer) serveRangeContent(
	w http.ResponseWriter, r *http.Request,
	totalSize int64, contentType, filename string,
) {
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", contentType)
	if filename != "" {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	}

	// Wrap writer with global throttle
	var out io.Writer = w
	if ts.throttle != nil {
		out = &throttledWriter{w: w, throttle: ts.throttle}
	}

	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		// Full content
		w.Header().Set("Content-Length", strconv.FormatInt(totalSize, 10))
		w.WriteHeader(http.StatusOK)
		written := writePatternData(out, totalSize)
		ts.recordBytes(written)
		return
	}

	// Parse Range header: "bytes=start-end"
	start, end, err := parseRangeHeader(rangeHeader, totalSize)
	if err != nil {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", totalSize))
		http.Error(w, "invalid range", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	length := end - start + 1
	w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, totalSize))
	w.WriteHeader(http.StatusPartialContent)

	written := writePatternDataFrom(out, start, length)
	ts.recordBytes(written)
}

// parseRangeHeader parses "bytes=start-end" or "bytes=start-" headers.
func parseRangeHeader(header string, totalSize int64) (int64, int64, error) {
	if len(header) < 6 || header[:6] != "bytes=" {
		return 0, 0, fmt.Errorf("invalid range format")
	}
	rangeSpec := header[6:]
	dashIdx := -1
	for i, c := range rangeSpec {
		if c == '-' {
			dashIdx = i
			break
		}
	}
	if dashIdx < 0 {
		return 0, 0, fmt.Errorf("invalid range format")
	}

	startStr := rangeSpec[:dashIdx]
	endStr := rangeSpec[dashIdx+1:]

	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid start: %w", err)
	}

	var end int64
	if endStr == "" {
		end = totalSize - 1
	} else {
		end, err = strconv.ParseInt(endStr, 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid end: %w", err)
		}
	}

	if start > end || start >= totalSize {
		return 0, 0, fmt.Errorf("range out of bounds")
	}
	if end >= totalSize {
		end = totalSize - 1
	}

	return start, end, nil
}

// serveThrottled serves content at a limited speed (bytes per second).
func (ts *TestServer) serveThrottled(w http.ResponseWriter, size, bytesPerSec int64) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(http.StatusOK)

	chunkSize := bytesPerSec / 10
	if chunkSize < 1024 {
		chunkSize = 1024
	}
	interval := time.Second / 10

	buf := make([]byte, chunkSize)
	var written int64

	for written < size {
		remaining := size - written
		if remaining < chunkSize {
			buf = buf[:remaining]
		}
		fillPattern(buf, written)
		n, err := w.Write(buf)
		written += int64(n)
		if err != nil {
			break
		}
		time.Sleep(interval)
	}
	ts.recordBytes(written)
}

// serveThrottledRange serves a range request with per-request speed throttling.
func (ts *TestServer) serveThrottledRange(w http.ResponseWriter, r *http.Request, totalSize, bytesPerSec int64) {
	start, end, err := parseRangeHeader(r.Header.Get("Range"), totalSize)
	if err != nil {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", totalSize))
		http.Error(w, "invalid range", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	length := end - start + 1
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, totalSize))
	w.WriteHeader(http.StatusPartialContent)

	chunkSize := bytesPerSec / 10
	if chunkSize < 1024 {
		chunkSize = 1024
	}
	interval := time.Second / 10

	buf := make([]byte, chunkSize)
	var written int64

	for written < length {
		remaining := length - written
		if remaining < chunkSize {
			buf = buf[:remaining]
		}
		fillPattern(buf, start+written)
		n, err := w.Write(buf)
		written += int64(n)
		if err != nil {
			break
		}
		time.Sleep(interval)
	}
	ts.recordBytes(written)
}
