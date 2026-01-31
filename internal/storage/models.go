package storage

import (
	"gorm.io/gorm"
)

// DownloadTask represents a download task in the database
type DownloadTask struct {
	ID            string         `gorm:"primaryKey" json:"id"`
	Filename      string         `json:"filename"`
	URL           string         `json:"url"`
	SavePath      string         `json:"save_path"`
	Status        string         `gorm:"index" json:"status"`          // downloading, completed, paused, error, pending
	Priority      int            `gorm:"default:1" json:"priority"`    // 0=Low, 1=Normal, 2=High
	QueueOrder    int            `gorm:"default:0" json:"queue_order"` // Sequential order in queue
	Category      string         `gorm:"index" json:"category"`
	TotalSize     int64          `json:"total_size"`
	Downloaded    int64          `json:"downloaded"`
	Progress      float64        `json:"progress"`
	Speed         float64        `json:"speed"` // bytes/sec
	TimeRemaining string         `json:"time_remaining"`
	MetaJSON      string         `json:"-"` // Store complex chunk data/headers as JSON
	FileExists    bool           `gorm:"-" json:"file_exists"`
	ExpectedHash  string         `json:"expected_hash"`
	HashAlgorithm string         `json:"hash_algorithm"`
	Headers       string         `json:"headers"`    // JSON serialized
	Cookies       string         `json:"cookies"`    // JSON serialized
	StartTime     string         `json:"start_time"` // ISO 8601 for scheduled start
	Domain        string         `json:"domain"`     // e.g. "google.com" for concurrency limits
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for DownloadTask
func (DownloadTask) TableName() string {
	return "download_tasks"
}

// PartState represents the state of a single download chunk
type PartState struct {
	Start    int64 `json:"s"`           // Start offset
	End      int64 `json:"e"`           // End offset
	Complete bool  `json:"c,omitempty"` // Is chunk fully downloaded and verified?
	Offset   int64 `json:"o,omitempty"` // Current write offset relative to Start (for clean pause)
}

// ResumeState represents the serialized resume data
type ResumeState struct {
	Version      int               `json:"v"`
	ETag         string            `json:"etag"`
	LastModified string            `json:"lm"`
	TotalSize    int64             `json:"total_size"`
	Parts        map[int]PartState `json:"parts"`
}

// DownloadLocation stores saved download locations with nicknames
type DownloadLocation struct {
	Path     string `gorm:"primaryKey" json:"path"`
	Nickname string `json:"nickname"` // e.g., "Gaming Drive", "SSD"
}

// TableName specifies the table name for DownloadLocation
func (DownloadLocation) TableName() string {
	return "download_locations"
}

// DailyStat tracks daily download statistics for analytics
type DailyStat struct {
	Date  string `gorm:"primaryKey"` // Format: "YYYY-MM-DD"
	Bytes int64  `gorm:"default:0"`  // Total bytes for this day
	Files int64  `gorm:"default:0"`  // Files completed this day
}

// TableName specifies the table name for DailyStat
func (DailyStat) TableName() string {
	return "daily_stats"
}

// AppSetting stores key-value application settings
type AppSetting struct {
	Key   string `gorm:"primaryKey"`
	Value string
}

// TableName specifies the table name for AppSetting
func (AppSetting) TableName() string {
	return "app_settings"
}

// SpeedTestHistory stores past speed test results
type SpeedTestHistory struct {
	ID             uint    `gorm:"primaryKey" json:"id"`
	DownloadSpeed  float64 `json:"download_mbps"`
	UploadSpeed    float64 `json:"upload_mbps"`
	Ping           int64   `json:"ping_ms"`
	Jitter         int64   `json:"jitter_ms"`
	ISP            string  `json:"isp"`
	ServerName     string  `json:"server_name"`
	ServerLocation string  `json:"server_location"`
	Timestamp      string  `json:"timestamp"`
}

// TableName specifies the table name for SpeedTestHistory
func (SpeedTestHistory) TableName() string {
	return "speed_test_history"
}

// Task is an alias for backward compatibility with existing code
// Deprecated: Use DownloadTask instead
type Task = DownloadTask
