package network

import (
	"context"
	"fmt"
	"time"

	"github.com/showwin/speedtest-go/speedtest"
)

// SpeedTestResult contains the results of a network speed test
type SpeedTestResult struct {
	DownloadSpeed  float64 `json:"download_mbps"`
	UploadSpeed    float64 `json:"upload_mbps"`
	Ping           int64   `json:"ping_ms"`
	Jitter         int64   `json:"jitter_ms"`
	ServerName     string  `json:"server_name"`
	ServerLocation string  `json:"server_location"`
	ServerHost     string  `json:"server_host"`
	ISP            string  `json:"isp"`
	Timestamp      string  `json:"timestamp"`
}

// SpeedTestPhase represents the current phase of the speed test
type SpeedTestPhase struct {
	Phase        string  `json:"phase"`         // "connecting", "ping", "download", "upload", "complete"
	PingMs       int64   `json:"ping_ms"`       // Available after ping phase
	DownloadMbps float64 `json:"download_mbps"` // Available during/after download
	UploadMbps   float64 `json:"upload_mbps"`   // Available during/after upload
	ServerName   string  `json:"server_name"`   // Available after connecting
	ISP          string  `json:"isp"`           // Available after connecting
}

// PhaseCallback is called during each phase of the speed test
type PhaseCallback func(phase SpeedTestPhase)

// RunSpeedTest performs a network speed test using nearest available server
func RunSpeedTest() (*SpeedTestResult, error) {
	return RunSpeedTestWithEvents(nil)
}

// RunSpeedTestWithEvents performs a speed test and calls the callback at each phase
func RunSpeedTestWithEvents(onPhase PhaseCallback) (*SpeedTestResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Phase: Connecting
	if onPhase != nil {
		onPhase(SpeedTestPhase{Phase: "connecting"})
	}

	// Fetch user info for location-based server selection
	user, err := speedtest.FetchUserInfo()
	if err != nil {
		return nil, fmt.Errorf("no internet connection")
	}

	// Fetch server list
	serverList, err := speedtest.FetchServers()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch servers")
	}

	// Get servers sorted by distance (closest first)
	targets, err := serverList.FindServer([]int{})
	if err != nil || len(targets) == 0 {
		return nil, fmt.Errorf("no speed test servers available")
	}

	server := targets[0]

	// Emit server info
	if onPhase != nil {
		onPhase(SpeedTestPhase{
			Phase:      "ping",
			ServerName: server.Name,
			ISP:        user.Isp,
		})
	}

	// Phase: Ping Test
	if err := server.PingTestContext(ctx, nil); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("speed test timed out")
		}
		return nil, fmt.Errorf("ping test failed")
	}

	pingMs := int64(server.Latency.Milliseconds())

	// Emit ping result
	if onPhase != nil {
		onPhase(SpeedTestPhase{
			Phase:      "download",
			PingMs:     pingMs,
			ServerName: server.Name,
			ISP:        user.Isp,
		})
	}

	// Phase: Download Test
	if err := server.DownloadTestContext(ctx); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("speed test timed out during download")
		}
		return nil, fmt.Errorf("download test failed")
	}

	downloadMbps := float64(server.DLSpeed) / 1000 / 1000 * 8

	// Emit download result
	if onPhase != nil {
		onPhase(SpeedTestPhase{
			Phase:        "upload",
			PingMs:       pingMs,
			DownloadMbps: downloadMbps,
			ServerName:   server.Name,
			ISP:          user.Isp,
		})
	}

	// Phase: Upload Test
	if err := server.UploadTestContext(ctx); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("speed test timed out during upload")
		}
		return nil, fmt.Errorf("upload test failed")
	}

	uploadMbps := float64(server.ULSpeed) / 1000 / 1000 * 8

	result := &SpeedTestResult{
		DownloadSpeed:  downloadMbps,
		UploadSpeed:    uploadMbps,
		Ping:           pingMs,
		Jitter:         int64(server.Jitter.Milliseconds()),
		ServerName:     server.Name,
		ServerLocation: fmt.Sprintf("%s, %s", server.Name, server.Country),
		ServerHost:     server.Host,
		ISP:            user.Isp,
		Timestamp:      time.Now().Format(time.RFC3339),
	}

	// Phase: Complete
	if onPhase != nil {
		onPhase(SpeedTestPhase{
			Phase:        "complete",
			PingMs:       pingMs,
			DownloadMbps: downloadMbps,
			UploadMbps:   uploadMbps,
			ServerName:   server.Name,
			ISP:          user.Isp,
		})
	}

	return result, nil
}
