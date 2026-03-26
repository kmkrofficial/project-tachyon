package engine

import (
	"testing"
)

func TestDownloadPartType(t *testing.T) {
	part := DownloadPart{
		ID:          1,
		StartOffset: 0,
		EndOffset:   1024,
		Attempts:    0,
	}

	if part.ID != 1 {
		t.Errorf("ID = %d, want 1", part.ID)
	}
	if part.StartOffset != 0 {
		t.Errorf("StartOffset = %d, want 0", part.StartOffset)
	}
	if part.EndOffset != 1024 {
		t.Errorf("EndOffset = %d, want 1024", part.EndOffset)
	}
	if part.Attempts != 0 {
		t.Errorf("Attempts = %d, want 0", part.Attempts)
	}
}

func TestActiveDownloadInfo(t *testing.T) {
	// Test that activeDownloadInfo can be created with a cancel func
	cancelled := false
	info := &activeDownloadInfo{
		Cancel: func() { cancelled = true },
	}

	if info.Cancel == nil {
		t.Fatal("Cancel should not be nil")
	}

	info.Cancel()
	if !cancelled {
		t.Error("Cancel function was not called")
	}
}
