package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// CacheMetrics implements the Metrics interface in the caching package,
// implementing reusable metrics for different caches.
type CacheMetrics struct {
	SizeVec *prometheus.GaugeVec
	GetVec  *prometheus.CounterVec
	AddVec  *prometheus.CounterVec
}

// CacheAdd meters the addition of an item with a given type to the cache,
// metering the change of the cache size of that type, and indicating a corresponding eviction if any.
func (m *CacheMetrics) CacheAdd(typeLabel string, typeCacheSize int, evicted bool) {
	m.SizeVec.WithLabelValues(typeLabel).Set(float64(typeCacheSize))
	if evicted {
		m.AddVec.WithLabelValues(typeLabel, "true").Inc()
	} else {
		m.AddVec.WithLabelValues(typeLabel, "false").Inc()
	}
}

// CacheGet meters a lookup of an item with a given type to the cache
// and indicating if the lookup was a hit.
func (m *CacheMetrics) CacheGet(typeLabel string, hit bool) {
	if hit {
		m.GetVec.WithLabelValues(typeLabel, "true").Inc()
	} else {
		m.GetVec.WithLabelValues(typeLabel, "false").Inc()
	}
}

func NewCacheMetrics(factory Factory, ns string, name string, displayName string) *CacheMetrics {
	return &CacheMetrics{
		SizeVec: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      name + "_size",
			Help:      displayName + " cache size",
		}, []string{
			"type",
		}),
		GetVec: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      name + "_get",
			Help:      displayName + " lookups, hitting or not",
		}, []string{
			"type",
			"hit",
		}),
		AddVec: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      name + "_add",
			Help:      displayName + " additions, evicting previous values or not",
		}, []string{
			"type",
			"evicted",
		}),
	}
}
