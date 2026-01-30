package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Release represents a GitHub release
type Release struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
}

// CheckForUpdates queries GitHub for the latest release
func CheckForUpdates(currentVersion string, owner, repo string) (*Release, error) {
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repo required")
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Project-Tachyon-Updater")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to check update: %d", resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}

	// Normalize versions (remove 'v' prefix)
	current := strings.TrimPrefix(currentVersion, "v")
	remote := strings.TrimPrefix(rel.TagName, "v")

	if current != remote {
		return &rel, nil
	}
	return nil, nil // No update
}
