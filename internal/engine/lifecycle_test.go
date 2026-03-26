package engine

import (
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

func TestWaitForSignals_CallbackFired(t *testing.T) {
	called := make(chan bool, 1)

	WaitForSignals(func() {
		called <- true
	})

	// Give goroutine time to start
	time.Sleep(10 * time.Millisecond)

	// Send signal to self
	p, _ := os.FindProcess(os.Getpid())
	p.Signal(os.Interrupt)

	select {
	case <-called:
		// success
	case <-time.After(2 * time.Second):
		t.Error("Callback was not fired within timeout")
	}

	// Reset signal handling for other tests
	signal.Reset(os.Interrupt, syscall.SIGTERM)
}

func TestWaitForSignals_NilCallback(t *testing.T) {
	// Should not panic even with nil callback
	WaitForSignals(nil)

	// Give goroutine time to start
	time.Sleep(10 * time.Millisecond)

	// Send signal — should not panic
	p, _ := os.FindProcess(os.Getpid())
	p.Signal(os.Interrupt)

	// Give it time to process
	time.Sleep(50 * time.Millisecond)

	// Reset
	signal.Reset(os.Interrupt, syscall.SIGTERM)
}
