package core

import (
	"log/slog"
	"os"
	"testing"
)

func TestNewEngine(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// Pass nil for storage in basic test for now, or mock if strictly required
	// Since NewEngine just assigns it, nil is fine for this specific test
	engine := NewEngine(logger, nil)

	if engine == nil {
		t.Fatal("NewEngine returned nil")
	}
}

func TestStartDownload_Validation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	engine := NewEngine(logger, nil)

	// Test Empty URL
	_, err := engine.StartDownload("", "test_path")
	if err == nil {
		t.Error("Expected error for empty URL, got nil")
	}

	// Test Empty Path (grab might handle this, but let's see)
	// Actually StartDownload doesn't validate path explicitly, but grab.NewRequest might.
}
