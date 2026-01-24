package core

import (
	"context"
	"fmt"
	"time"

	"github.com/showwin/speedtest-go/speedtest"
)

// SpeedTestResult contains the results of a network speed test
type SpeedTestResult struct {
	DownloadSpeed  float64   `json:"download_mbps"`
	UploadSpeed    float64   `json:"upload_mbps"`
	Ping           int64     `json:"ping_ms"`
	ServerName     string    `json:"server_name"`
	ServerLocation string    `json:"server_location"`
	ServerHost     string    `json:"server_host"`
	ISP            string    `json:"isp"`
	Timestamp      time.Time `json:"timestamp"`
}

// RunSpeedTest performs a network speed test using nearest available server
func RunSpeedTest() (*SpeedTestResult, error) {
	// Create a 30-second timeout context (speed tests can take time)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Fetch user info for location-based server selection
	user, err := speedtest.FetchUserInfo()
	if err != nil {
		return nil, fmt.Errorf("no internet connection: %w", err)
	}

	// Fetch server list
	serverList, err := speedtest.FetchServers()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch servers: %w", err)
	}

	// Get servers sorted by distance (closest first)
	targets, err := serverList.FindServer([]int{})
	if err != nil || len(targets) == 0 {
		return nil, fmt.Errorf("no speed test servers available")
	}

	// Use the closest server
	server := targets[0]

	// Run ping test first
	if err := server.PingTestContext(ctx, nil); err != nil {
		// Check if context was cancelled
		if ctx.Err() != nil {
			return nil, fmt.Errorf("speed test timed out")
		}
		return nil, fmt.Errorf("ping test failed: %w", err)
	}

	// Run download test
	if err := server.DownloadTestContext(ctx); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("speed test timed out during download")
		}
		return nil, fmt.Errorf("download test failed: %w", err)
	}

	// Run upload test
	if err := server.UploadTestContext(ctx); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("speed test timed out during upload")
		}
		return nil, fmt.Errorf("upload test failed: %w", err)
	}

	// Build result with all server information
	// DLSpeed and ULSpeed are in bytes per second, convert to Mbps
	result := &SpeedTestResult{
		DownloadSpeed:  float64(server.DLSpeed) / 1000 / 1000 * 8, // bytes/s to Mbps
		UploadSpeed:    float64(server.ULSpeed) / 1000 / 1000 * 8, // bytes/s to Mbps
		Ping:           int64(server.Latency.Milliseconds()),
		ServerName:     server.Name,
		ServerLocation: fmt.Sprintf("%s, %s", server.Name, server.Country),
		ServerHost:     server.Host,
		ISP:            user.Isp,
		Timestamp:      time.Now(),
	}

	return result, nil
}
