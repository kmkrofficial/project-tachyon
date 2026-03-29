package engine

import (
	"os"
	"sync"
)

// writeRequest is a unit of work for the batched writer goroutine.
type writeRequest struct {
	data   []byte
	offset int64
	done   chan error
}

// batchWriter owns the file handle and serializes all WriteAt calls through
// a channel, eliminating lock contention when many download workers write
// concurrently. Callers receive errors via the per-request done channel.
type batchWriter struct {
	file *os.File
	ch   chan writeRequest
	wg   sync.WaitGroup
	pool *sync.Pool // Buffer pool for recycling data slices
}

// newBatchWriter starts a background goroutine that processes write requests.
func newBatchWriter(file *os.File, pool *sync.Pool) *batchWriter {
	bw := &batchWriter{
		file: file,
		ch:   make(chan writeRequest, 256),
		pool: pool,
	}
	bw.wg.Add(1)
	go bw.loop()
	return bw
}

// Write enqueues a write and blocks until the I/O completes.
// data is copied internally so the caller may reuse its buffer immediately.
func (bw *batchWriter) Write(data []byte, offset int64) error {
	// Copy data so caller can reuse its buffer immediately
	buf := make([]byte, len(data))
	copy(buf, data)

	done := make(chan error, 1)
	bw.ch <- writeRequest{data: buf, offset: offset, done: done}
	return <-done
}

// Close drains the queue and waits for the writer goroutine to exit.
func (bw *batchWriter) Close() {
	close(bw.ch)
	bw.wg.Wait()
}

func (bw *batchWriter) loop() {
	defer bw.wg.Done()
	for req := range bw.ch {
		_, err := bw.file.WriteAt(req.data, req.offset)
		req.done <- err
	}
}
