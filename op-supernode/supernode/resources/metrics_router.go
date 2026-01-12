package resources

import (
	"io"
	"net/http"
	"sync"

	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsRouter allows multiple Prometheus metrics
// "Gatherers"
//
//	to be served by a single HTTP server.
//
// The Gatherers must not collide with each other,
// e.g. each must have a unique name or label set.
// This can be accomplished by using a distinct,
// global label on each Gatherer.
type MetricsRouter struct {
	log     gethlog.Logger
	mu      sync.RWMutex
	g       prometheus.Gatherers
	handler http.Handler
	// optional resource closers
	closers []io.Closer
}

func NewMetricsRouter(log gethlog.Logger) *MetricsRouter {
	// TODO could accept number of gatherers as constructor argument
	return &MetricsRouter{log: log, g: make([]prometheus.Gatherer, 0)}
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

func (r *MetricsRouter) AddRegistry(g prometheus.Gatherer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.g = append(r.g, g)
	r.handler = promhttp.HandlerFor(r.g, promhttp.HandlerOpts{})
}

func (r *MetricsRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	seg, _ := splitFirstSegment(req.URL.Path)
	if seg != "metrics" {
		http.NotFound(w, req)
		return
	}
	r.mu.RLock()
	r.handler.ServeHTTP(w, req)
	r.mu.RUnlock()
}
