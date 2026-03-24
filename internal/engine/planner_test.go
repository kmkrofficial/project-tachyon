package engine

import (
	"log/slog"
	"os"
	"testing"

	"project-tachyon/internal/network"
)

// newPlannerEngine creates a minimal TachyonEngine for testing planner functions.
func newPlannerEngine(maxWorkers int, baseChunk int64) *TachyonEngine {
	return &TachyonEngine{
		logger:            slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		congestion:        network.NewCongestionController(4, maxWorkers),
		maxWorkersPerTask: maxWorkers,
		baseChunkSize:     baseChunk,
	}
}

// --- planDownloadParts ---

func TestPlanDownloadParts_NoRangeSupport(t *testing.T) {
	e := newPlannerEngine(16, 0)
	parts := e.planDownloadParts(10*1024*1024, false)
	if len(parts) != 1 {
		t.Fatalf("expected 1 streaming part, got %d", len(parts))
	}
	if parts[0].EndOffset != StreamEndOffset {
		t.Error("single part should use StreamEndOffset")
	}
	if parts[0].StartOffset != 0 {
		t.Error("single part should start at 0")
	}
}

func TestPlanDownloadParts_ZeroSize(t *testing.T) {
	e := newPlannerEngine(16, 0)
	parts := e.planDownloadParts(0, true)
	if len(parts) != 1 {
		t.Fatalf("expected 1 streaming part for zero-size, got %d", len(parts))
	}
	if parts[0].EndOffset != StreamEndOffset {
		t.Error("zero-size should fallback to stream")
	}
}

func TestPlanDownloadParts_NegativeSize(t *testing.T) {
	e := newPlannerEngine(16, 0)
	parts := e.planDownloadParts(-1, true)
	if len(parts) != 1 {
		t.Fatalf("expected 1 streaming part for negative size, got %d", len(parts))
	}
}

func TestPlanDownloadParts_SmallFile(t *testing.T) {
	e := newPlannerEngine(16, 0)
	// 2MB file with 1MB chunks → expect ~2 parts
	size := int64(2 * 1024 * 1024)
	parts := e.planDownloadParts(size, true)

	// Verify coverage
	if parts[0].StartOffset != 0 {
		t.Error("first part must start at 0")
	}
	last := parts[len(parts)-1]
	if last.EndOffset != size-1 {
		t.Errorf("last part end=%d, want=%d", last.EndOffset, size-1)
	}

	// Verify contiguity
	for i := 1; i < len(parts); i++ {
		if parts[i].StartOffset != parts[i-1].EndOffset+1 {
			t.Fatalf("gap between part %d and %d", i-1, i)
		}
	}
}

func TestPlanDownloadParts_LargeFile(t *testing.T) {
	e := newPlannerEngine(16, 0)
	// 500MB → chunk = 2MB, tail chunk = 512KB at 80%
	size := int64(500 * 1024 * 1024)
	parts := e.planDownloadParts(size, true)

	if len(parts) < 10 {
		t.Fatalf("expected many parts for 500MB, got %d", len(parts))
	}

	// Verify last part ends at size-1
	last := parts[len(parts)-1]
	if last.EndOffset != size-1 {
		t.Errorf("last part end=%d, want=%d", last.EndOffset, size-1)
	}

	// Tail chunks (past 80%) should be smaller
	tailStart := int64(float64(size) * 0.8)
	baseChunk := e.selectChunkSize(size) // 2MB
	var foundSmallerTail bool
	for _, p := range parts {
		if p.StartOffset >= tailStart {
			chunkLen := p.EndOffset - p.StartOffset + 1
			if chunkLen < baseChunk {
				foundSmallerTail = true
				break
			}
		}
	}
	if !foundSmallerTail {
		t.Error("expected finer tail chunks past 80%")
	}
}

func TestPlanDownloadParts_VeryLargeFile(t *testing.T) {
	e := newPlannerEngine(16, 0)
	// 5GB file → 8MB chunks
	size := int64(5 * 1024 * 1024 * 1024)
	parts := e.planDownloadParts(size, true)

	last := parts[len(parts)-1]
	if last.EndOffset != size-1 {
		t.Errorf("last part end=%d, want=%d", last.EndOffset, size-1)
	}

	// Contiguity
	for i := 1; i < len(parts); i++ {
		if parts[i].StartOffset != parts[i-1].EndOffset+1 {
			t.Fatalf("gap at part %d", i)
		}
	}
}

func TestPlanDownloadParts_ExactChunkBoundary(t *testing.T) {
	e := newPlannerEngine(16, 0)
	// File size exactly 1MB (1 chunk for <=128MB tier)
	size := int64(1 * 1024 * 1024)
	parts := e.planDownloadParts(size, true)

	if parts[0].StartOffset != 0 || parts[len(parts)-1].EndOffset != size-1 {
		t.Error("range mismatch")
	}
}

func TestPlanDownloadParts_UniquePartIDs(t *testing.T) {
	e := newPlannerEngine(16, 0)
	parts := e.planDownloadParts(10*1024*1024, true)
	ids := make(map[int]bool)
	for _, p := range parts {
		if ids[p.ID] {
			t.Fatalf("duplicate part ID: %d", p.ID)
		}
		ids[p.ID] = true
	}
}

func TestPlanDownloadParts_AllAttemptsZero(t *testing.T) {
	e := newPlannerEngine(16, 0)
	parts := e.planDownloadParts(10*1024*1024, true)
	for _, p := range parts {
		if p.Attempts != 0 {
			t.Errorf("part %d should start with 0 attempts", p.ID)
		}
	}
}

// --- selectChunkSize ---

func TestSelectChunkSize_Tiers(t *testing.T) {
	e := newPlannerEngine(16, 0)

	tests := []struct {
		name      string
		totalSize int64
		want      int64
	}{
		{"tiny 1MB", 1 * 1024 * 1024, 1 * 1024 * 1024},
		{"128MB boundary", 128 * 1024 * 1024, 1 * 1024 * 1024},
		{"129MB", 129 * 1024 * 1024, 2 * 1024 * 1024},
		{"1GB", 1024 * 1024 * 1024, 2 * 1024 * 1024},
		{"1GB + 1", 1024*1024*1024 + 1, 4 * 1024 * 1024},
		{"4GB", 4 * 1024 * 1024 * 1024, 4 * 1024 * 1024},
		{"4GB + 1", 4*1024*1024*1024 + 1, 8 * 1024 * 1024},
		{"10GB", 10 * 1024 * 1024 * 1024, 8 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.selectChunkSize(tt.totalSize)
			if got != tt.want {
				t.Errorf("selectChunkSize(%d) = %d, want %d", tt.totalSize, got, tt.want)
			}
		})
	}
}

func TestSelectChunkSize_WithBaseChunkOverride(t *testing.T) {
	// Override with 4MB
	e := newPlannerEngine(16, 4*1024*1024)
	got := e.selectChunkSize(1 * 1024 * 1024) // would normally be 1MB
	if got != 4*1024*1024 {
		t.Errorf("expected override chunk 4MB, got %d", got)
	}
}

func TestSelectChunkSize_OverrideClamped(t *testing.T) {
	// Override below min
	e := newPlannerEngine(16, 100)
	got := e.selectChunkSize(1024 * 1024)
	if got != minAdaptiveChunk {
		t.Errorf("expected clamped to min %d, got %d", minAdaptiveChunk, got)
	}

	// Override above max
	e2 := newPlannerEngine(16, 100*1024*1024)
	got2 := e2.selectChunkSize(1024 * 1024)
	if got2 != maxAdaptiveChunk {
		t.Errorf("expected clamped to max %d, got %d", maxAdaptiveChunk, got2)
	}
}

// --- clampChunk ---

func TestClampChunk(t *testing.T) {
	tests := []struct {
		name  string
		input int64
		want  int64
	}{
		{"below min", 100, minAdaptiveChunk},
		{"at min", minAdaptiveChunk, minAdaptiveChunk},
		{"in range", 2 * 1024 * 1024, 2 * 1024 * 1024},
		{"at max", maxAdaptiveChunk, maxAdaptiveChunk},
		{"above max", 100 * 1024 * 1024, maxAdaptiveChunk},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampChunk(tt.input)
			if got != tt.want {
				t.Errorf("clampChunk(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// --- selectWorkerCount ---

func TestSelectWorkerCount_NoRangeSupport(t *testing.T) {
	e := newPlannerEngine(16, 0)
	got := e.selectWorkerCount("example.com", 100, false)
	if got != 1 {
		t.Errorf("no ranges → expect 1 worker, got %d", got)
	}
}

func TestSelectWorkerCount_ZeroParts(t *testing.T) {
	e := newPlannerEngine(16, 0)
	got := e.selectWorkerCount("example.com", 0, true)
	if got != 1 {
		t.Errorf("0 parts → expect 1 worker, got %d", got)
	}
}

func TestSelectWorkerCount_CappedByParts(t *testing.T) {
	e := newPlannerEngine(64, 0)
	got := e.selectWorkerCount("example.com", 2, true)
	if got > 2 {
		t.Errorf("workers should be capped at numParts (2), got %d", got)
	}
}

func TestSelectWorkerCount_CappedByMaxWorkers(t *testing.T) {
	e := newPlannerEngine(4, 0)
	got := e.selectWorkerCount("example.com", 1000, true)
	if got > 4 {
		t.Errorf("workers should be capped at maxWorkersPerTask (4), got %d", got)
	}
}

func TestSelectWorkerCount_MinimumFour(t *testing.T) {
	e := newPlannerEngine(24, 0)
	got := e.selectWorkerCount("example.com", 1000, true)
	if got < 4 {
		t.Errorf("expected minimum 4 workers, got %d", got)
	}
}

func TestSelectWorkerCount_MaxWorkerOne(t *testing.T) {
	e := newPlannerEngine(1, 0)
	got := e.selectWorkerCount("example.com", 100, true)
	if got != 1 {
		t.Errorf("maxWorkers=1 should yield 1, got %d", got)
	}
}

// --- markHostSingleStream / isHostSingleStream ---

func TestHostSingleStream(t *testing.T) {
	e := newPlannerEngine(16, 0)

	if e.isHostSingleStream("cdn.example.com") {
		t.Error("new host should not be single-stream")
	}

	e.markHostSingleStream("cdn.example.com")
	if !e.isHostSingleStream("cdn.example.com") {
		t.Error("marked host should be single-stream")
	}

	// Other hosts unaffected
	if e.isHostSingleStream("other.com") {
		t.Error("unmarked host should not be single-stream")
	}
}

func TestHostSingleStream_EmptyHost(t *testing.T) {
	e := newPlannerEngine(16, 0)

	e.markHostSingleStream("") // should be no-op
	if e.isHostSingleStream("") {
		t.Error("empty host should return false")
	}
}
