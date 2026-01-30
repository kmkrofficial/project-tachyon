package core

import (
	"os"
	"os/signal"
	"syscall"
)

// WaitForSignals listens for os.Interrupt and syscall.SIGTERM
// and calls the provided onSignal function when triggered.
func WaitForSignals(onSignal func()) {
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		// Block until signal received
		<-sigChan

		// Trigger callback
		if onSignal != nil {
			onSignal()
		}
	}()
}
