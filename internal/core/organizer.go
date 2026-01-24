package core

import (
	"os"
	"path/filepath"
	"strings"
)

type Category string

const (
	CategoryVideo    Category = "Video"
	CategoryMusic    Category = "Music"
	CategoryImage    Category = "Images"
	CategoryArchives Category = "Archives"
	CategoryDocs     Category = "Documents"
	CategoryPrograms Category = "Programs"
	CategoryOther    Category = "Others"
)

var (
	extMap = map[string]Category{
		// Video
		".mp4": CategoryVideo, ".mkv": CategoryVideo, ".webm": CategoryVideo, ".avi": CategoryVideo, ".mov": CategoryVideo,
		// Music
		".mp3": CategoryMusic, ".wav": CategoryMusic, ".flac": CategoryMusic, ".aac": CategoryMusic,
		// Images
		".jpg": CategoryImage, ".jpeg": CategoryImage, ".png": CategoryImage, ".gif": CategoryImage, ".webp": CategoryImage,
		// Archives
		".zip": CategoryArchives, ".rar": CategoryArchives, ".7z": CategoryArchives, ".tar": CategoryArchives, ".gz": CategoryArchives,
		// Docs
		".pdf": CategoryDocs, ".doc": CategoryDocs, ".docx": CategoryDocs, ".xls": CategoryDocs, ".xlsx": CategoryDocs, ".txt": CategoryDocs,
		// Programs
		".exe": CategoryPrograms, ".msi": CategoryPrograms, ".dmg": CategoryPrograms, ".deb": CategoryPrograms,
	}
	// Correcting typo in map definition during writing: using literal strings or updating consts
	// I will use string literals or fix the const names below
)

// Re-defining properly locally to ensure compilation
var extensionCategories = map[string]string{
	".mp4": "Video", ".mkv": "Video", ".webm": "Video", ".avi": "Video", ".mov": "Video",
	".mp3": "Music", ".wav": "Music", ".flac": "Music", ".aac": "Music",
	".jpg": "Images", ".jpeg": "Images", ".png": "Images", ".gif": "Images", ".webp": "Images",
	".zip": "Archives", ".rar": "Archives", ".7z": "Archives", ".tar": "Archives", ".gz": "Archives",
	".pdf": "Documents", ".doc": "Documents", ".docx": "Documents", ".txt": "Documents",
	".exe": "Programs", ".msi": "Programs", ".dmg": "Programs", ".deb": "Programs",
}

// GetCategory returns the category based on file extension
func GetCategory(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if cat, ok := extensionCategories[ext]; ok {
		return cat
	}
	return "Others"
}

// GetOrganizedPath returns the full path including category folder
func GetOrganizedPath(baseDir, filename string) (string, error) {
	category := GetCategory(filename)
	categoryPath := filepath.Join(baseDir, category)

	// Create Category Folder if not exists
	if err := os.MkdirAll(categoryPath, 0755); err != nil {
		return "", err
	}

	return filepath.Join(categoryPath, filename), nil
}
