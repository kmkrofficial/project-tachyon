package security

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowsDefenderScanner_ScanFile_Real(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows Defender test on non-Windows OS")
	}

	// check if MpCmdRun.exe exists
	if _, err := os.Stat("C:\\Program Files\\Windows Defender\\MpCmdRun.exe"); os.IsNotExist(err) {
		t.Skip("Skipping Windows Defender test: MpCmdRun.exe not found")
	}

	// Create a temporary clean file
	tmpDir := t.TempDir()
	cleanFile := filepath.Join(tmpDir, "clean_file.txt")
	err := os.WriteFile(cleanFile, []byte("This is a clean test file for Project Tachyon AV scanning."), 0644)
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	scanner := NewWindowsDefenderScanner(logger)

	// We want to test the REAL execution, so we don't mock execCommand
	// scanner.timeout is already set to 5 mins by default now

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	start := time.Now()
	err = scanner.ScanFile(ctx, cleanFile)
	duration := time.Since(start)

	t.Logf("Scan took %v", duration)

	assert.NoError(t, err, "ScanFile should not return error for a clean file")
}

func TestWindowsDefenderScanner_ScanTimeout(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows Defender test on non-Windows OS")
	}

	// Mock logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	scanner := NewWindowsDefenderScanner(logger)

	// Set a very short timeout to force a timeout error
	scanner.timeout = 1 * time.Millisecond

	// Create a temporary clean file
	tmpDir := t.TempDir()
	cleanFile := filepath.Join(tmpDir, "timeout_file.txt")
	err := os.WriteFile(cleanFile, []byte("This file scan should timeout."), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	err = scanner.ScanFile(ctx, cleanFile)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scan timed out")
}
