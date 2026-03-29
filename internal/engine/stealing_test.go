package engine

import (
	"testing"
)

func TestInflightTracker_StartAndComplete(t *testing.T) {
	tr := newInflightTracker()
	part := DownloadPart{ID: 0, StartOffset: 0, EndOffset: 1023}
	tr.Start(part)

	tr.mu.Lock()
	if _, ok := tr.parts[0]; !ok {
		t.Error("part 0 should be tracked")
	}
	tr.mu.Unlock()

	tr.Complete(0)
	tr.mu.Lock()
	if _, ok := tr.parts[0]; ok {
		t.Error("part 0 should be removed after Complete")
	}
	tr.mu.Unlock()
}

func TestInflightTracker_StealLargest(t *testing.T) {
	tr := newInflightTracker()

	// Create two in-flight parts: one small (almost done), one large
	small := DownloadPart{ID: 0, StartOffset: 0, EndOffset: 1024*1024 - 1}
	large := DownloadPart{ID: 1, StartOffset: 1024 * 1024, EndOffset: 10*1024*1024 - 1}

	tr.Start(small)
	tr.Start(large)
	tr.UpdateProgress(0, 900*1024)  // small: almost done
	tr.UpdateProgress(1, 1024*1024) // large: 1MB of 9MB done

	stolen, fromID := tr.StealLargest(100)
	if stolen == nil {
		t.Fatal("expected a stolen part")
	}
	if fromID != 1 {
		t.Errorf("expected steal from part 1 (largest), got %d", fromID)
	}
	if stolen.StartOffset <= large.StartOffset {
		t.Error("stolen part should start after the midpoint of the original")
	}
	if stolen.EndOffset != large.EndOffset {
		t.Errorf("stolen part end should match original end: got %d, want %d", stolen.EndOffset, large.EndOffset)
	}
}

func TestInflightTracker_StealNothingWhenTooSmall(t *testing.T) {
	tr := newInflightTracker()

	// A part that's only 500KB remaining — below the 1MB threshold
	tiny := DownloadPart{ID: 0, StartOffset: 0, EndOffset: 512*1024 - 1}
	tr.Start(tiny)

	stolen, _ := tr.StealLargest(100)
	if stolen != nil {
		t.Error("should not steal from a tiny in-flight part")
	}
}

func TestInflightTracker_StealNothingWhenEmpty(t *testing.T) {
	tr := newInflightTracker()
	stolen, _ := tr.StealLargest(100)
	if stolen != nil {
		t.Error("should not steal when tracker is empty")
	}
}

func TestInflightTracker_UpdateProgress(t *testing.T) {
	tr := newInflightTracker()
	part := DownloadPart{ID: 5, StartOffset: 0, EndOffset: 10*1024*1024 - 1}
	tr.Start(part)
	tr.UpdateProgress(5, 5*1024*1024)

	tr.mu.Lock()
	p := tr.parts[5]
	if p.bytesDownloaded != 5*1024*1024 {
		t.Errorf("expected 5MB downloaded, got %d", p.bytesDownloaded)
	}
	tr.mu.Unlock()
}
