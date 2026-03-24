package network

import (
	"context"
	"net"
	"sync"
	"time"
)

// DNSCache provides a thread-safe local DNS cache to avoid redundant lookups
// during multi-part downloads to the same host.
type DNSCache struct {
	mu      sync.RWMutex
	entries map[string]*dnsEntry
	ttl     time.Duration
}

type dnsEntry struct {
	addrs   []string
	expires time.Time
}

// NewDNSCache creates a cache with the given TTL.
func NewDNSCache(ttl time.Duration) *DNSCache {
	return &DNSCache{
		entries: make(map[string]*dnsEntry),
		ttl:     ttl,
	}
}

// DialContext returns a net.Dialer.DialContext replacement that caches DNS results.
func (c *DNSCache) DialContext(timeout, keepAlive time.Duration) func(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: keepAlive,
	}

	return func(ctx context.Context, netw, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return dialer.DialContext(ctx, netw, addr)
		}

		// Check cache
		if ip := c.get(host); ip != "" {
			return dialer.DialContext(ctx, netw, net.JoinHostPort(ip, port))
		}

		// Resolve and cache
		addrs, err := net.DefaultResolver.LookupHost(ctx, host)
		if err != nil || len(addrs) == 0 {
			// Fall through to normal dial on lookup failure
			return dialer.DialContext(ctx, netw, addr)
		}

		c.put(host, addrs)
		return dialer.DialContext(ctx, netw, net.JoinHostPort(addrs[0], port))
	}
}

func (c *DNSCache) get(host string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	e, ok := c.entries[host]
	if !ok || time.Now().After(e.expires) {
		return ""
	}
	return e.addrs[0]
}

func (c *DNSCache) put(host string, addrs []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[host] = &dnsEntry{
		addrs:   addrs,
		expires: time.Now().Add(c.ttl),
	}
}
