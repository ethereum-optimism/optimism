package resources

import (
	"io"
	"net/http"
	"sync"

	gethlog "github.com/ethereum/go-ethereum/log"
)

// MetricsRouter multiplexes Prometheus metrics by
// directing http requests to all registered handlers
type MetricsRouter struct {
	log      gethlog.Logger
	mu       sync.RWMutex
	handlers []http.Handler // one for each chain
	// optional resource closers
	closers []io.Closer
}

func NewMetricsRouter(log gethlog.Logger) *MetricsRouter {
	return &MetricsRouter{log: log, handlers: make([]http.Handler, 0)}
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

func (r *MetricsRouter) AddHandler(h http.Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers = append(r.handlers, h)
}

func (r *MetricsRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	seg, _ := splitFirstSegment(req.URL.Path)
	if seg != "metrics" {
		http.NotFound(w, req)
		return
	}
	r.mu.RLock()
	for _, h := range r.handlers {
		h.ServeHTTP(w, req)
	}
	r.mu.RUnlock()
}
