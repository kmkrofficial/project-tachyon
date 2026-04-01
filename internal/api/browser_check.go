package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"project-tachyon/internal/engine"
	"project-tachyon/internal/filesystem"
)

// CheckRequest is the payload for collision detection.
type CheckRequest struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
}

// CheckResponse describes the collision status for a requested download.
type CheckResponse struct {
	Status   string  `json:"status"`              // "clear", "downloading", "completed", "exists"
	TaskID   string  `json:"task_id,omitempty"`   // ID of the colliding task
	Filename string  `json:"filename,omitempty"`  // Resolved filename
	SavePath string  `json:"save_path,omitempty"` // Path on disk
	Progress float64 `json:"progress,omitempty"`  // Download progress (0-100)
	Size     int64   `json:"size,omitempty"`      // File size on disk or total size
}

func (s *ControlServer) handleBrowserCheck(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req CheckRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBody)).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL required", http.StatusBadRequest)
		return
	}

	resp := s.checkCollision(req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *ControlServer) checkCollision(req CheckRequest) CheckResponse {
	// 1. Check if URL is currently downloading or pending
	task, err := s.engine.GetTaskByURL(req.URL)
	if err == nil {
		switch task.Status {
		case "downloading", "pending", "probing", "scheduled", "merging", "verifying":
			return CheckResponse{
				Status:   "downloading",
				TaskID:   task.ID,
				Filename: task.Filename,
				SavePath: task.SavePath,
				Progress: task.Progress,
				Size:     task.TotalSize,
			}
		case "completed":
			// Check if the file still exists on disk
			if info, statErr := os.Stat(task.SavePath); statErr == nil {
				return CheckResponse{
					Status:   "completed",
					TaskID:   task.ID,
					Filename: task.Filename,
					SavePath: task.SavePath,
					Size:     info.Size(),
				}
			}
			// File was deleted — treat as clear
		}
	}

	// 2. Check if a file with the same name exists in the download directory
	if req.Filename != "" {
		filename := engine.SanitizeFilename(req.Filename)
		defaultPath, pathErr := filesystem.GetDefaultDownloadPath()
		if pathErr == nil {
			organized, _ := filesystem.GetOrganizedPath(defaultPath, filename)
			if info, statErr := os.Stat(organized); statErr == nil {
				return CheckResponse{
					Status:   "exists",
					Filename: filepath.Base(organized),
					SavePath: organized,
					Size:     info.Size(),
				}
			}
		}
	}

	return CheckResponse{Status: "clear"}
}
