package reverseproxy

import (
	"io"
	"net/http"
	"strings"
	"sync"

	gethlog "github.com/ethereum/go-ethereum/log"
)

// Config defines runtime options for the reverse proxy server.
type Config struct {
	EnableWebsockets bool
	MaxBodyBytes     int64
}

// Proxy multiplexes JSON-RPC requests by the first path segment which represents the chainID.
type Proxy struct {
	log     gethlog.Logger
	cfg     Config
	mu      sync.RWMutex
	paths   map[string]http.Handler // chainID -> handler
	closers []io.Closer
}

// New constructs an empty Proxy. Handlers can be added later via SetHandler.
func New(log gethlog.Logger, cfg Config) *Proxy {
	return &Proxy{log: log, cfg: cfg, paths: make(map[string]http.Handler)}
}

// Close releases any resources created by the factory.
func (p *Proxy) Close() error {
	var firstErr error
	for _, c := range p.closers {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// SetHandler replaces or adds the handler for a given chainID at runtime.
func (p *Proxy) SetHandler(chainID string, h http.Handler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.paths == nil {
		p.paths = make(map[string]http.Handler)
	}
	p.paths[chainID] = h
}

// ServeHTTP routes requests to the chain-specific handler, after stripping the chain prefix.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	chainID, remainder := splitFirstSegment(r.URL.Path)
	if chainID == "" {
		http.NotFound(w, r)
		return
	}

	p.mu.RLock()
	h, ok := p.paths[chainID]
	p.mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	// Rewrite path so the downstream handler sees root or the remaining path after the chainID
	// We only touch URL.Path and RequestURI for correctness; leave the body and headers intact.
	origPath := r.URL.Path
	origReqURI := r.RequestURI
	r.URL.Path = remainder
	if r.URL.RawQuery != "" {
		r.RequestURI = remainder + "?" + r.URL.RawQuery
	} else {
		r.RequestURI = remainder
	}
	defer func() {
		r.URL.Path = origPath
		r.RequestURI = origReqURI
	}()

	h.ServeHTTP(w, r)
}

// splitFirstSegment returns the first non-empty path segment and the remainder path starting with '/'.
func splitFirstSegment(p string) (seg string, remainder string) {
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return "", "/"
	}
	idx := strings.IndexByte(p, '/')
	if idx == -1 {
		return p, "/"
	}
	return p[:idx], "/" + p[idx+1:]
}
