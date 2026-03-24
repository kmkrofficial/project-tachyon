package engine

import "sync"

// WorkerPool is a fixed-size goroutine pool that processes generic work items.
// It amortises goroutine creation/teardown across many short-lived download tasks.
type WorkerPool struct {
	jobCh chan func()
	wg    sync.WaitGroup
}

// NewWorkerPool spins up `size` persistent goroutines that pull work from a shared channel.
func NewWorkerPool(size int) *WorkerPool {
	if size < 1 {
		size = 1
	}
	wp := &WorkerPool{
		jobCh: make(chan func(), size*4),
	}
	wp.wg.Add(size)
	for i := 0; i < size; i++ {
		go func() {
			defer wp.wg.Done()
			for fn := range wp.jobCh {
				fn()
			}
		}()
	}
	return wp
}

// Submit enqueues a unit of work. Blocks if the pool's job buffer is full.
func (wp *WorkerPool) Submit(fn func()) {
	wp.jobCh <- fn
}

// Close drains the pool and waits for all goroutines to exit.
func (wp *WorkerPool) Close() {
	close(wp.jobCh)
	wp.wg.Wait()
}
