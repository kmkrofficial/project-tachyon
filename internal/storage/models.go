package storage

import (
	"time"

	"gorm.io/gorm"
)

// DownloadTask represents a download task in the database
type DownloadTask struct {
	ID            string         `gorm:"primaryKey" json:"id"`
	Filename      string         `json:"filename"`
	URL           string         `json:"url"`
	SavePath      string         `json:"save_path"`
	Status        string         `gorm:"index" json:"status"`       // downloading, completed, paused, error, pending
	Priority      int            `gorm:"default:1" json:"priority"` // 0=Low, 1=Normal, 2=High
	Category      string         `gorm:"index" json:"category"`
	TotalSize     int64          `json:"total_size"`
	Downloaded    int64          `json:"downloaded"`
	Progress      float64        `json:"progress"`
	Speed         float64        `json:"speed"` // bytes/sec
	TimeRemaining string         `json:"time_remaining"`
	MetaJSON      string         `json:"-"` // Store complex chunk data/headers as JSON
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for DownloadTask
func (DownloadTask) TableName() string {
	return "download_tasks"
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

// Task is an alias for backward compatibility with existing code
// Deprecated: Use DownloadTask instead
type Task = DownloadTask
