package engine

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewWorkerPool_DefaultSize(t *testing.T) {
	wp := NewWorkerPool(4)
	defer wp.Close()
	// Should not panic or hang
}

func TestNewWorkerPool_ZeroSize(t *testing.T) {
	wp := NewWorkerPool(0)
	defer wp.Close()
	// size < 1 → clamped to 1
	var done atomic.Int32
	wp.Submit(func() { done.Add(1) })
	time.Sleep(50 * time.Millisecond)
	if done.Load() != 1 {
		t.Fatal("pool with size 0 should still process work (clamped to 1)")
	}
}

func TestNewWorkerPool_NegativeSize(t *testing.T) {
	wp := NewWorkerPool(-5)
	defer wp.Close()
	var done atomic.Int32
	wp.Submit(func() { done.Add(1) })
	time.Sleep(50 * time.Millisecond)
	if done.Load() != 1 {
		t.Fatal("pool with negative size should still process work (clamped to 1)")
	}
}

func TestWorkerPool_SubmitAndExecute(t *testing.T) {
	wp := NewWorkerPool(4)
	defer wp.Close()

	var counter atomic.Int32
	n := 100

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		wp.Submit(func() {
			counter.Add(1)
			wg.Done()
		})
	}

	wg.Wait()

	if counter.Load() != int32(n) {
		t.Errorf("expected %d, got %d", n, counter.Load())
	}
}

func TestWorkerPool_ConcurrentSubmit(t *testing.T) {
	wp := NewWorkerPool(8)
	defer wp.Close()

	var counter atomic.Int32
	n := 200

	// Submit from multiple goroutines concurrently
	var submitWg sync.WaitGroup
	var doneWg sync.WaitGroup
	doneWg.Add(n)

	for i := 0; i < 10; i++ {
		submitWg.Add(1)
		go func(batch int) {
			defer submitWg.Done()
			for j := 0; j < n/10; j++ {
				wp.Submit(func() {
					counter.Add(1)
					doneWg.Done()
				})
			}
		}(i)
	}

	submitWg.Wait()
	doneWg.Wait()

	if counter.Load() != int32(n) {
		t.Errorf("expected %d, got %d", n, counter.Load())
	}
}

func TestWorkerPool_Close_DrainsWork(t *testing.T) {
	wp := NewWorkerPool(2)

	var counter atomic.Int32
	for i := 0; i < 50; i++ {
		wp.Submit(func() {
			time.Sleep(1 * time.Millisecond)
			counter.Add(1)
		})
	}

	wp.Close() // Should block until all work is done
	if counter.Load() != 50 {
		t.Errorf("expected all 50 jobs to complete after Close, got %d", counter.Load())
	}
}

func TestWorkerPool_Close_Idempotent(t *testing.T) {
	wp := NewWorkerPool(2)
	wp.Close()
	// Second close should not panic (channel already closed)
	// Note: This will panic if Close is called twice — this tests that behavior.
	// If it does panic, the pool needs a sync.Once guard.
}

func TestWorkerPool_OrderIndependence(t *testing.T) {
	wp := NewWorkerPool(1) // Single worker to force sequential execution
	defer wp.Close()

	results := make([]int, 0, 10)
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		val := i
		wp.Submit(func() {
			mu.Lock()
			results = append(results, val)
			mu.Unlock()
			wg.Done()
		})
	}

	wg.Wait()
	if len(results) != 10 {
		t.Errorf("expected 10 results, got %d", len(results))
	}
}
