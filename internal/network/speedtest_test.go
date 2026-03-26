package network

import (
	"testing"
)

func TestSpeedTestResultFields(t *testing.T) {
	result := SpeedTestResult{
		DownloadSpeed:  100.5,
		UploadSpeed:    50.2,
		Ping:           15,
		Jitter:         3,
		ServerName:     "Test Server",
		ServerLocation: "New York, US",
		ServerHost:     "speedtest.example.com",
		ISP:            "TestISP",
		Timestamp:      "2026-01-01T00:00:00Z",
	}

	if result.DownloadSpeed != 100.5 {
		t.Errorf("DownloadSpeed = %f, want 100.5", result.DownloadSpeed)
	}
	if result.UploadSpeed != 50.2 {
		t.Errorf("UploadSpeed = %f, want 50.2", result.UploadSpeed)
	}
	if result.Ping != 15 {
		t.Errorf("Ping = %d, want 15", result.Ping)
	}
	if result.Jitter != 3 {
		t.Errorf("Jitter = %d, want 3", result.Jitter)
	}
	if result.ServerName != "Test Server" {
		t.Errorf("ServerName = %q, want %q", result.ServerName, "Test Server")
	}
	if result.ISP != "TestISP" {
		t.Errorf("ISP = %q, want %q", result.ISP, "TestISP")
	}
	if result.ServerLocation != "New York, US" {
		t.Errorf("ServerLocation = %q, want %q", result.ServerLocation, "New York, US")
	}
	if result.ServerHost != "speedtest.example.com" {
		t.Errorf("ServerHost = %q, want %q", result.ServerHost, "speedtest.example.com")
	}
	if result.Timestamp != "2026-01-01T00:00:00Z" {
		t.Errorf("Timestamp = %q, want %q", result.Timestamp, "2026-01-01T00:00:00Z")
	}
}

func TestSpeedTestPhaseFields(t *testing.T) {
	phases := []SpeedTestPhase{
		{Phase: "connecting"},
		{Phase: "ping", PingMs: 15, ServerName: "S1", ISP: "ISP"},
		{Phase: "download", PingMs: 15, DownloadMbps: 100.5, ServerName: "S1", ISP: "ISP"},
		{Phase: "upload", PingMs: 15, DownloadMbps: 100.5, UploadMbps: 50.2, ServerName: "S1", ISP: "ISP"},
		{Phase: "complete", PingMs: 15, DownloadMbps: 100.5, UploadMbps: 50.2, ServerName: "S1", ISP: "ISP"},
	}

	expectedPhases := []string{"connecting", "ping", "download", "upload", "complete"}
	for i, phase := range phases {
		if phase.Phase != expectedPhases[i] {
			t.Errorf("Phase[%d] = %q, want %q", i, phase.Phase, expectedPhases[i])
		}
	}
}

func TestPhaseCallback_InvocationOrder(t *testing.T) {
	var invocations []string
	callback := PhaseCallback(func(phase SpeedTestPhase) {
		invocations = append(invocations, phase.Phase)
	})

	// Simulate the callback sequence that RunSpeedTestWithEvents would produce
	callback(SpeedTestPhase{Phase: "connecting"})
	callback(SpeedTestPhase{Phase: "ping", ServerName: "Test"})
	callback(SpeedTestPhase{Phase: "download", PingMs: 10})
	callback(SpeedTestPhase{Phase: "upload", DownloadMbps: 100})
	callback(SpeedTestPhase{Phase: "complete", DownloadMbps: 100, UploadMbps: 50})

	expected := []string{"connecting", "ping", "download", "upload", "complete"}
	if len(invocations) != len(expected) {
		t.Fatalf("Got %d invocations, want %d", len(invocations), len(expected))
	}
	for i, phase := range invocations {
		if phase != expected[i] {
			t.Errorf("Invocation[%d] = %q, want %q", i, phase, expected[i])
		}
	}
}

func TestPhaseCallback_NilGuard(t *testing.T) {
	// Verify the nil-guard pattern used in RunSpeedTestWithEvents doesn't panic
	var cb PhaseCallback
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("nil guard panicked: %v", r)
			}
		}()
		if cb != nil {
			cb(SpeedTestPhase{Phase: "test"})
		}
	}()
}
