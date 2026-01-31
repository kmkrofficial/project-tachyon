package core

import (
	"encoding/json"
	"fmt"
	"project-tachyon/internal/storage"
	"sync"
)

// StateManager handles persistence and validation of download state
type StateManager struct {
	mu sync.RWMutex
}

func NewStateManager() *StateManager {
	return &StateManager{}
}

// Load parses the MetaJSON from a task
func (sm *StateManager) Load(metaJSON string) (*storage.ResumeState, error) {
	if metaJSON == "" {
		return nil, nil // No state
	}

	var state storage.ResumeState
	if err := json.Unmarshal([]byte(metaJSON), &state); err != nil {
		return nil, fmt.Errorf("failed to parse state: %w", err)
	}
	return &state, nil
}

// Serialize converts state to JSON string
func (sm *StateManager) Serialize(state *storage.ResumeState) (string, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state.Version = 1 // Current version
	data, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Validate checks if remote headers match the stored state
func (sm *StateManager) Validate(state *storage.ResumeState, remoteHeaders map[string]string) bool {
	if state == nil {
		return true // No state to validate against implies new or fresh start is valid
	}

	// 1. ETag Check (Strong Validator)
	if state.ETag != "" {
		remoteETag := remoteHeaders["ETag"]
		if remoteETag != "" && remoteETag != state.ETag {
			return false
		}
	}

	// 2. Last-Modified Check (Weak Validator)
	if state.LastModified != "" {
		remoteLM := remoteHeaders["Last-Modified"]
		if remoteLM != "" && remoteLM != state.LastModified {
			return false
		}
	}

	// 3. Size Check (Sanity)
	// Note: Engine should handle Content-Length logic separately, but good to check state consistency
	return true
}

// CreateInitialState generates a fresh state for a new download
func (sm *StateManager) CreateInitialState(totalSize int64, etag, lastModified string) *storage.ResumeState {
	return &storage.ResumeState{
		Version:      1,
		ETag:         etag,
		LastModified: lastModified,
		TotalSize:    totalSize,
		Parts:        make(map[int]storage.PartState),
	}
}
