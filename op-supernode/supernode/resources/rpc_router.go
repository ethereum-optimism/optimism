package resources

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	gethlog "github.com/ethereum/go-ethereum/log"
)

const (
	defaultGatePoll    = 25 * time.Millisecond
	defaultGateTimeout = 60 * time.Second
)

// RouterConfig defines runtime options for the RPC router server.
type RouterConfig struct {
	EnableWebsockets bool
	MaxBodyBytes     int64
	GateTimeout      time.Duration
}

type chainRoute struct {
	handler http.Handler
	ready   func() bool
}

// Router multiplexes JSON-RPC requests by the first path segment which represents the chainID.
type Router struct {
	log          gethlog.Logger
	cfg          RouterConfig
	mu           sync.RWMutex
	routes       map[string]*chainRoute // chainID -> route
	root         http.Handler           // root handler mounted at '/'
	closers      []io.Closer
	gateTimeout  time.Duration
	pollInterval time.Duration
}

// NewRouter constructs an empty Router. Handlers can be added later via SetHandler.
func NewRouter(log gethlog.Logger, cfg RouterConfig) *Router {
	gateTimeout := cfg.GateTimeout
	if gateTimeout == 0 {
		gateTimeout = defaultGateTimeout
	}
	return &Router{
		log:          log,
		cfg:          cfg,
		routes:       make(map[string]*chainRoute),
		gateTimeout:  gateTimeout,
		pollInterval: defaultGatePoll,
	}
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

// getOrCreateRouteLocked returns the chain route, creating it if necessary.
// Callers must hold r.mu for writes.
func (r *Router) getOrCreateRouteLocked(chainID string) *chainRoute {
	route := r.routes[chainID]
	if route == nil {
		route = &chainRoute{}
		r.routes[chainID] = route
	}
	return route
}

// SetHandler replaces or adds the handler for a given chainID at runtime.
func (r *Router) SetHandler(chainID string, h http.Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.getOrCreateRouteLocked(chainID).handler = h
}

// SetReadinessCheck registers the readiness predicate for a given chainID.
// Requests wait while the predicate returns false, and dispatch immediately
// when no predicate is registered.
func (r *Router) SetReadinessCheck(chainID string, fn func() bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.getOrCreateRouteLocked(chainID).ready = fn
}

// RemoveHandler permanently removes the chain route from the router. Requests
// waiting on the removed route observe the same not-found response as requests
// for an unknown chain.
func (r *Router) RemoveHandler(chainID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.routes, chainID)
}

// SetRootHandler sets the optional handler for '/'. If unset, '/' returns 404.
func (r *Router) SetRootHandler(h http.Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.root = h
}

// ServeHTTP routes requests to the chain-specific handler, after stripping the chain prefix.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	chainID, remainder := splitFirstSegment(req.URL.Path)
	if chainID == "" {
		r.mu.RLock()
		root := r.root
		r.mu.RUnlock()
		if root == nil {
			http.NotFound(w, req)
			return
		}
		root.ServeHTTP(w, req)
		return
	}

	r.mu.RLock()
	route := r.routes[chainID]
	var readyFn func() bool
	if route != nil {
		readyFn = route.ready
	}
	r.mu.RUnlock()
	if route == nil {
		http.NotFound(w, req)
		return
	}

	if readyFn != nil && !readyFn() {
		waitResult := r.waitReady(req.Context(), chainID)
		switch waitResult {
		case waitReady:
		case waitRemoved:
			http.NotFound(w, req)
			return
		case waitTimedOut:
			http.Error(w, "chain RPC not ready", http.StatusServiceUnavailable)
			return
		}
	}

	r.mu.RLock()
	route = r.routes[chainID]
	var h http.Handler
	if route != nil {
		h = route.handler
	}
	r.mu.RUnlock()
	if route == nil {
		http.NotFound(w, req)
		return
	}
	if h == nil {
		http.Error(w, "chain RPC not ready", http.StatusServiceUnavailable)
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

type waitReadyResult int

const (
	waitReady waitReadyResult = iota
	waitRemoved
	waitTimedOut
)

func (r *Router) waitReady(ctx context.Context, chainID string) waitReadyResult {
	gateCtx := ctx
	var cancel context.CancelFunc
	if r.gateTimeout > 0 {
		gateCtx, cancel = context.WithTimeout(ctx, r.gateTimeout)
		defer cancel()
	}

	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	for {
		r.mu.RLock()
		route := r.routes[chainID]
		var readyFn func() bool
		if route != nil {
			readyFn = route.ready
		}
		r.mu.RUnlock()
		if route == nil {
			return waitRemoved
		}
		if readyFn == nil || readyFn() {
			return waitReady
		}

		select {
		case <-ticker.C:
		case <-gateCtx.Done():
			return waitTimedOut
		}
	}
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
