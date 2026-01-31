# Project Tachyon API Reference

This document describes the Wails-exposed API methods for frontend integration.

## Download Control

### AddDownload(url string) string
Starts a new download with default settings. Returns the download ID.

### AddDownloadWithParams(url, path, filename string, options map[string]string) string
Starts a download with custom options:
- `headers`: Custom HTTP headers
- `cookies`: Custom cookies

### PauseDownload(id string)
Pauses an active download.

### ResumeDownload(id string) error
Resumes a paused, stopped, or error download.

### StopDownload(id string)
Stops a download permanently (state is preserved for manual resume).

### DeleteDownload(id string, deleteFile bool)
Deletes a download task and optionally removes the downloaded file.

---

## URL Refresh (403 Handling)

When a download receives HTTP 403 Forbidden, Tachyon pauses the task with status `needs_auth` and emits the `download:needs_auth` event.

### UpdateDownloadURL(taskID string, newURL string) error

Updates the URL for a download that requires authentication refresh.

**Usage Flow:**
1. Download receives 403 → Status becomes `needs_auth`
2. Frontend receives `download:needs_auth` event with task ID
3. User provides new URL (e.g., refreshed auth token)
4. Call `UpdateDownloadURL(taskID, newURL)` → Status becomes `paused`
5. Call `ResumeDownload(taskID)` to continue

**Events:**
- `download:needs_auth`: Emitted when 403 detected
  - `id`: Task ID
  - `reason`: Error description
- `download:url_updated`: Emitted when URL successfully updated
  - `id`: Task ID
  - `new_url`: The new URL

---

## Events Reference

### Download Events
| Event | Payload | Description |
|-------|---------|-------------|
| `download:started` | `{id, filename}` | Download began |
| `download:progress` | `{id, downloaded, speed, ...}` | Progress update |
| `download:completed` | `{id, path}` | Download finished |
| `download:paused` | `{id}` | Download paused |
| `download:error` | `{id, error}` | Download failed |
| `download:needs_auth` | `{id, reason}` | URL expired (403) |
| `download:url_updated` | `{id, new_url}` | URL refreshed |
