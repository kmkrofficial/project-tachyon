package security

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type AccessLogEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	SourceIP  string    `json:"source_ip"`
	UserAgent string    `json:"user_agent"`
	Action    string    `json:"action"` // e.g., "POST /queue" or "MCP:download_file"
	Status    int       `json:"status"` // 200, 401, 403
	Details   string    `json:"details"`
}

type AuditLogger struct {
	ctx     context.Context
	logFile *os.File
	mu      sync.Mutex
	logPath string
	logger  *slog.Logger
}

func NewAuditLogger(logger *slog.Logger) *AuditLogger {
	appData, _ := os.UserConfigDir()
	logDir := filepath.Join(appData, "Tachyon", "logs")
	os.MkdirAll(logDir, 0755)

	path := filepath.Join(logDir, "mcp_access.log")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Error("Failed to open audit log", "error", err)
	}

	return &AuditLogger{
		logFile: f,
		logPath: path,
		logger:  logger,
	}
}

func (a *AuditLogger) SetContext(ctx context.Context) {
	a.ctx = ctx
}

func (a *AuditLogger) Log(sourceIP, userAgent, action string, status int, details string) {
	entry := AccessLogEntry{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		SourceIP:  sourceIP,
		UserAgent: userAgent,
		Action:    action,
		Status:    status,
		Details:   details,
	}

	// Write to file
	a.mu.Lock()
	if a.logFile != nil {
		jsonBytes, _ := json.Marshal(entry)
		a.logFile.WriteString(string(jsonBytes) + "\n")
	}
	a.mu.Unlock()

	// Emit to UI
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "onAuditLog", entry)
	}

	// Also log to system logger for dev debugging
	level := slog.LevelInfo
	if status >= 400 {
		level = slog.LevelWarn
	}
	a.logger.Log(context.Background(), level, "Audit", "action", action, "status", status, "ip", sourceIP)
}

func (a *AuditLogger) Close() {
	if a.logFile != nil {
		a.logFile.Close()
	}
}

// Helper to read recent logs for UI
func (a *AuditLogger) GetRecentLogs(limit int) []AccessLogEntry {
	a.mu.Lock()
	defer a.mu.Unlock()

	content, err := os.ReadFile(a.logPath)
	if err != nil {
		return []AccessLogEntry{}
	}

	lines := splitLines(string(content))
	var entries []AccessLogEntry

	// Parse valid JSON lines backwards
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		var entry AccessLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			entries = append(entries, entry)
		}
		if len(entries) >= limit {
			break
		}
	}
	return entries
}

func splitLines(s string) []string {
	// Simple split by newline
	return strings.Split(s, "\n")
}
