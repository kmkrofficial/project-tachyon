package engine

import (
	"os"
	"sync"
	"testing"
)

func TestBatchWriter_BasicWrite(t *testing.T) {
	f, err := os.CreateTemp("", "bw_test_*.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	pool := &sync.Pool{New: func() interface{} { b := make([]byte, 256); return &b }}
	bw := newBatchWriter(f, pool)

	data := []byte("hello, tachyon!")
	if err := bw.Write(data, 0); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	bw.Close()
	f.Close()

	// Verify
	content, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if string(content[:len(data)]) != "hello, tachyon!" {
		t.Errorf("expected 'hello, tachyon!', got %q", string(content[:len(data)]))
	}
}

func TestBatchWriter_ConcurrentWrites(t *testing.T) {
	f, err := os.CreateTemp("", "bw_conc_*.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	// Pre-allocate file
	f.Truncate(1024)

	pool := &sync.Pool{New: func() interface{} { b := make([]byte, 256); return &b }}
	bw := newBatchWriter(f, pool)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(offset int64) {
			defer wg.Done()
			data := []byte{byte(offset)}
			if err := bw.Write(data, offset); err != nil {
				t.Errorf("Write at offset %d failed: %v", offset, err)
			}
		}(int64(i * 100))
	}
	wg.Wait()
	bw.Close()
	f.Close()

	content, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		if content[i*100] != byte(i*100) {
			t.Errorf("byte at offset %d: expected %d, got %d", i*100, byte(i*100), content[i*100])
		}
	}
}

func TestBatchWriter_CallerBufferReuse(t *testing.T) {
	f, err := os.CreateTemp("", "bw_reuse_*.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Truncate(64)

	pool := &sync.Pool{New: func() interface{} { b := make([]byte, 256); return &b }}
	bw := newBatchWriter(f, pool)

	// Write data, then overwrite buffer — batchWriter should have copied
	buf := []byte("original")
	if err := bw.Write(buf, 0); err != nil {
		t.Fatal(err)
	}
	copy(buf, "modified")
	bw.Close()
	f.Close()

	content, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if string(content[:8]) != "original" {
		t.Errorf("expected 'original', got %q (buffer reuse corruption)", string(content[:8]))
	}
}
