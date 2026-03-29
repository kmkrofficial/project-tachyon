package engine

import (
	"sync"
)

// inflightTracker tracks which parts are currently being downloaded,
// enabling idle workers to "steal" and bisect slow in-flight parts.
type inflightTracker struct {
	mu    sync.Mutex
	parts map[int]*inflightPart
}

type inflightPart struct {
	part            DownloadPart
	bytesDownloaded int64 // Approximate — updated by worker
}

func newInflightTracker() *inflightTracker {
	return &inflightTracker{parts: make(map[int]*inflightPart)}
}

// Start marks a part as in-flight.
func (t *inflightTracker) Start(part DownloadPart) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.parts[part.ID] = &inflightPart{part: part}
}

// Complete removes a part from the in-flight set.
func (t *inflightTracker) Complete(id int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.parts, id)
}

// UpdateProgress updates approximate bytes downloaded for an in-flight part.
func (t *inflightTracker) UpdateProgress(id int, downloaded int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if p, ok := t.parts[id]; ok {
		p.bytesDownloaded = downloaded
	}
}

// StealLargest finds the in-flight part with the most remaining bytes and
// bisects it. Returns the new part (second half) and the original part ID,
// or nil if no stealable part exists.
// Minimum stealable remainder is 1 MB to avoid micro-parts.
func (t *inflightTracker) StealLargest(nextID int) (*DownloadPart, int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	const minStealBytes = 1024 * 1024 // 1 MB

	var bestID int
	var bestRemaining int64
	var bestPart *inflightPart

	for id, p := range t.parts {
		if p.part.EndOffset == StreamEndOffset {
			continue
		}
		total := p.part.EndOffset - p.part.StartOffset + 1
		remaining := total - p.bytesDownloaded
		if remaining > bestRemaining {
			bestRemaining = remaining
			bestID = id
			bestPart = p
		}
	}

	if bestPart == nil || bestRemaining < 2*minStealBytes {
		return nil, 0
	}

	// Bisect: shrink original, return second half
	originalEnd := bestPart.part.EndOffset
	midpoint := bestPart.part.StartOffset + bestPart.bytesDownloaded + (bestRemaining / 2)
	bestPart.part.EndOffset = midpoint

	stolen := DownloadPart{
		ID:          nextID,
		StartOffset: midpoint + 1,
		EndOffset:   originalEnd,
		Attempts:    0,
	}
	t.parts[nextID] = &inflightPart{part: stolen}

	return &stolen, bestID
}
