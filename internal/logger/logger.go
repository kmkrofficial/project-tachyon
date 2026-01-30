package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ANSI color codes
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
)

type ConsoleHandler struct {
	mu  sync.Mutex
	out io.Writer
}

func NewConsoleHandler(out io.Writer) *ConsoleHandler {
	return &ConsoleHandler{out: out}
}

func (h *ConsoleHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *ConsoleHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	levelColor := Reset
	switch r.Level {
	case slog.LevelDebug:
		levelColor = Gray
	case slog.LevelInfo:
		levelColor = Green
	case slog.LevelWarn:
		levelColor = Yellow
	case slog.LevelError:
		levelColor = Red
	}

	timeStr := r.Time.Format(time.TimeOnly)
	msg := fmt.Sprintf("%s%s%s [%s] %s\n", levelColor, r.Level.String()[:4], Reset, timeStr, r.Message)

	_, err := h.out.Write([]byte(msg))
	return err
}

func (h *ConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *ConsoleHandler) WithGroup(name string) slog.Handler {
	return h
}

// WailsHandler emits logs as Wails events
type WailsHandler struct {
	mu  sync.Mutex
	ctx context.Context
}

func NewWailsHandler() *WailsHandler {
	return &WailsHandler{}
}

func (h *WailsHandler) SetContext(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ctx = ctx
}

func (h *WailsHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *WailsHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.ctx == nil {
		return nil
	}

	data := make(map[string]interface{})
	r.Attrs(func(a slog.Attr) bool {
		data[a.Key] = a.Value.Any()
		return true
	})

	runtime.EventsEmit(h.ctx, "log:entry", map[string]interface{}{
		"level":   r.Level.String(),
		"message": r.Message,
		"time":    r.Time.Format(time.RFC3339),
		"data":    data,
	})

	return nil
}

func (h *WailsHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h // Simplification
}

func (h *WailsHandler) WithGroup(name string) slog.Handler {
	return h
}

// New creates a new logger with FanoutHandler (JSON in File + Console + Wails)
func New(consoleOutput io.Writer) (*slog.Logger, *WailsHandler, error) {
	appData, err := os.UserConfigDir()
	if err != nil {
		return nil, nil, err
	}
	logDir := filepath.Join(appData, "Tachyon", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, nil, err
	}

	f, err := os.OpenFile(filepath.Join(logDir, "app.json"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}

	jsonHandler := slog.NewJSONHandler(f, nil)
	consoleHandler := NewConsoleHandler(consoleOutput)
	wailsHandler := NewWailsHandler()

	// Simple fanout handler
	handler := &FanoutHandler{
		handlers: []slog.Handler{jsonHandler, consoleHandler, wailsHandler},
	}

	return slog.New(handler), wailsHandler, nil
}

type FanoutHandler struct {
	handlers []slog.Handler
}

func (h *FanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *FanoutHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		_ = handler.Handle(ctx, r)
	}
	return nil
}

func (h *FanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}
	return &FanoutHandler{handlers: newHandlers}
}

func (h *FanoutHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	return &FanoutHandler{handlers: newHandlers}
}
