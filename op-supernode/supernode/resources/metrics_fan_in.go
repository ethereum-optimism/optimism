package resources

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsFanIn is an http.handler
// which allows multiple Prometheus metrics
// "Gatherers" to be combined.
//
// The Gatherers must not collide with each other,
// e.g. each must have a unique name or label set.
// This can be accomplished by using a distinct,
// global label on each Gatherer.
type MetricsFanIn struct {
	mu      sync.RWMutex
	g       prometheus.Gatherers
	handler http.Handler
}

func NewMetricsFanIn(numGatherers int) *MetricsFanIn {
	emptyRegistry := prometheus.NewRegistry()
	return &MetricsFanIn{
		g:       make(prometheus.Gatherers, 0, numGatherers),
		handler: promhttp.HandlerFor(emptyRegistry, promhttp.HandlerOpts{})}
}

func (r *MetricsFanIn) AddRegistry(g prometheus.Gatherer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.g = append(r.g, g)
	r.handler = promhttp.HandlerFor(r.g, promhttp.HandlerOpts{})
}

func (r *MetricsFanIn) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	seg, _ := splitFirstSegment(req.URL.Path)
	if seg != "metrics" {
		http.NotFound(w, req)
		return
	}
	r.mu.RLock()
	r.handler.ServeHTTP(w, req)
	r.mu.RUnlock()
}
