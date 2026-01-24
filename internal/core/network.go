package core

import (
	"context"
	"fmt"
	"time"

	"github.com/showwin/speedtest-go/speedtest"
)

type SpeedTestResult struct {
	DownloadSpeed float64   `json:"download_mbps"`
	UploadSpeed   float64   `json:"upload_mbps"`
	Ping          int64     `json:"ping_ms"`
	ServerHost    string    `json:"server_host"`
	Timestamp     time.Time `json:"timestamp"`
}

func RunSpeedTest() (*SpeedTestResult, error) {
	// Create a 15-second timeout context to prevent freezing
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Fetch user info for location-based server selection
	user, err := speedtest.FetchUserInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}

	// Fetch server list and find closest servers
	serverList, err := speedtest.FetchServers()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch servers: %w", err)
	}

	// Get servers sorted by distance (closest first)
	targets, err := serverList.FindServer([]int{})
	if err != nil || len(targets) == 0 {
		return nil, fmt.Errorf("no servers found: %w", err)
	}

	// Use the closest server
	server := targets[0]

	// Run ping test
	err = server.PingTestContext(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("ping test failed: %w", err)
	}

	// Run download test
	err = server.DownloadTestContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("download test failed: %w", err)
	}

	// Run upload test
	err = server.UploadTestContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("upload test failed: %w", err)
	}

	// Build result
	result := &SpeedTestResult{
		DownloadSpeed: float64(server.DLSpeed) / 1024 / 1024, // Convert bytes/s to Mbps
		UploadSpeed:   float64(server.ULSpeed) / 1024 / 1024, // Convert bytes/s to Mbps
		Ping:          int64(server.Latency.Milliseconds()),
		ServerHost:    fmt.Sprintf("%s (%s)", server.Name, server.Host),
		Timestamp:     time.Now(),
	}

	// Log for debugging (uses user info)
	_ = user // Suppress unused warning, can be used for logging ISP info

	return result, nil
}
