package security

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Scanner interface for antivirus integration
type Scanner interface {
	// ScanFile scans the file at the given path
	// Returns nil if clean, error if threat detected or scan failed
	ScanFile(ctx context.Context, filePath string) error
	// Name returns the scanner name for logging
	Name() string
}

// ScanResult represents the outcome of a scan
type ScanResult struct {
	Clean   bool
	Threat  string
	Message string
}

// execCommandFunc is a function type for creating exec.Cmd, allowing injection for testing
type execCommandFunc func(ctx context.Context, name string, arg ...string) *exec.Cmd

// defaultExecCommand is the default implementation using exec.CommandContext
func defaultExecCommand(ctx context.Context, name string, arg ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, arg...)
}

// WindowsDefenderScanner triggers Windows Defender via MpCmdRun.exe
type WindowsDefenderScanner struct {
	logger      *slog.Logger
	execCommand execCommandFunc
	timeout     time.Duration
}

// NewWindowsDefenderScanner creates a new Windows Defender scanner
func NewWindowsDefenderScanner(logger *slog.Logger) *WindowsDefenderScanner {
	return &WindowsDefenderScanner{
		logger:      logger,
		execCommand: defaultExecCommand,
		timeout:     60 * time.Second, // 60 second timeout for large files
	}
}

// SetExecCommand sets a custom exec command function (for testing)
func (s *WindowsDefenderScanner) SetExecCommand(fn execCommandFunc) {
	s.execCommand = fn
}

// Name returns the scanner name
func (s *WindowsDefenderScanner) Name() string {
	return "Windows Defender"
}

// ScanFile triggers Windows Defender to scan the specified file
// Exit codes:
//   - 0: No threats detected
//   - 2: Threats detected and remediated/quarantined
//   - Other: Scan error or file not found
func (s *WindowsDefenderScanner) ScanFile(ctx context.Context, filePath string) error {
	// Create timeout context
	scanCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// MpCmdRun.exe path (standard Windows location)
	mpCmdPath := `C:\Program Files\Windows Defender\MpCmdRun.exe`

	// Arguments:
	// -Scan: Perform scan
	// -ScanType 3: Custom file scan
	// -File: Specify file to scan
	// -DisableRemediation: Don't auto-quarantine (just report)
	cmd := s.execCommand(scanCtx, mpCmdPath, "-Scan", "-ScanType", "3", "-File", filePath, "-DisableRemediation")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	s.logger.Info("Starting AV scan", "scanner", s.Name(), "file", filePath)

	err := cmd.Run()

	// Check for timeout
	if scanCtx.Err() == context.DeadlineExceeded {
		s.logger.Warn("AV scan timed out", "file", filePath, "timeout", s.timeout)
		return fmt.Errorf("scan timed out after %v", s.timeout)
	}

	// Check for context cancellation (e.g., download cancelled)
	if scanCtx.Err() == context.Canceled {
		s.logger.Info("AV scan cancelled", "file", filePath)
		return nil // Don't report error if user cancelled
	}

	// Parse exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			switch exitCode {
			case 2:
				// Threat detected
				output := stdout.String()
				threat := parseThreatFromOutput(output)
				s.logger.Warn("Threat detected by AV", "file", filePath, "threat", threat)
				return fmt.Errorf("threat detected: %s", threat)
			default:
				// Other error (file not found, permission denied, etc.)
				s.logger.Warn("AV scan failed", "file", filePath, "exitCode", exitCode, "stderr", stderr.String())
				return fmt.Errorf("scan failed with exit code %d", exitCode)
			}
		}
		// exec error (command not found, etc.)
		s.logger.Warn("AV scanner not available", "error", err)
		return fmt.Errorf("scanner not available: %w", err)
	}

	// Exit code 0: Clean
	s.logger.Info("AV scan completed - clean", "file", filePath)
	return nil
}

// parseThreatFromOutput extracts threat name from Windows Defender output
func parseThreatFromOutput(output string) string {
	// Windows Defender output contains "Threat" information
	// Example: "Threat                  : Trojan:Win32/Example.A!ml"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Threat") && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "unknown threat"
}

// NoOpScanner is a scanner that does nothing (for non-Windows platforms)
type NoOpScanner struct {
	logger *slog.Logger
}

// NewNoOpScanner creates a new no-op scanner
func NewNoOpScanner(logger *slog.Logger) *NoOpScanner {
	return &NoOpScanner{logger: logger}
}

// Name returns the scanner name
func (s *NoOpScanner) Name() string {
	return "NoOp (Native AV not available)"
}

// ScanFile logs a warning and returns nil (no scanning performed)
func (s *NoOpScanner) ScanFile(ctx context.Context, filePath string) error {
	s.logger.Warn("Native AV scanning skipped - not available on this platform",
		"platform", runtime.GOOS,
		"file", filePath,
	)
	return nil
}

// NewScanner creates the appropriate scanner for the current platform
func NewScanner(logger *slog.Logger) Scanner {
	if runtime.GOOS == "windows" {
		return NewWindowsDefenderScanner(logger)
	}
	// Linux/Mac: Return no-op scanner
	return NewNoOpScanner(logger)
}
