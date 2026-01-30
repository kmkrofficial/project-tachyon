package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// Storage handles all database operations using SQLite
type Storage struct {
	DB *gorm.DB
}

// NewStorage initializes the SQLite database connection
func NewStorage() (*Storage, error) {
	// Get app data directory
	appData, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config dir: %w", err)
	}

	dbDir := filepath.Join(appData, "Tachyon")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db dir: %w", err)
	}

	dbPath := filepath.Join(dbDir, "tachyon.db")

	// Open SQLite with Glebarez (Pure Go, no CGO)
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	db.Exec("PRAGMA journal_mode=WAL;")
	db.Exec("PRAGMA synchronous=NORMAL;")
	db.Exec("PRAGMA cache_size=10000;")

	// Auto-migrate tables
	err = db.AutoMigrate(
		&DownloadTask{},
		&DownloadLocation{},
		&DailyStat{},
		&AppSetting{},
		&SpeedTestHistory{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &Storage{DB: db}, nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	sqlDB, err := s.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Checkpoint forces a WAL checkpoint to ensure durability
func (s *Storage) Checkpoint() error {
	return s.DB.Exec("PRAGMA wal_checkpoint(TRUNCATE);").Error
}

// ============= Task Management =============

// SaveTask creates or updates a download task (upsert)
func (s *Storage) SaveTask(task DownloadTask) error {
	task.UpdatedAt = time.Now().Format(time.RFC3339)
	return s.DB.Save(&task).Error
}

// GetTask retrieves a specific task by ID
func (s *Storage) GetTask(id string) (DownloadTask, error) {
	var task DownloadTask
	err := s.DB.First(&task, "id = ?", id).Error
	return task, err
}

// GetTaskByURL retrieves a task by URL (to check history)
func (s *Storage) GetTaskByURL(url string) (DownloadTask, error) {
	var task DownloadTask
	// We want the most recent one if duplicates exist?
	err := s.DB.Where("url = ?", url).Order("created_at desc").First(&task).Error
	return task, err
}

// GetAllTasks returns all non-deleted tasks, newest first
// GetAllTasks returns all non-deleted tasks, newest first
func (s *Storage) GetAllTasks() ([]DownloadTask, error) {
	var tasks []DownloadTask
	err := s.DB.Order("created_at desc").Find(&tasks).Error
	return tasks, err
}

// GetTasksByStatus returns tasks filtered by status
func (s *Storage) GetTasksByStatus(status string, limit int) ([]DownloadTask, error) {
	var tasks []DownloadTask
	query := s.DB.Where("status = ?", status).Order("created_at desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&tasks).Error
	return tasks, err
}

// GetActiveTasks returns all downloading or pending tasks
func (s *Storage) GetActiveTasks() ([]DownloadTask, error) {
	var tasks []DownloadTask
	err := s.DB.Where("status IN ?", []string{"downloading", "pending"}).
		Order("priority desc, created_at asc").
		Find(&tasks).Error
	return tasks, err
}

// DeleteTask soft-deletes a task
func (s *Storage) DeleteTask(id string) error {
	return s.DB.Delete(&DownloadTask{}, "id = ?", id).Error
}

// UpdateTaskStatus updates just the status field
func (s *Storage) UpdateTaskStatus(id, status string) error {
	return s.DB.Model(&DownloadTask{}).Where("id = ?", id).Update("status", status).Error
}

// UpdateTaskProgress updates progress and speed for a task
func (s *Storage) UpdateTaskProgress(id string, progress float64, downloaded int64, speed float64) error {
	return s.DB.Model(&DownloadTask{}).Where("id = ?", id).Updates(map[string]interface{}{
		"progress":   progress,
		"downloaded": downloaded,
		"speed":      speed,
		"updated_at": time.Now(),
	}).Error
}

// ============= Download Locations =============

// AddLocation adds or updates a download location
func (s *Storage) AddLocation(path, nickname string) error {
	loc := DownloadLocation{Path: path, Nickname: nickname}
	return s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "path"}},
		DoUpdates: clause.AssignmentColumns([]string{"nickname"}),
	}).Create(&loc).Error
}

// GetLocations returns all saved download locations
func (s *Storage) GetLocations() ([]DownloadLocation, error) {
	var locations []DownloadLocation
	err := s.DB.Find(&locations).Error
	return locations, err
}

// DeleteLocation removes a download location
func (s *Storage) DeleteLocation(path string) error {
	return s.DB.Delete(&DownloadLocation{}, "path = ?", path).Error
}

// ============= Statistics (SQL Analytics) =============

// IncrementStat atomically increments today's download bytes and optionally files
func (s *Storage) IncrementStat(key string, bytes int64) error {
	today := time.Now().Format("2006-01-02")

	// Use upsert with SQL increment
	return s.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "date"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"bytes": gorm.Expr("bytes + ?", bytes),
		}),
	}).Create(&DailyStat{Date: today, Bytes: bytes}).Error
}

// IncrementDailyBytes adds bytes to today's stats
func (s *Storage) IncrementDailyBytes(bytes int64) error {
	today := time.Now().Format("2006-01-02")
	return s.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "date"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"bytes": gorm.Expr("bytes + ?", bytes),
		}),
	}).Create(&DailyStat{Date: today, Bytes: bytes}).Error
}

// IncrementDailyFiles adds a file count to today's stats
func (s *Storage) IncrementDailyFiles() error {
	today := time.Now().Format("2006-01-02")
	return s.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "date"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"files": gorm.Expr("files + 1"),
		}),
	}).Create(&DailyStat{Date: today, Files: 1}).Error
}

// GetTotalLifetime returns total bytes downloaded all-time using SQL SUM
func (s *Storage) GetTotalLifetime() (int64, error) {
	var total int64
	err := s.DB.Model(&DailyStat{}).Select("IFNULL(SUM(bytes), 0)").Row().Scan(&total)
	return total, err
}

// GetTotalFiles returns total files downloaded all-time using SQL SUM
func (s *Storage) GetTotalFiles() (int64, error) {
	var total int64
	err := s.DB.Model(&DailyStat{}).Select("IFNULL(SUM(files), 0)").Row().Scan(&total)
	return total, err
}

// GetDailyHistory returns the last N days of stats
func (s *Storage) GetDailyHistory(days int) ([]DailyStat, error) {
	var stats []DailyStat
	err := s.DB.Order("date desc").Limit(days).Find(&stats).Error
	return stats, err
}

// GetStatInt returns a stat value (for backward compatibility)
func (s *Storage) GetStatInt(key string) (int64, error) {
	if key == "stat_total_lifetime" {
		return s.GetTotalLifetime()
	}
	if key == "stat_total_files" {
		return s.GetTotalFiles()
	}
	// For daily stats, parse the date from key
	return 0, nil
}

// ============= App Settings =============

// GetString retrieves a string setting by key
func (s *Storage) GetString(key string) (string, error) {
	var setting AppSetting
	err := s.DB.First(&setting, "key = ?", key).Error
	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	return setting.Value, err
}

// SetString stores a string setting
func (s *Storage) SetString(key, value string) error {
	return s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(&AppSetting{Key: key, Value: value}).Error
}

// GetStringList retrieves a comma-separated list as slice
func (s *Storage) GetStringList(key string) ([]string, error) {
	val, err := s.GetString(key)
	if err != nil || val == "" {
		return []string{}, err
	}
	// Simple split by comma
	var result []string
	for _, item := range splitAndTrim(val) {
		if item != "" {
			result = append(result, item)
		}
	}
	return result, nil
}

// SetStringList stores a slice as comma-separated string
func (s *Storage) SetStringList(key string, list []string) error {
	val := joinWithComma(list)
	return s.SetString(key, val)
}

// Helper functions
func splitAndTrim(s string) []string {
	var result []string
	current := ""
	for _, c := range s {
		if c == ',' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func joinWithComma(list []string) string {
	result := ""
	for i, item := range list {
		if i > 0 {
			result += ","
		}
		result += item
	}
	return result
}

// ============= Speed Test History =============

// SaveSpeedTest saves a speed test result
func (s *Storage) SaveSpeedTest(history SpeedTestHistory) error {
	return s.DB.Create(&history).Error
}

// GetSpeedTestHistory returns the last N speed tests
func (s *Storage) GetSpeedTestHistory(limit int) ([]SpeedTestHistory, error) {
	var history []SpeedTestHistory
	err := s.DB.Order("timestamp desc").Limit(limit).Find(&history).Error
	return history, err
}
