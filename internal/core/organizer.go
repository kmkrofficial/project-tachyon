package core

import (
	"fmt"
	"os"
	"path/filepath"
	"project-tachyon/internal/storage"
	"strings"
)

// SmartOrganizer handles automatic file categorization and moving
type SmartOrganizer struct {
	enableSmartSorting bool
}

func NewSmartOrganizer() *SmartOrganizer {
	return &SmartOrganizer{
		enableSmartSorting: true, // Default true, later from config
	}
}

// GetCategory returns the category for a given filename based on extension
func GetCategory(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	// Images
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg":
		return "Images"
	// Videos
	case ".mp4", ".mkv", ".mov", ".avi", ".webm", ".wmv":
		return "Videos"
	// Music
	case ".mp3", ".wav", ".flac", ".aac", ".ogg", ".m4a":
		return "Music"
	// Archives
	case ".zip", ".rar", ".7z", ".tar", ".gz", ".iso":
		return "Archives"
	// Documents
	case ".pdf", ".docx", ".xlsx", ".pptx", ".txt", ".md":
		return "Documents"
	// Software
	case ".exe", ".msi", ".dmg", ".pkg", ".deb":
		return "Software"
	default:
		return "Others"
	}
}

// GetOrganizedPath returns the full path where the file should be stored
func GetOrganizedPath(baseDir, filename string) (string, error) {
	category := GetCategory(filename)
	return filepath.Join(baseDir, category, filename), nil
}

// OrganizeFile moves the completed download to a categorized subfolder
func (o *SmartOrganizer) OrganizeFile(task *storage.DownloadTask) (string, error) {
	if !o.enableSmartSorting {
		return task.SavePath, nil
	}

	category := GetCategory(task.Filename)

	// Base directory is the parent of current SavePath
	baseDir := filepath.Dir(task.SavePath)

	targetDir := filepath.Join(baseDir, category)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return task.SavePath, fmt.Errorf("failed to create category dir: %w", err)
	}

	targetPath := filepath.Join(targetDir, task.Filename)
	targetPath = o.findAvailablePath(targetPath)

	if err := os.Rename(task.SavePath, targetPath); err != nil {
		return task.SavePath, fmt.Errorf("failed to move file: %w", err)
	}

	return targetPath, nil
}

func (o *SmartOrganizer) findAvailablePath(basePath string) string {
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return basePath
	}
	ext := filepath.Ext(basePath)

	dir := filepath.Dir(basePath)
	filename := filepath.Base(basePath)
	nameOnly := strings.TrimSuffix(filename, ext)

	for i := 1; i < 1000; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", nameOnly, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	// Fallback
	return filepath.Join(dir, fmt.Sprintf("%s_%d%s", nameOnly, 9999, ext))
}
