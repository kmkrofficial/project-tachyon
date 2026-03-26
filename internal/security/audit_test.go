package security

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestNewAuditLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	al := NewAuditLogger(logger)
	if al == nil {
		t.Fatal("NewAuditLogger returned nil")
	}
	defer al.Close()

	if al.logPath == "" {
		t.Error("logPath should not be empty")
	}
}

func TestAuditLogger_Log(t *testing.T) {
	// Setup temp log directory
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_audit.log")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}

	al := &AuditLogger{
		logFile: f,
		logPath: logPath,
		logger:  logger,
	}
	defer al.Close()

	// Log an entry
	al.Log("127.0.0.1", "test-agent", "POST /v1/test", 200, "test details")

	// Read and verify log file
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log: %v", err)
	}

	var entry AccessLogEntry
	if err := json.Unmarshal(content[:len(content)-1], &entry); err != nil {
		t.Fatalf("Failed to parse log entry: %v", err)
	}

	if entry.SourceIP != "127.0.0.1" {
		t.Errorf("SourceIP = %q, want %q", entry.SourceIP, "127.0.0.1")
	}
	if entry.Action != "POST /v1/test" {
		t.Errorf("Action = %q, want %q", entry.Action, "POST /v1/test")
	}
	if entry.Status != 200 {
		t.Errorf("Status = %d, want 200", entry.Status)
	}
	if entry.Details != "test details" {
		t.Errorf("Details = %q, want %q", entry.Details, "test details")
	}
	if entry.ID == "" {
		t.Error("ID should not be empty")
	}
}

func TestAuditLogger_GetRecentLogs(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_audit.log")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}

	al := &AuditLogger{
		logFile: f,
		logPath: logPath,
		logger:  logger,
	}
	defer al.Close()

	// Log multiple entries
	al.Log("10.0.0.1", "agent-1", "GET /status", 200, "OK")
	al.Log("10.0.0.2", "agent-2", "POST /download", 201, "Created")
	al.Log("10.0.0.3", "agent-3", "GET /unauthorized", 403, "Forbidden")

	// Get recent logs
	entries := al.GetRecentLogs(2)
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}

	// Should be in reverse chronological order
	if entries[0].SourceIP != "10.0.0.3" {
		t.Errorf("First entry IP = %q, want %q", entries[0].SourceIP, "10.0.0.3")
	}
	if entries[1].SourceIP != "10.0.0.2" {
		t.Errorf("Second entry IP = %q, want %q", entries[1].SourceIP, "10.0.0.2")
	}
}

func TestAuditLogger_GetRecentLogs_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "empty.log")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	al := &AuditLogger{
		logPath: logPath,
		logger:  logger,
	}

	entries := al.GetRecentLogs(10)
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries for empty log, got %d", len(entries))
	}
}

func TestAuditLogger_Close(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "close_test.log")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	f, _ := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	al := &AuditLogger{
		logFile: f,
		logPath: logPath,
		logger:  logger,
	}

	// Should not panic
	al.Close()

	// Should not panic on nil
	al2 := &AuditLogger{logger: logger}
	al2.Close()
}

func TestSplitLines(t *testing.T) {
	lines := splitLines("a\nb\nc")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "a" || lines[1] != "b" || lines[2] != "c" {
		t.Errorf("Lines = %v, want [a b c]", lines)
	}
}

func TestAccessLogEntry_JSON(t *testing.T) {
	entry := AccessLogEntry{
		ID:       "test-id",
		SourceIP: "127.0.0.1",
		Action:   "TEST",
		Status:   200,
		Details:  "test",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded AccessLogEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.ID != entry.ID {
		t.Errorf("ID mismatch: %q vs %q", decoded.ID, entry.ID)
	}
	if decoded.Status != entry.Status {
		t.Errorf("Status mismatch: %d vs %d", decoded.Status, entry.Status)
	}
}
