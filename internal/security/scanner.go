package security

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
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

// ClamAVScanner connects to a ClamAV daemon via TCP socket
type ClamAVScanner struct {
	logger  *slog.Logger
	host    string
	timeout time.Duration
	// dialFunc allows injection for testing
	dialFunc func(ctx context.Context, network, address string) (net.Conn, error)
}

// NewClamAVScanner creates a new ClamAV scanner
// host should be in format "hostname:port" (e.g., "localhost:3310")
func NewClamAVScanner(logger *slog.Logger, host string) *ClamAVScanner {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return &ClamAVScanner{
		logger:   logger,
		host:     host,
		timeout:  60 * time.Second,
		dialFunc: dialer.DialContext,
	}
}

// SetDialFunc sets a custom dial function (for testing)
func (s *ClamAVScanner) SetDialFunc(fn func(ctx context.Context, network, address string) (net.Conn, error)) {
	s.dialFunc = fn
}

// Name returns the scanner name
func (s *ClamAVScanner) Name() string {
	return "ClamAV"
}

// ScanFile scans a file using ClamAV's INSTREAM protocol
// Protocol: zINSTREAM\0 followed by chunks: [4-byte big-endian length][data]
// Terminate with 4 zero bytes
func (s *ClamAVScanner) ScanFile(ctx context.Context, filePath string) error {
	// Create timeout context
	scanCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	s.logger.Info("Starting ClamAV scan", "host", s.host, "file", filePath)

	// Connect to ClamAV daemon
	conn, err := s.dialFunc(scanCtx, "tcp", s.host)
	if err != nil {
		s.logger.Warn("Failed to connect to ClamAV", "host", s.host, "error", err)
		return fmt.Errorf("failed to connect to ClamAV at %s: %w", s.host, err)
	}
	defer conn.Close()

	// Set deadline on connection
	if deadline, ok := scanCtx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}

	// Open the file to scan
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for scanning: %w", err)
	}
	defer file.Close()

	// Send INSTREAM command (null-terminated)
	if _, err := conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return fmt.Errorf("failed to send INSTREAM command: %w", err)
	}

	// Stream file in chunks
	chunkSize := 8192 // 8KB chunks
	buf := make([]byte, chunkSize)
	sizeBuf := make([]byte, 4)

	for {
		n, readErr := file.Read(buf)
		if n > 0 {
			// Write chunk size (4 bytes big-endian)
			sizeBuf[0] = byte(n >> 24)
			sizeBuf[1] = byte(n >> 16)
			sizeBuf[2] = byte(n >> 8)
			sizeBuf[3] = byte(n)
			if _, err := conn.Write(sizeBuf); err != nil {
				return fmt.Errorf("failed to send chunk size: %w", err)
			}
			// Write chunk data
			if _, err := conn.Write(buf[:n]); err != nil {
				return fmt.Errorf("failed to send chunk data: %w", err)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("failed to read file: %w", readErr)
		}
	}

	// Send terminating zero-length chunk
	if _, err := conn.Write([]byte{0, 0, 0, 0}); err != nil {
		return fmt.Errorf("failed to send termination: %w", err)
	}

	// Read response
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read ClamAV response: %w", err)
	}

	result := strings.TrimSpace(string(response[:n]))
	s.logger.Debug("ClamAV response", "response", result)

	// Parse response
	// Clean: "stream: OK"
	// Infected: "stream: <virus_name> FOUND"
	if strings.HasSuffix(result, "OK") {
		s.logger.Info("ClamAV scan completed - clean", "file", filePath)
		return nil
	}

	if strings.Contains(result, "FOUND") {
		// Extract virus name
		threat := parseClamAVThreat(result)
		s.logger.Warn("ClamAV detected threat", "file", filePath, "threat", threat)
		return fmt.Errorf("threat detected: %s", threat)
	}

	// Unexpected response
	s.logger.Warn("ClamAV unexpected response", "response", result)
	return fmt.Errorf("unexpected ClamAV response: %s", result)
}

// parseClamAVThreat extracts the threat name from ClamAV response
// Format: "stream: Eicar-Test-Signature FOUND"
func parseClamAVThreat(response string) string {
	// Remove "stream: " prefix
	response = strings.TrimPrefix(response, "stream: ")
	// Remove " FOUND" suffix
	if idx := strings.LastIndex(response, " FOUND"); idx != -1 {
		return response[:idx]
	}
	return response
}

// NewScanner creates the appropriate scanner for the current platform
// Priority: ClamAV (if CLAMAV_HOST set) > Windows Defender (on Windows) > NoOp
func NewScanner(logger *slog.Logger) Scanner {
	// Check for ClamAV host environment variable
	clamavHost := os.Getenv("CLAMAV_HOST")
	if clamavHost != "" {
		logger.Info("Using ClamAV scanner", "host", clamavHost)
		return NewClamAVScanner(logger, clamavHost)
	}

	if runtime.GOOS == "windows" {
		return NewWindowsDefenderScanner(logger)
	}
	// Linux/Mac without ClamAV: Return no-op scanner
	return NewNoOpScanner(logger)
}
