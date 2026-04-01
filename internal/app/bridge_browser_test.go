package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectBrowsers_ReturnsSlice(t *testing.T) {
	a := &App{}
	browsers := a.DetectBrowsers()

	// Should return a slice (possibly empty on CI, but never nil panic)
	if browsers == nil {
		// nil is acceptable — no browsers found
		browsers = []BrowserInfo{}
	}

	for _, b := range browsers {
		if b.Name == "" {
			t.Error("browser name should not be empty")
		}
		if b.Type != "chromium" && b.Type != "firefox" {
			t.Errorf("browser type = %q, want chromium or firefox", b.Type)
		}
		if b.Path == "" {
			t.Error("browser path should not be empty")
		}
	}
}

func TestGetNativeMessagingManifest_Chromium(t *testing.T) {
	a := &App{}
	raw := a.GetNativeMessagingManifest("chromium")

	var manifest map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &manifest); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if manifest["name"] != "com.tachyon.downloadmanager" {
		t.Errorf("name = %v, want com.tachyon.downloadmanager", manifest["name"])
	}
	if manifest["type"] != "stdio" {
		t.Errorf("type = %v, want stdio", manifest["type"])
	}
	if _, ok := manifest["allowed_origins"]; !ok {
		t.Error("chromium manifest should have allowed_origins")
	}
}

func TestGetNativeMessagingManifest_Firefox(t *testing.T) {
	a := &App{}
	raw := a.GetNativeMessagingManifest("firefox")

	var manifest map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &manifest); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := manifest["allowed_extensions"]; !ok {
		t.Error("firefox manifest should have allowed_extensions")
	}
}

func TestCheckManifestForTDM_Positive(t *testing.T) {
	// Create a temporary extension directory structure
	dir := t.TempDir()
	versionDir := filepath.Join(dir, "1.0.0")
	os.MkdirAll(versionDir, 0o755)

	manifest := map[string]string{
		"name":    "Tachyon Download Manager",
		"version": "1.1",
	}
	data, _ := json.Marshal(manifest)
	os.WriteFile(filepath.Join(versionDir, "manifest.json"), data, 0o644)

	if !checkManifestForTDM(dir) {
		t.Error("should detect TDM extension manifest")
	}
}

func TestCheckManifestForTDM_Negative(t *testing.T) {
	dir := t.TempDir()
	versionDir := filepath.Join(dir, "1.0.0")
	os.MkdirAll(versionDir, 0o755)

	manifest := map[string]string{
		"name":    "Some Other Extension",
		"version": "2.0",
	}
	data, _ := json.Marshal(manifest)
	os.WriteFile(filepath.Join(versionDir, "manifest.json"), data, 0o644)

	if checkManifestForTDM(dir) {
		t.Error("should not detect non-TDM extension")
	}
}

func TestCheckManifestForTDM_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	if checkManifestForTDM(dir) {
		t.Error("should return false for empty dir")
	}
}

func TestBrowserInfo_JSON(t *testing.T) {
	b := BrowserInfo{
		Name:            "Google Chrome",
		Type:            "chromium",
		Path:            "/usr/bin/google-chrome",
		Version:         "120.0.0",
		ExtensionLoaded: true,
	}

	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded BrowserInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Name != b.Name || decoded.Type != b.Type || decoded.ExtensionLoaded != b.ExtensionLoaded {
		t.Errorf("round-trip mismatch: got %+v", decoded)
	}
}
