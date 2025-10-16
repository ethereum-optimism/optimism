package resources

import (
	"io"
	"net/http"
	"strings"
	"sync"

	gethlog "github.com/ethereum/go-ethereum/log"
)

// RouterConfig defines runtime options for the RPC router server.
type RouterConfig struct {
	EnableWebsockets bool
	MaxBodyBytes     int64
}

// Router multiplexes JSON-RPC requests by the first path segment which represents the chainID.
type Router struct {
	log     gethlog.Logger
	cfg     RouterConfig
	mu      sync.RWMutex
	paths   map[string]http.Handler // chainID -> handler
	closers []io.Closer
}

// NewRouter constructs an empty Router. Handlers can be added later via SetHandler.
func NewRouter(log gethlog.Logger, cfg RouterConfig) *Router {
	return &Router{log: log, cfg: cfg, paths: make(map[string]http.Handler)}
}

// Close releases any resources created by the factory.
func (r *Router) Close() error {
	var firstErr error
	for _, c := range r.closers {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// SetHandler replaces or adds the handler for a given chainID at runtime.
func (r *Router) SetHandler(chainID string, h http.Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.paths == nil {
		r.paths = make(map[string]http.Handler)
	}
	r.paths[chainID] = h
}

// ServeHTTP routes requests to the chain-specific handler, after stripping the chain prefix.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	chainID, remainder := splitFirstSegment(req.URL.Path)
	if chainID == "" {
		http.NotFound(w, req)
		return
	}

	r.mu.RLock()
	h, ok := r.paths[chainID]
	r.mu.RUnlock()
	if !ok {
		http.NotFound(w, req)
		return
	}

	// Rewrite path so the downstream handler sees root or the remaining path after the chainID
	// We only touch URL.Path and RequestURI for correctness; leave the body and headers intact.
	origPath := req.URL.Path
	origReqURI := req.RequestURI
	req.URL.Path = remainder
	if req.URL.RawQuery != "" {
		req.RequestURI = remainder + "?" + req.URL.RawQuery
	} else {
		req.RequestURI = remainder
	}
	defer func() {
		req.URL.Path = origPath
		req.RequestURI = origReqURI
	}()

	h.ServeHTTP(w, req)
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
