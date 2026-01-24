package core

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Category constants for file organization
const (
	CategoryVideo     = "Videos"
	CategoryMusic     = "Music"
	CategoryImages    = "Images"
	CategoryArchives  = "Archives"
	CategoryDocuments = "Documents"
	CategorySoftware  = "Software"
	CategoryOthers    = "Others"
)

// TachyonRootFolder is the main download folder name
const TachyonRootFolder = "Tachyon Downloads"

// Extension to category mapping
var extensionCategories = map[string]string{
	// Video
	".mp4": CategoryVideo, ".mkv": CategoryVideo, ".webm": CategoryVideo,
	".avi": CategoryVideo, ".mov": CategoryVideo, ".wmv": CategoryVideo,
	".flv": CategoryVideo, ".m4v": CategoryVideo,
	// Music
	".mp3": CategoryMusic, ".wav": CategoryMusic, ".flac": CategoryMusic,
	".aac": CategoryMusic, ".ogg": CategoryMusic, ".m4a": CategoryMusic,
	// Images
	".jpg": CategoryImages, ".jpeg": CategoryImages, ".png": CategoryImages,
	".gif": CategoryImages, ".webp": CategoryImages, ".bmp": CategoryImages,
	".svg": CategoryImages, ".ico": CategoryImages,
	// Archives
	".zip": CategoryArchives, ".rar": CategoryArchives, ".7z": CategoryArchives,
	".tar": CategoryArchives, ".gz": CategoryArchives, ".bz2": CategoryArchives,
	// Documents
	".pdf": CategoryDocuments, ".doc": CategoryDocuments, ".docx": CategoryDocuments,
	".xls": CategoryDocuments, ".xlsx": CategoryDocuments, ".ppt": CategoryDocuments,
	".pptx": CategoryDocuments, ".txt": CategoryDocuments, ".rtf": CategoryDocuments,
	".odt": CategoryDocuments, ".ods": CategoryDocuments,
	// Software
	".exe": CategorySoftware, ".msi": CategorySoftware, ".dmg": CategorySoftware,
	".deb": CategorySoftware, ".rpm": CategorySoftware, ".pkg": CategorySoftware,
	".appimage": CategorySoftware, ".apk": CategorySoftware,
}

// GetCategory returns the category based on file extension
func GetCategory(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if cat, ok := extensionCategories[ext]; ok {
		return cat
	}
	return CategoryOthers
}

// GetDefaultDownloadPath returns the default Tachyon Downloads path
// Cross-platform: ~/Downloads/Tachyon Downloads/
func GetDefaultDownloadPath() (string, error) {
	var downloadsDir string

	switch runtime.GOOS {
	case "windows":
		// Windows: Use USERPROFILE\Downloads
		userProfile := os.Getenv("USERPROFILE")
		if userProfile == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			userProfile = home
		}
		downloadsDir = filepath.Join(userProfile, "Downloads")
	case "darwin":
		// macOS: Use ~/Downloads
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		downloadsDir = filepath.Join(home, "Downloads")
	default:
		// Linux/Other: Check XDG_DOWNLOAD_DIR or fallback to ~/Downloads
		xdgDownload := os.Getenv("XDG_DOWNLOAD_DIR")
		if xdgDownload != "" {
			downloadsDir = xdgDownload
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			downloadsDir = filepath.Join(home, "Downloads")
		}
	}

	// Create Tachyon root folder
	tachyonRoot := filepath.Join(downloadsDir, TachyonRootFolder)
	if err := os.MkdirAll(tachyonRoot, 0755); err != nil {
		return "", err
	}

	return tachyonRoot, nil
}

// GetOrganizedPath returns the full path including category subfolder
// baseDir should be the Tachyon root or a custom location
func GetOrganizedPath(baseDir, filename string) (string, error) {
	category := GetCategory(filename)
	categoryPath := filepath.Join(baseDir, category)

	// Create category subfolder if not exists
	if err := os.MkdirAll(categoryPath, 0755); err != nil {
		return "", err
	}

	return filepath.Join(categoryPath, filename), nil
}

// EnsureCategoryFolders pre-creates all category subfolders
func EnsureCategoryFolders(baseDir string) error {
	categories := []string{
		CategoryVideo, CategoryMusic, CategoryImages,
		CategoryArchives, CategoryDocuments, CategorySoftware, CategoryOthers,
	}
	for _, cat := range categories {
		path := filepath.Join(baseDir, cat)
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}
	return nil
}
