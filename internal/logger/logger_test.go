package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// --- ConsoleHandler ---

func TestConsoleHandler_Handle(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(&buf)

	record := slog.NewRecord(fixedTime(), slog.LevelInfo, "test message", 0)
	if err := h.Handle(context.Background(), record); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("expected INFO in output, got %q", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("expected message in output, got %q", output)
	}
}

func TestConsoleHandler_Levels(t *testing.T) {
	levels := []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError}
	for _, level := range levels {
		var buf bytes.Buffer
		h := NewConsoleHandler(&buf)

		record := slog.NewRecord(fixedTime(), level, "msg", 0)
		if err := h.Handle(context.Background(), record); err != nil {
			t.Fatalf("Handle failed for %s: %v", level, err)
		}

		if buf.Len() == 0 {
			t.Errorf("expected output for level %s", level)
		}
	}
}

func TestConsoleHandler_Enabled(t *testing.T) {
	h := NewConsoleHandler(&bytes.Buffer{})
	// Should be enabled for all levels
	if !h.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("should be enabled for Debug")
	}
	if !h.Enabled(context.Background(), slog.LevelError) {
		t.Error("should be enabled for Error")
	}
}

func TestConsoleHandler_WithAttrs(t *testing.T) {
	h := NewConsoleHandler(&bytes.Buffer{})
	h2 := h.WithAttrs([]slog.Attr{slog.String("key", "val")})
	if h2 == nil {
		t.Error("WithAttrs should return non-nil handler")
	}
}

func TestConsoleHandler_WithGroup(t *testing.T) {
	h := NewConsoleHandler(&bytes.Buffer{})
	h2 := h.WithGroup("group")
	if h2 == nil {
		t.Error("WithGroup should return non-nil handler")
	}
}

// --- WailsHandler ---

func TestWailsHandler_NoContext(t *testing.T) {
	h := NewWailsHandler()
	// Handle without context should not panic
	record := slog.NewRecord(fixedTime(), slog.LevelInfo, "test", 0)
	err := h.Handle(context.Background(), record)
	if err != nil {
		t.Errorf("Handle without context should return nil, got %v", err)
	}
}

func TestWailsHandler_Enabled(t *testing.T) {
	h := NewWailsHandler()
	if !h.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("should be enabled for all levels")
	}
}

func TestWailsHandler_WithAttrs(t *testing.T) {
	h := NewWailsHandler()
	h2 := h.WithAttrs(nil)
	if h2 == nil {
		t.Error("WithAttrs should return non-nil")
	}
}

func TestWailsHandler_WithGroup(t *testing.T) {
	h := NewWailsHandler()
	h2 := h.WithGroup("test")
	if h2 == nil {
		t.Error("WithGroup should return non-nil")
	}
}

func TestNewWailsHandler(t *testing.T) {
	h := NewWailsHandler()
	if h == nil {
		t.Fatal("NewWailsHandler returned nil")
	}
}

// --- FanoutHandler ---

func TestFanoutHandler_Enabled(t *testing.T) {
	buf := &bytes.Buffer{}
	h := &FanoutHandler{
		handlers: []slog.Handler{
			NewConsoleHandler(buf),
		},
	}

	if !h.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("FanoutHandler should be enabled if any child is enabled")
	}
}

func TestFanoutHandler_Handle(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	h := &FanoutHandler{
		handlers: []slog.Handler{
			NewConsoleHandler(&buf1),
			NewConsoleHandler(&buf2),
		},
	}

	record := slog.NewRecord(fixedTime(), slog.LevelWarn, "fanout test", 0)
	if err := h.Handle(context.Background(), record); err != nil {
		t.Fatalf("FanoutHandler.Handle failed: %v", err)
	}

	if buf1.Len() == 0 {
		t.Error("handler 1 should have received log")
	}
	if buf2.Len() == 0 {
		t.Error("handler 2 should have received log")
	}
}

func TestFanoutHandler_WithAttrs(t *testing.T) {
	h := &FanoutHandler{
		handlers: []slog.Handler{
			NewConsoleHandler(&bytes.Buffer{}),
		},
	}
	h2 := h.WithAttrs([]slog.Attr{slog.String("k", "v")})
	if h2 == nil {
		t.Error("WithAttrs should return non-nil")
	}
	f2, ok := h2.(*FanoutHandler)
	if !ok {
		t.Fatal("WithAttrs should return *FanoutHandler")
	}
	if len(f2.handlers) != 1 {
		t.Error("should preserve handler count")
	}
}

func TestFanoutHandler_WithGroup(t *testing.T) {
	h := &FanoutHandler{
		handlers: []slog.Handler{
			NewConsoleHandler(&bytes.Buffer{}),
			NewConsoleHandler(&bytes.Buffer{}),
		},
	}
	h2 := h.WithGroup("group")
	if h2 == nil {
		t.Error("WithGroup should return non-nil")
	}
	f2, ok := h2.(*FanoutHandler)
	if !ok {
		t.Fatal("WithGroup should return *FanoutHandler")
	}
	if len(f2.handlers) != 2 {
		t.Error("should preserve handler count")
	}
}

func TestFanoutHandler_EmptyHandlers(t *testing.T) {
	h := &FanoutHandler{handlers: nil}

	if h.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("no handlers → not enabled")
	}

	record := slog.NewRecord(fixedTime(), slog.LevelInfo, "msg", 0)
	if err := h.Handle(context.Background(), record); err != nil {
		t.Errorf("Handle with no handlers should not error: %v", err)
	}
}

// --- New (logger constructor) ---

func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer
	logger, wailsHandler, err := New(&buf)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if logger == nil {
		t.Fatal("logger is nil")
	}
	if wailsHandler == nil {
		t.Fatal("wailsHandler is nil")
	}

	// Log something and verify console output
	logger.Info("hello from test")
	if buf.Len() == 0 {
		t.Error("expected console output after logging")
	}
}

// --- Color constants ---

func TestColorConstants(t *testing.T) {
	// Verify ANSI codes are defined and non-empty
	colors := []string{Reset, Red, Green, Yellow, Blue, Purple, Cyan, Gray}
	for _, c := range colors {
		if c == "" {
			t.Error("color constant should not be empty")
		}
	}
}

// helper
func fixedTime() time.Time {
	return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
}
