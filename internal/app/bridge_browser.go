package app

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// BrowserInfo describes a detected browser installation.
type BrowserInfo struct {
	Name            string `json:"name"`
	Type            string `json:"type"`    // "chromium" or "firefox"
	Path            string `json:"path"`    // executable path
	Version         string `json:"version"` // detected version string
	ExtensionDir    string `json:"extensionDir"`
	ExtensionLoaded bool   `json:"extensionLoaded"`
}

// DetectBrowsers scans the system for installed Chromium and Firefox browsers
// and returns info about each, including whether the TDM extension is loaded.
func (a *App) DetectBrowsers() []BrowserInfo {
	var browsers []BrowserInfo

	switch runtime.GOOS {
	case "windows":
		browsers = detectBrowsersWindows()
	case "darwin":
		browsers = detectBrowsersDarwin()
	case "linux":
		browsers = detectBrowsersLinux()
	}

	// Check if extension is already loaded
	for i := range browsers {
		browsers[i].ExtensionLoaded = isExtensionInstalled(browsers[i])
	}

	return browsers
}

// GetNativeMessagingManifest generates the native messaging host manifest JSON
// for a specific browser type. The manifest tells the browser how to launch TDM.
func (a *App) GetNativeMessagingManifest(browserType string) string {
	exePath, err := os.Executable()
	if err != nil {
		return "{}"
	}

	manifest := map[string]interface{}{
		"name":        "com.tachyon.downloadmanager",
		"description": "Tachyon Download Manager Native Host",
		"path":        exePath,
		"type":        "stdio",
	}

	if browserType == "firefox" {
		manifest["allowed_extensions"] = []string{"tachyon-download-manager@tachyon.dev"}
	} else {
		manifest["allowed_origins"] = []string{
			"chrome-extension://*/",
		}
	}

	data, _ := json.MarshalIndent(manifest, "", "  ")
	return string(data)
}

// InstallNativeMessagingHost writes the native messaging host manifest for
// the given browser so TDM can be launched from the extension.
func (a *App) InstallNativeMessagingHost(browserType string) error {
	manifest := a.GetNativeMessagingManifest(browserType)
	dir := nativeMessagingDir(browserType)
	if dir == "" {
		return fmt.Errorf("unsupported browser type: %s", browserType)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	path := filepath.Join(dir, "com.tachyon.downloadmanager.json")
	return os.WriteFile(path, []byte(manifest), 0o644)
}

// ─── Windows ──────────────────────────────────────────────────────────────────

func detectBrowsersWindows() []BrowserInfo {
	type candidate struct {
		name     string
		btype    string
		envVar   string
		relative string
	}

	localAppData := os.Getenv("LOCALAPPDATA")
	programFiles := os.Getenv("PROGRAMFILES")
	programFilesX86 := os.Getenv("PROGRAMFILES(X86)")

	candidates := []candidate{
		{"Google Chrome", "chromium", localAppData, `Google\Chrome\Application\chrome.exe`},
		{"Google Chrome", "chromium", programFiles, `Google\Chrome\Application\chrome.exe`},
		{"Microsoft Edge", "chromium", programFilesX86, `Microsoft\Edge\Application\msedge.exe`},
		{"Microsoft Edge", "chromium", programFiles, `Microsoft\Edge\Application\msedge.exe`},
		{"Brave", "chromium", localAppData, `BraveSoftware\Brave-Browser\Application\brave.exe`},
		{"Vivaldi", "chromium", localAppData, `Vivaldi\Application\vivaldi.exe`},
		{"Opera", "chromium", localAppData, `Programs\Opera\opera.exe`},
		{"Firefox", "firefox", programFiles, `Mozilla Firefox\firefox.exe`},
		{"Firefox", "firefox", programFilesX86, `Mozilla Firefox\firefox.exe`},
	}

	seen := make(map[string]bool)
	var browsers []BrowserInfo

	for _, c := range candidates {
		if c.envVar == "" {
			continue
		}
		full := filepath.Join(c.envVar, c.relative)
		if _, err := os.Stat(full); err != nil {
			continue
		}
		if seen[full] {
			continue
		}
		seen[full] = true

		browsers = append(browsers, BrowserInfo{
			Name:         c.name,
			Type:         c.btype,
			Path:         full,
			Version:      getFileVersion(full),
			ExtensionDir: chromiumExtensionDir(c.name),
		})
	}
	return browsers
}

// ─── macOS ────────────────────────────────────────────────────────────────────

func detectBrowsersDarwin() []BrowserInfo {
	type candidate struct {
		name  string
		btype string
		path  string
	}

	candidates := []candidate{
		{"Google Chrome", "chromium", "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"},
		{"Microsoft Edge", "chromium", "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"},
		{"Brave", "chromium", "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser"},
		{"Vivaldi", "chromium", "/Applications/Vivaldi.app/Contents/MacOS/Vivaldi"},
		{"Opera", "chromium", "/Applications/Opera.app/Contents/MacOS/Opera"},
		{"Firefox", "firefox", "/Applications/Firefox.app/Contents/MacOS/firefox"},
	}

	var browsers []BrowserInfo
	for _, c := range candidates {
		if _, err := os.Stat(c.path); err != nil {
			continue
		}
		browsers = append(browsers, BrowserInfo{
			Name:         c.name,
			Type:         c.btype,
			Path:         c.path,
			ExtensionDir: chromiumExtensionDir(c.name),
		})
	}
	return browsers
}

// ─── Linux ────────────────────────────────────────────────────────────────────

func detectBrowsersLinux() []BrowserInfo {
	type candidate struct {
		name   string
		btype  string
		binary string
	}

	candidates := []candidate{
		{"Google Chrome", "chromium", "google-chrome"},
		{"Google Chrome (Stable)", "chromium", "google-chrome-stable"},
		{"Chromium", "chromium", "chromium-browser"},
		{"Chromium", "chromium", "chromium"},
		{"Microsoft Edge", "chromium", "microsoft-edge"},
		{"Brave", "chromium", "brave-browser"},
		{"Vivaldi", "chromium", "vivaldi"},
		{"Opera", "chromium", "opera"},
		{"Firefox", "firefox", "firefox"},
	}

	seen := make(map[string]bool)
	var browsers []BrowserInfo

	for _, c := range candidates {
		path, err := exec.LookPath(c.binary)
		if err != nil {
			continue
		}
		if seen[path] {
			continue
		}
		seen[path] = true

		browsers = append(browsers, BrowserInfo{
			Name:         c.name,
			Type:         c.btype,
			Path:         path,
			ExtensionDir: chromiumExtensionDir(c.name),
		})
	}
	return browsers
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func chromiumExtensionDir(browserName string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	name := strings.ToLower(browserName)
	switch runtime.GOOS {
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		switch {
		case strings.Contains(name, "chrome"):
			return filepath.Join(localAppData, `Google\Chrome\User Data\Default\Extensions`)
		case strings.Contains(name, "edge"):
			return filepath.Join(localAppData, `Microsoft\Edge\User Data\Default\Extensions`)
		case strings.Contains(name, "brave"):
			return filepath.Join(localAppData, `BraveSoftware\Brave-Browser\User Data\Default\Extensions`)
		case strings.Contains(name, "vivaldi"):
			return filepath.Join(localAppData, `Vivaldi\User Data\Default\Extensions`)
		case strings.Contains(name, "opera"):
			return filepath.Join(os.Getenv("APPDATA"), `Opera Software\Opera Stable\Extensions`)
		}
	case "darwin":
		switch {
		case strings.Contains(name, "chrome"):
			return filepath.Join(home, "Library/Application Support/Google/Chrome/Default/Extensions")
		case strings.Contains(name, "edge"):
			return filepath.Join(home, "Library/Application Support/Microsoft Edge/Default/Extensions")
		case strings.Contains(name, "brave"):
			return filepath.Join(home, "Library/Application Support/BraveSoftware/Brave-Browser/Default/Extensions")
		}
	case "linux":
		switch {
		case strings.Contains(name, "chrome"):
			return filepath.Join(home, ".config/google-chrome/Default/Extensions")
		case strings.Contains(name, "chromium"):
			return filepath.Join(home, ".config/chromium/Default/Extensions")
		case strings.Contains(name, "edge"):
			return filepath.Join(home, ".config/microsoft-edge/Default/Extensions")
		case strings.Contains(name, "brave"):
			return filepath.Join(home, ".config/BraveSoftware/Brave-Browser/Default/Extensions")
		}
	}
	return ""
}

func nativeMessagingDir(browserType string) string {
	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "windows":
		// Windows uses registry, but we can also use the file-based approach
		appData := os.Getenv("LOCALAPPDATA")
		if browserType == "firefox" {
			return filepath.Join(appData, "Mozilla", "NativeMessagingHosts")
		}
		return filepath.Join(appData, "Google", "Chrome", "NativeMessagingHosts")
	case "darwin":
		if browserType == "firefox" {
			return filepath.Join(home, "Library/Application Support/Mozilla/NativeMessagingHosts")
		}
		return filepath.Join(home, "Library/Application Support/Google/Chrome/NativeMessagingHosts")
	case "linux":
		if browserType == "firefox" {
			return filepath.Join(home, ".mozilla/native-messaging-hosts")
		}
		return filepath.Join(home, ".config/google-chrome/NativeMessagingHosts")
	}
	return ""
}

func isExtensionInstalled(b BrowserInfo) bool {
	if b.ExtensionDir == "" {
		return false
	}
	// Check if extension dir exists and has any TDM-related content
	entries, err := os.ReadDir(b.ExtensionDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		// Check each extension folder for our manifest
		manifestPath := filepath.Join(b.ExtensionDir, e.Name())
		if checkManifestForTDM(manifestPath) {
			return true
		}
	}
	return false
}

func checkManifestForTDM(dir string) bool {
	// Walk version subdirectories to find manifest.json with our extension name
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		mPath := filepath.Join(dir, e.Name(), "manifest.json")
		data, err := os.ReadFile(mPath)
		if err != nil {
			continue
		}
		var manifest struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(data, &manifest) == nil {
			if strings.Contains(strings.ToLower(manifest.Name), "tachyon") {
				return true
			}
		}
	}
	return false
}

func getFileVersion(path string) string {
	// Basic version detection — try running with --version flag
	if runtime.GOOS != "windows" {
		out, err := exec.Command(path, "--version").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	return ""
}
