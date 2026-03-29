package engine

import (
	"context"
	"io"
	"net/http"
	"time"
)

// WarmUpHost pre-establishes idle TCP connections to the target host so the
// first download chunks don't pay TLS handshake + TCP slow-start latency.
// It fires `count` parallel HEAD requests that complete quickly (no body)
// and return connections to the pool.
func (e *TachyonEngine) WarmUpHost(host string, count int) {
	if count < 1 || host == "" {
		return
	}
	if count > 4 {
		count = 4
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{}, count)
	for i := 0; i < count; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			req, err := http.NewRequestWithContext(ctx, "HEAD", "https://"+host+"/", nil)
			if err != nil {
				return
			}
			req.Header.Set("User-Agent", e.GetUserAgent())
			resp, err := e.httpClient.Do(req)
			if err != nil {
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}()
	}
	for i := 0; i < count; i++ {
		<-done
	}
}
