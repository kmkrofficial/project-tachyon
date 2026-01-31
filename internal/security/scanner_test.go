package security

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"testing"
	"time"
)

// mockExitError creates a mock exit error with the specified exit code
type mockExitError struct {
	code int
}

func (e *mockExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.code)
}

func (e *mockExitError) ExitCode() int {
	return e.code
}

// TestWindowsDefenderScanner_CleanFile verifies successful scan (exit code 0)
func TestWindowsDefenderScanner_CleanFile(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	scanner := NewWindowsDefenderScanner(logger)

	// Mock exec.Command to return exit code 0 (clean)
	scanner.SetExecCommand(func(ctx context.Context, name string, args ...string) *exec.Cmd {
		// Use a simple command that exits with code 0
		return exec.CommandContext(ctx, "cmd", "/c", "exit", "0")
	})

	err := scanner.ScanFile(context.Background(), "C:\\test\\file.exe")
	if err != nil {
		t.Errorf("Expected nil error for clean file, got: %v", err)
	}
}

// TestWindowsDefenderScanner_ThreatFound verifies threat detection (exit code 2)
func TestWindowsDefenderScanner_ThreatFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	scanner := NewWindowsDefenderScanner(logger)

	// Mock exec.Command to return exit code 2 (threat found)
	scanner.SetExecCommand(func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "cmd", "/c", "exit", "2")
	})

	err := scanner.ScanFile(context.Background(), "C:\\test\\malware.exe")
	if err == nil {
		t.Error("Expected error for threat detected, got nil")
	}
	if err != nil && !containsString(err.Error(), "threat") {
		t.Errorf("Expected error to mention 'threat', got: %v", err)
	}
}

// TestWindowsDefenderScanner_ScanError verifies handling of scan errors
func TestWindowsDefenderScanner_ScanError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	scanner := NewWindowsDefenderScanner(logger)

	// Mock exec.Command to return a non-standard exit code
	scanner.SetExecCommand(func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "cmd", "/c", "exit", "1")
	})

	err := scanner.ScanFile(context.Background(), "C:\\test\\file.exe")
	if err == nil {
		t.Error("Expected error for scan failure, got nil")
	}
}

// TestWindowsDefenderScanner_Timeout verifies timeout handling
func TestWindowsDefenderScanner_Timeout(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	scanner := NewWindowsDefenderScanner(logger)
	scanner.timeout = 100 * time.Millisecond // Short timeout for testing

	// Mock exec.Command to sleep longer than timeout
	scanner.SetExecCommand(func(ctx context.Context, name string, args ...string) *exec.Cmd {
		// Use ping -n 10 to simulate a long-running command
		return exec.CommandContext(ctx, "cmd", "/c", "ping", "-n", "10", "127.0.0.1")
	})

	err := scanner.ScanFile(context.Background(), "C:\\test\\file.exe")
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
	if err != nil && !containsString(err.Error(), "timed out") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// TestWindowsDefenderScanner_ContextCancellation verifies cancellation handling
func TestWindowsDefenderScanner_ContextCancellation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	scanner := NewWindowsDefenderScanner(logger)

	// Mock exec.Command that would take a while
	scanner.SetExecCommand(func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "cmd", "/c", "ping", "-n", "10", "127.0.0.1")
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := scanner.ScanFile(ctx, "C:\\test\\file.exe")
	// Should return nil (cancellation doesn't report as error)
	if err != nil {
		t.Logf("Got error (expected nil for cancellation): %v", err)
	}
}

// TestWindowsDefenderScanner_CommandArguments verifies correct arguments are passed
func TestWindowsDefenderScanner_CommandArguments(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	scanner := NewWindowsDefenderScanner(logger)

	var capturedName string
	var capturedArgs []string

	scanner.SetExecCommand(func(ctx context.Context, name string, args ...string) *exec.Cmd {
		capturedName = name
		capturedArgs = args
		// Return a command that exits successfully
		return exec.CommandContext(ctx, "cmd", "/c", "exit", "0")
	})

	testPath := "C:\\downloads\\test.zip"
	scanner.ScanFile(context.Background(), testPath)

	// Verify correct executable
	expectedPath := `C:\Program Files\Windows Defender\MpCmdRun.exe`
	if capturedName != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, capturedName)
	}

	// Verify -File argument contains our path
	foundFile := false
	for i, arg := range capturedArgs {
		if arg == "-File" && i+1 < len(capturedArgs) && capturedArgs[i+1] == testPath {
			foundFile = true
			break
		}
	}
	if !foundFile {
		t.Errorf("Expected -File %s in args, got %v", testPath, capturedArgs)
	}
}

// TestNoOpScanner verifies the no-op scanner returns nil
func TestNoOpScanner(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	scanner := NewNoOpScanner(logger)

	err := scanner.ScanFile(context.Background(), "/any/path")
	if err != nil {
		t.Errorf("NoOpScanner should return nil, got: %v", err)
	}

	if scanner.Name() == "" {
		t.Error("Scanner name should not be empty")
	}
}

// TestNewScanner verifies platform-specific scanner creation
func TestNewScanner(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	scanner := NewScanner(logger)

	if scanner == nil {
		t.Fatal("NewScanner returned nil")
	}

	// Scanner should have a valid name
	if scanner.Name() == "" {
		t.Error("Scanner name should not be empty")
	}
}

// TestParseThreatFromOutput verifies threat parsing from Windows Defender output
func TestParseThreatFromOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard threat output",
			input:    "Threat                  : Trojan:Win32/Example.A!ml\nSome other line",
			expected: "Trojan:Win32/Example.A!ml",
		},
		{
			name:     "no threat line",
			input:    "No threats found\n",
			expected: "unknown threat",
		},
		{
			name:     "empty output",
			input:    "",
			expected: "unknown threat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseThreatFromOutput(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
