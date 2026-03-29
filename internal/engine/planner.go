package engine

const (
	minAdaptiveChunk = int64(512 * 1024)
	maxAdaptiveChunk = int64(16 * 1024 * 1024)
	StreamEndOffset  = int64(^uint64(0) >> 1)
)

// planDownloadParts builds a deterministic segment plan with finer tail chunks
// to reduce straggler effects near completion.
func (e *TachyonEngine) planDownloadParts(totalSize int64, acceptRanges bool) []DownloadPart {
	if totalSize <= 0 || !acceptRanges {
		return []DownloadPart{{ID: 0, StartOffset: 0, EndOffset: StreamEndOffset, Attempts: 0}}
	}

	baseChunk := e.selectChunkSize(totalSize)
	tailChunk := baseChunk / 4
	if tailChunk < minAdaptiveChunk {
		tailChunk = minAdaptiveChunk
	}

	tailStart := int64(float64(totalSize) * 0.8)
	parts := make([]DownloadPart, 0, int(totalSize/baseChunk)+16)
	offset := int64(0)
	id := 0
	for offset < totalSize {
		chunk := baseChunk
		if offset >= tailStart {
			chunk = tailChunk
		}

		end := offset + chunk - 1
		if end >= totalSize {
			end = totalSize - 1
		}

		parts = append(parts, DownloadPart{
			ID:          id,
			StartOffset: offset,
			EndOffset:   end,
			Attempts:    0,
		})
		offset = end + 1
		id++
	}

	return parts
}

func (e *TachyonEngine) selectChunkSize(totalSize int64) int64 {
	if e.baseChunkSize > 0 {
		return clampChunk(e.baseChunkSize)
	}

	switch {
	case totalSize <= 64*1024*1024:
		return 2 * 1024 * 1024
	case totalSize <= 512*1024*1024:
		return 4 * 1024 * 1024
	case totalSize <= 2*1024*1024*1024:
		return 8 * 1024 * 1024
	default:
		return 16 * 1024 * 1024
	}
}

func (e *TachyonEngine) selectWorkerCount(host string, numParts int, acceptRanges bool) int {
	return e.selectWorkerCountH2(host, numParts, acceptRanges, false)
}

func (e *TachyonEngine) selectWorkerCountH2(host string, numParts int, acceptRanges bool, isH2 bool) int {
	if !acceptRanges {
		return 1
	}
	if numParts < 1 {
		return 1
	}

	workers := e.congestion.GetIdealConcurrency(host)
	if workers < 4 {
		workers = 4
	}

	maxWorkers := e.maxWorkersPerTask
	if maxWorkers < 1 {
		maxWorkers = 1
	}

	// HTTP/2 multiplexes all streams over one TCP connection — more workers
	// cause head-of-line blocking with diminishing returns. Cap at 8.
	if isH2 && maxWorkers > 8 {
		maxWorkers = 8
	}

	if workers > maxWorkers {
		workers = maxWorkers
	}
	if workers > numParts {
		workers = numParts
	}
	if workers < 1 {
		workers = 1
	}
	return workers
}

func clampChunk(size int64) int64 {
	if size < minAdaptiveChunk {
		return minAdaptiveChunk
	}
	if size > maxAdaptiveChunk {
		return maxAdaptiveChunk
	}
	return size
}

func (e *TachyonEngine) markHostSingleStream(host string) {
	if host == "" {
		return
	}
	e.hostSingleStream.Store(host, true)
}

func (e *TachyonEngine) isHostSingleStream(host string) bool {
	if host == "" {
		return false
	}
	v, ok := e.hostSingleStream.Load(host)
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}
