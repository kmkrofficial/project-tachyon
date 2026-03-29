package engine

import (
	"testing"
	"time"
)

func TestProbeCache_PutAndGet(t *testing.T) {
	pc := newProbeCache()
	result := &ProbeResult{Size: 1024, Filename: "test.zip"}
	pc.Put("http://example.com/test.zip", result)

	cached := pc.Get("http://example.com/test.zip")
	if cached == nil {
		t.Fatal("expected cached result")
	}
	if cached.Size != 1024 {
		t.Errorf("expected size 1024, got %d", cached.Size)
	}
	if cached.Filename != "test.zip" {
		t.Errorf("expected filename test.zip, got %s", cached.Filename)
	}
}

func TestProbeCache_Miss(t *testing.T) {
	pc := newProbeCache()
	if pc.Get("http://nonexistent.com/file") != nil {
		t.Error("expected nil for uncached URL")
	}
}

func TestProbeCache_Delete(t *testing.T) {
	pc := newProbeCache()
	pc.Put("http://example.com/a", &ProbeResult{Size: 100})
	pc.Delete("http://example.com/a")
	if pc.Get("http://example.com/a") != nil {
		t.Error("expected nil after delete")
	}
}

func TestProbeCache_TTLExpiry(t *testing.T) {
	pc := newProbeCache()
	pc.Put("http://example.com/b", &ProbeResult{Size: 200})

	// Manually expire
	pc.mu.Lock()
	pc.items["http://example.com/b"].created = time.Now().Add(-probeCacheTTL - time.Second)
	pc.mu.Unlock()

	if pc.Get("http://example.com/b") != nil {
		t.Error("expected nil for expired entry")
	}
}
