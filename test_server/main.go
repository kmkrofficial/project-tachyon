package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	port := flag.Int("port", 9876, "port to listen on")
	logFile := flag.String("logfile", "", "path to write JSON logs (empty = stderr only)")
	bandwidth := flag.String("bandwidth", "", "global bandwidth limit (e.g. 50mb, 100mb, 500kb). empty = unlimited")
	flag.Parse()

	// Parse bandwidth limit
	var bandwidthBPS int64
	if *bandwidth != "" {
		bw, err := parseSize(*bandwidth)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid bandwidth value %q: %v\n", *bandwidth, err)
			os.Exit(1)
		}
		bandwidthBPS = bw
	}

	// Configure structured logger
	var handler slog.Handler
	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open log file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		handler = slog.NewJSONHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	logger := slog.New(handler)

	srv := NewTestServer(logger, bandwidthBPS)
	mux := srv.Handler()

	addr := fmt.Sprintf(":%d", *port)
	server := &http.Server{Addr: addr, Handler: mux}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("shutdown signal received")
		server.Close()
	}()

	logger.Info("test server starting", "addr", addr, "logfile", *logFile, "bandwidth", *bandwidth)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
	logger.Info("test server stopped")
}
