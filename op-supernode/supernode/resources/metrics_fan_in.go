package resources

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsFanIn is an http.handler
// which allows multiple Prometheus metrics
// "Gatherers" to be combined and served at the
// /metrics path.
//
// The Gatherers must not collide with each other,
// e.g. each must have a unique name or label set.
// This can be accomplished by using a distinct,
// global label on each Gatherer.
type MetricsFanIn struct {
	mu           sync.RWMutex
	numGatherers int
	gm           map[string]prometheus.Gatherer // keyed by decimal chain ID
	handler      http.Handler
}

func NewMetricsFanIn(numGatherers int) *MetricsFanIn {
	emptyRegistry := prometheus.NewRegistry()
	return &MetricsFanIn{
		numGatherers: numGatherers,
		gm:           make(map[string]prometheus.Gatherer),
		handler:      promhttp.HandlerFor(emptyRegistry, promhttp.HandlerOpts{})}
}

func (r *MetricsFanIn) SetMetricsRegistry(key string, g prometheus.Gatherer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gm[key] = g

	gs := make(prometheus.Gatherers, 0, r.numGatherers)
	for _, gr := range r.gm {
		gs = append(gs, gr)
	}

	r.handler = promhttp.HandlerFor(gs, promhttp.HandlerOpts{})
}

func (r *MetricsFanIn) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	seg, _ := splitFirstSegment(req.URL.Path)
	if seg != "metrics" {
		http.NotFound(w, req)
		return
	}
	var handler http.Handler
	r.mu.RLock()
	handler = r.handler
	r.mu.RUnlock()
	handler.ServeHTTP(w, req)
}
