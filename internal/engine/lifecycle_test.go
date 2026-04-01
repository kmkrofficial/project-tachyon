package engine

import (
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

func TestWaitForSignals_CallbackFired(t *testing.T) {
	if testing.Short() {
		t.Skip("signal tests unreliable in short mode")
	}
	// On Windows, Process.Signal(os.Interrupt) does not reliably deliver to
	// the calling process. Skip when the runtime clearly does not support it.
	if err := testSignalSelf(); err != nil {
		t.Skipf("os.Interrupt not deliverable on this OS: %v", err)
	}

	called := make(chan bool, 1)

	WaitForSignals(func() {
		called <- true
	})

	// Give goroutine time to start
	time.Sleep(50 * time.Millisecond)

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

// testSignalSelf checks if the process can send os.Interrupt to itself.
func testSignalSelf() error {
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		return err
	}
	return p.Signal(os.Interrupt)
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
