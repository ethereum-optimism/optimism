package resources

import (
	"io"
	"net/http"
	"sync"

	gethlog "github.com/ethereum/go-ethereum/log"
)

// MetricsRouter multiplexes Prometheus metrics by the first path segment (chainID),
// and then requires the next segment to be 'metrics'. Effective paths:
//
//	/{chain}/metrics
//
// The wrapped handler is expected to serve on '/'.
type MetricsRouter struct {
	log   gethlog.Logger
	mu    sync.RWMutex
	paths map[string]http.Handler // chainID -> handler
	// optional resource closers
	closers []io.Closer
}

func NewMetricsRouter(log gethlog.Logger) *MetricsRouter {
	return &MetricsRouter{log: log, paths: make(map[string]http.Handler)}
}

func (r *MetricsRouter) Close() error {
	var firstErr error
	for _, c := range r.closers {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (r *MetricsRouter) SetHandler(chainID string, h http.Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.paths == nil {
		r.paths = make(map[string]http.Handler)
	}
	r.paths[chainID] = h
}

func (r *MetricsRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	chainID, remainder := splitFirstSegment(req.URL.Path)
	if chainID == "" {
		http.NotFound(w, req)
		return
	}
	// next segment must be 'metrics'
	seg, _ := splitFirstSegment(remainder)
	if seg != "metrics" {
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

	// Normalize path to '/'
	origPath := req.URL.Path
	origReqURI := req.RequestURI
	req.URL.Path = "/"
	if req.URL.RawQuery != "" {
		req.RequestURI = "/?" + req.URL.RawQuery
	} else {
		req.RequestURI = "/"
	}
	defer func() {
		req.URL.Path = origPath
		req.RequestURI = origReqURI
	}()
	h.ServeHTTP(w, req)
}
