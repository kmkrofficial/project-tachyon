package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// handleMediaImage serves an image file with a valid PNG/JPEG/GIF header + padding.
func (ts *TestServer) handleMediaImage(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/media/image/")
	if filename == "" {
		filename = "image.png"
	}
	size := parseSizeQuery(r, 1024*1024) // default 1MB
	ext := extOf(filename)

	var header []byte
	contentType := "application/octet-stream"

	switch ext {
	case ".png":
		header = pngHeader()
		contentType = "image/png"
	case ".jpg", ".jpeg":
		header = jpegHeader()
		contentType = "image/jpeg"
	case ".gif":
		header = gifHeader()
		contentType = "image/gif"
	case ".webp":
		header = webpHeader()
		contentType = "image/webp"
	case ".bmp":
		header = bmpHeader()
		contentType = "image/bmp"
	default:
		header = pngHeader()
		contentType = "image/png"
	}

	ts.serveMediaFile(w, r, size, contentType, filename, header)
}

// handleMediaVideo serves a video file with a valid MP4/WebM header + padding.
func (ts *TestServer) handleMediaVideo(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/media/video/")
	if filename == "" {
		filename = "video.mp4"
	}
	size := parseSizeQuery(r, 5*1024*1024) // default 5MB

	var header []byte
	contentType := "video/mp4"

	switch extOf(filename) {
	case ".mp4":
		header = mp4Header()
		contentType = "video/mp4"
	case ".webm":
		header = webmHeader()
		contentType = "video/webm"
	case ".mkv":
		header = mkvHeader()
		contentType = "video/x-matroska"
	default:
		header = mp4Header()
		contentType = "video/mp4"
	}

	ts.serveMediaFile(w, r, size, contentType, filename, header)
}

// handleMediaAudio serves an audio file with valid MP3/OGG header + padding.
func (ts *TestServer) handleMediaAudio(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/media/audio/")
	if filename == "" {
		filename = "audio.mp3"
	}
	size := parseSizeQuery(r, 2*1024*1024) // default 2MB

	var header []byte
	contentType := "audio/mpeg"

	switch extOf(filename) {
	case ".mp3":
		header = mp3Header()
		contentType = "audio/mpeg"
	case ".ogg":
		header = oggHeader()
		contentType = "audio/ogg"
	case ".wav":
		header = wavHeader(size)
		contentType = "audio/wav"
	case ".flac":
		header = flacHeader()
		contentType = "audio/flac"
	default:
		header = mp3Header()
		contentType = "audio/mpeg"
	}

	ts.serveMediaFile(w, r, size, contentType, filename, header)
}

// handleMediaDoc serves a document file (PDF/TXT) with valid header + padding.
func (ts *TestServer) handleMediaDoc(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/media/document/")
	if filename == "" {
		filename = "document.pdf"
	}
	size := parseSizeQuery(r, 500*1024) // default 500KB

	var header []byte
	contentType := "application/octet-stream"

	switch extOf(filename) {
	case ".pdf":
		header = pdfHeader()
		contentType = "application/pdf"
	case ".txt":
		header = []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit.\n")
		contentType = "text/plain"
	default:
		header = pdfHeader()
		contentType = "application/pdf"
	}

	ts.serveMediaFile(w, r, size, contentType, filename, header)
}

// handleMediaZip serves an archive file with valid ZIP/TAR header + padding.
func (ts *TestServer) handleMediaZip(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/media/archive/")
	if filename == "" {
		filename = "archive.zip"
	}
	size := parseSizeQuery(r, 3*1024*1024) // default 3MB

	var header []byte
	contentType := "application/octet-stream"

	switch extOf(filename) {
	case ".zip":
		header = zipHeader()
		contentType = "application/zip"
	case ".tar":
		header = tarHeader()
		contentType = "application/x-tar"
	case ".gz", ".tar.gz", ".tgz":
		header = gzipHeader()
		contentType = "application/gzip"
	case ".7z":
		header = sevenZipHeader()
		contentType = "application/x-7z-compressed"
	default:
		header = zipHeader()
		contentType = "application/zip"
	}

	ts.serveMediaFile(w, r, size, contentType, filename, header)
}

// serveMediaFile serves media content: valid file header followed by pattern fill.
// For non-range requests, writes the magic header bytes first, then pattern data for the remainder.
// For range requests, delegates entirely to serveRangeContent (range offsets are into the logical file).
func (ts *TestServer) serveMediaFile(
	w http.ResponseWriter, r *http.Request,
	size int64, contentType, filename string, header []byte,
) {
	ts.log.Info("serving media", "handler", "media", "filename", filename,
		"content_type", contentType, "size", size, "range", r.Header.Get("Range"), "remote", r.RemoteAddr)

	// For range requests, delegate to range handler (content is pattern-only for simplicity).
	if r.Header.Get("Range") != "" {
		ts.serveRangeContent(w, r, size, contentType, filename)
		return
	}

	// Full request: write header bytes + pattern fill for the remainder.
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	if filename != "" {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	}
	w.WriteHeader(http.StatusOK)

	headerLen := int64(len(header))
	if headerLen > size {
		headerLen = size
	}
	w.Write(header[:headerLen])
	written := headerLen

	if written < size {
		written += writePatternDataFrom(w, written, size-written)
	}
	ts.recordBytes(written)
}

// parseSizeQuery reads the "size" query parameter, falling back to defaultSize.
func parseSizeQuery(r *http.Request, defaultSize int64) int64 {
	s := r.URL.Query().Get("size")
	if s == "" {
		return defaultSize
	}
	v, err := parseSize(s)
	if err != nil {
		return defaultSize
	}
	return v
}

// extOf returns the lowercase extension of a filename.
func extOf(filename string) string {
	idx := strings.LastIndex(filename, ".")
	if idx < 0 {
		return ""
	}
	return strings.ToLower(filename[idx:])
}

// parseSize converts human-readable sizes like "10mb", "500kb" to bytes.
func parseSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	multipliers := []struct {
		suffix string
		mult   int64
	}{
		{"gb", 1024 * 1024 * 1024},
		{"mb", 1024 * 1024},
		{"kb", 1024},
		{"b", 1},
	}
	for _, m := range multipliers {
		if strings.HasSuffix(s, m.suffix) {
			num := strings.TrimSuffix(s, m.suffix)
			v, err := strconv.ParseFloat(num, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid size: %s", s)
			}
			return int64(v * float64(m.mult)), nil
		}
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size: %s", s)
	}
	return v, nil
}

// parseSizeFromPath extracts and parses a size from the URL path.
func parseSizeFromPath(path, prefix string) (int64, error) {
	trimmed := strings.TrimPrefix(path, prefix)
	if trimmed == "" {
		return 0, fmt.Errorf("missing size in path")
	}
	return parseSize(trimmed)
}
