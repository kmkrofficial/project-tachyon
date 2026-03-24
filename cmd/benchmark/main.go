package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const defaultUbuntuURL = "https://releases.ubuntu.com/24.04/ubuntu-24.04.2-live-server-amd64.iso"

type benchConfig struct {
	Workers int
	ChunkMB int
}

type benchResult struct {
	Config       benchConfig
	Duration     time.Duration
	Bytes        int64
	Errors       int
	PartLatMsP95 float64
	ThroughputMB float64
	Score        float64
}

type downloadPart struct {
	Start int64
	End   int64
}

func main() {
	url := flag.String("url", defaultUbuntuURL, "Benchmark URL (Ubuntu ISO by default)")
	testBytesMB := flag.Int("test-bytes-mb", 512, "How many MB to benchmark via ranged download")
	timeoutSec := flag.Int("timeout-sec", 180, "Timeout per benchmark case")
	requestRetries := flag.Int("retries", 2, "Retries per ranged request")
	flag.Parse()

	testBytes := int64(*testBytesMB) * 1024 * 1024
	if testBytes <= 0 {
		fmt.Println("test-bytes-mb must be > 0")
		return
	}

	fmt.Printf("Aggressive downloader benchmark\n")
	fmt.Printf("Target URL: %s\n", *url)
	fmt.Printf("Sample size: %d MB\n\n", *testBytesMB)

	baseline := []benchConfig{
		{Workers: 4, ChunkMB: 1},
		{Workers: 8, ChunkMB: 1},
		{Workers: 8, ChunkMB: 2},
		{Workers: 12, ChunkMB: 2},
		{Workers: 16, ChunkMB: 4},
		{Workers: 20, ChunkMB: 4},
		{Workers: 24, ChunkMB: 8},
	}

	phase1 := runBatch(*url, baseline, testBytes, time.Duration(*timeoutSec)*time.Second, *requestRetries)
	if len(phase1) == 0 {
		fmt.Println("No successful benchmark runs.")
		return
	}

	slices.SortFunc(phase1, func(a, b benchResult) int {
		if a.Score == b.Score {
			return 0
		}
		if a.Score > b.Score {
			return -1
		}
		return 1
	})

	printResults("Phase 1", phase1)

	phase2Configs := refineConfigs(phase1)
	phase2 := runBatch(*url, phase2Configs, testBytes, time.Duration(*timeoutSec)*time.Second, *requestRetries)
	all := append(phase1, phase2...)
	slices.SortFunc(all, func(a, b benchResult) int {
		if a.Score == b.Score {
			return 0
		}
		if a.Score > b.Score {
			return -1
		}
		return 1
	})

	printResults("Final ranking", all)
	best := all[0]
	fmt.Printf("\nRecommended tuning: workers=%d chunk=%dMB\n", best.Config.Workers, best.Config.ChunkMB)
	fmt.Printf("Engine call: SetDownloadTuning(%d, %d)\n", best.Config.Workers, best.Config.ChunkMB*1024*1024)
}

func runBatch(url string, cfgs []benchConfig, testBytes int64, timeout time.Duration, retries int) []benchResult {
	results := make([]benchResult, 0, len(cfgs))
	for _, cfg := range cfgs {
		res, err := runSingle(url, cfg, testBytes, timeout, retries)
		if err != nil {
			fmt.Printf("- workers=%d chunk=%dMB failed: %v\n", cfg.Workers, cfg.ChunkMB, err)
			continue
		}
		fmt.Printf("- workers=%d chunk=%dMB throughput=%.2f MB/s errors=%d p95=%.1fms score=%.2f\n",
			cfg.Workers, cfg.ChunkMB, res.ThroughputMB, res.Errors, res.PartLatMsP95, res.Score)
		results = append(results, res)
	}
	return results
}

func runSingle(url string, cfg benchConfig, testBytes int64, timeout time.Duration, retries int) (benchResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   20 * time.Second,
			KeepAlive: 20 * time.Second,
		}).DialContext,
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   64,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true,
	}
	client := &http.Client{Transport: transport}

	chunkSize := int64(cfg.ChunkMB) * 1024 * 1024
	if chunkSize <= 0 {
		return benchResult{}, fmt.Errorf("invalid chunk size")
	}

	parts := makeParts(testBytes, chunkSize)
	partCh := make(chan downloadPart, len(parts))
	for _, p := range parts {
		partCh <- p
	}
	close(partCh)

	var totalBytes int64
	var totalErrors int32
	latencies := make([]float64, 0, len(parts))
	latMu := sync.Mutex{}
	wg := sync.WaitGroup{}

	started := time.Now()
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 128*1024)
			for p := range partCh {
				lat := runPart(ctx, client, url, p, retries, buf, &totalBytes)
				if lat < 0 {
					atomic.AddInt32(&totalErrors, 1)
					continue
				}
				latMu.Lock()
				latencies = append(latencies, lat)
				latMu.Unlock()
			}
		}()
	}
	wg.Wait()
	dur := time.Since(started)
	if dur <= 0 {
		dur = time.Second
	}

	bytesDone := atomic.LoadInt64(&totalBytes)
	errorsDone := int(atomic.LoadInt32(&totalErrors))
	if bytesDone == 0 {
		return benchResult{}, fmt.Errorf("no bytes downloaded; check enterprise network policy, URL access, or outbound TLS filtering")
	}
	throughput := float64(bytesDone) / dur.Seconds() / (1024 * 1024)
	p95 := percentile(latencies, 95)
	score := scoreResult(throughput, errorsDone, p95)

	return benchResult{
		Config:       cfg,
		Duration:     dur,
		Bytes:        bytesDone,
		Errors:       errorsDone,
		PartLatMsP95: p95,
		ThroughputMB: throughput,
		Score:        score,
	}, nil
}

func runPart(ctx context.Context, client *http.Client, url string, p downloadPart, retries int, buf []byte, totalBytes *int64) float64 {
	for attempt := 0; attempt <= retries; attempt++ {
		started := time.Now()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return -1
		}
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", p.Start, p.End))
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("User-Agent", "tachyon-benchmark/1.0")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}

		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				atomic.AddInt64(totalBytes, int64(n))
			}
			if readErr != nil {
				if readErr == io.EOF {
					resp.Body.Close()
					return float64(time.Since(started).Milliseconds())
				}
				break
			}
		}
		resp.Body.Close()
	}
	return -1
}

func makeParts(totalBytes, chunkSize int64) []downloadPart {
	parts := make([]downloadPart, 0, int(totalBytes/chunkSize)+1)
	offset := int64(0)
	for offset < totalBytes {
		end := offset + chunkSize - 1
		if end >= totalBytes {
			end = totalBytes - 1
		}
		parts = append(parts, downloadPart{Start: offset, End: end})
		offset = end + 1
	}
	return parts
}

func refineConfigs(results []benchResult) []benchConfig {
	seen := map[string]bool{}
	out := make([]benchConfig, 0, 12)
	limit := min(3, len(results))
	for i := 0; i < limit; i++ {
		base := results[i].Config
		neighbors := []benchConfig{
			{Workers: base.Workers - 4, ChunkMB: base.ChunkMB},
			{Workers: base.Workers - 2, ChunkMB: base.ChunkMB},
			{Workers: base.Workers + 2, ChunkMB: base.ChunkMB},
			{Workers: base.Workers + 4, ChunkMB: base.ChunkMB},
			{Workers: base.Workers, ChunkMB: max(1, base.ChunkMB/2)},
			{Workers: base.Workers, ChunkMB: min(16, base.ChunkMB*2)},
		}
		for _, cfg := range neighbors {
			if cfg.Workers < 1 {
				continue
			}
			if cfg.Workers > 64 {
				continue
			}
			key := fmt.Sprintf("%d:%d", cfg.Workers, cfg.ChunkMB)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, cfg)
		}
	}
	return out
}

func percentile(values []float64, p int) float64 {
	if len(values) == 0 {
		return 0
	}
	copyVals := append([]float64(nil), values...)
	slices.Sort(copyVals)
	idx := int(math.Ceil((float64(p)/100.0)*float64(len(copyVals)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(copyVals) {
		idx = len(copyVals) - 1
	}
	return copyVals[idx]
}

func scoreResult(throughputMB float64, errors int, p95ms float64) float64 {
	errPenalty := 1.0 / (1.0 + float64(errors))
	latPenalty := 1000.0 / (1000.0 + p95ms)
	return throughputMB * errPenalty * latPenalty
}

func printResults(title string, results []benchResult) {
	fmt.Printf("\n%s\n", title)
	fmt.Println(strings.Repeat("-", len(title)))
	limit := min(10, len(results))
	for i := 0; i < limit; i++ {
		r := results[i]
		fmt.Printf("%2d) workers=%-2d chunk=%-2dMB throughput=%6.2f MB/s p95=%6.1fms errors=%-2d score=%7.2f\n",
			i+1, r.Config.Workers, r.Config.ChunkMB, r.ThroughputMB, r.PartLatMsP95, r.Errors, r.Score)
	}
}
