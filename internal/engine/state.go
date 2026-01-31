package engine

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

// ===== Bitfield Helpers for Efficient State Serialization =====

// CompletedPartsToBitfield converts a map[int]bool of completed parts to a compact []byte bitmap
// This is O(n) where n is numParts, and produces ceil(numParts/8) bytes
// Bit i is set if part i is complete
func CompletedPartsToBitfield(completedParts map[int]bool, numParts int) []byte {
	if numParts <= 0 {
		return nil
	}

	// Calculate number of bytes needed
	numBytes := (numParts + 7) / 8
	bitfield := make([]byte, numBytes)

	for partID := range completedParts {
		if partID >= 0 && partID < numParts {
			byteIdx := partID / 8
			bitIdx := uint(partID % 8)
			bitfield[byteIdx] |= (1 << bitIdx)
		}
	}

	return bitfield
}

// BitfieldToCompletedParts converts a []byte bitmap back to map[int]bool
// Only includes parts that are marked as complete (bit set to 1)
func BitfieldToCompletedParts(bitfield []byte, numParts int) map[int]bool {
	result := make(map[int]bool)

	if len(bitfield) == 0 || numParts <= 0 {
		return result
	}

	for partID := 0; partID < numParts; partID++ {
		byteIdx := partID / 8
		if byteIdx >= len(bitfield) {
			break
		}
		bitIdx := uint(partID % 8)
		if (bitfield[byteIdx] & (1 << bitIdx)) != 0 {
			result[partID] = true
		}
	}

	return result
}

// CountCompletedParts quickly counts the number of completed parts from a bitfield
// Uses population count for efficiency
func CountCompletedParts(bitfield []byte) int {
	count := 0
	for _, b := range bitfield {
		count += popCount(b)
	}
	return count
}

// popCount counts the number of set bits in a byte (Hamming weight)
func popCount(b byte) int {
	count := 0
	for b != 0 {
		count += int(b & 1)
		b >>= 1
	}
	return count
}

// CompactResumeState is an optimized version of ResumeState using bitfield
// This reduces storage size from O(n * sizeof(PartState)) to O(n/8) bytes
type CompactResumeState struct {
	Version         int    `json:"v"`
	ETag            string `json:"etag,omitempty"`
	LastModified    string `json:"lm,omitempty"`
	TotalSize       int64  `json:"total_size"`
	NumParts        int    `json:"num_parts"`
	CompletedBitmap []byte `json:"bitmap,omitempty"` // Base64 encoded in JSON
}

// ToCompact converts a ResumeState to CompactResumeState
func (sm *StateManager) ToCompact(state *storage.ResumeState, numParts int) *CompactResumeState {
	if state == nil {
		return nil
	}

	// Convert parts map to completedParts map[int]bool
	completedParts := make(map[int]bool)
	for id, part := range state.Parts {
		if part.Complete {
			completedParts[id] = true
		}
	}

	return &CompactResumeState{
		Version:         state.Version,
		ETag:            state.ETag,
		LastModified:    state.LastModified,
		TotalSize:       state.TotalSize,
		NumParts:        numParts,
		CompletedBitmap: CompletedPartsToBitfield(completedParts, numParts),
	}
}

// FromCompact converts a CompactResumeState back to ResumeState
func (sm *StateManager) FromCompact(compact *CompactResumeState) *storage.ResumeState {
	if compact == nil {
		return nil
	}

	completedParts := BitfieldToCompletedParts(compact.CompletedBitmap, compact.NumParts)

	parts := make(map[int]storage.PartState)
	for id := range completedParts {
		parts[id] = storage.PartState{Complete: true}
	}

	return &storage.ResumeState{
		Version:      compact.Version,
		ETag:         compact.ETag,
		LastModified: compact.LastModified,
		TotalSize:    compact.TotalSize,
		Parts:        parts,
	}
}

// SerializeCompact converts state to compact JSON format
func (sm *StateManager) SerializeCompact(state *storage.ResumeState, numParts int) (string, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	compact := sm.ToCompact(state, numParts)
	compact.Version = 2 // Mark as compact format

	data, err := json.Marshal(compact)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
