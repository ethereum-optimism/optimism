package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
)

type CacheMetricer interface {
	CacheAdd(chainID eth.ChainID, label string, cacheSize int, evicted bool)
	CacheGet(chainID eth.ChainID, label string, hit bool)
}

type CacheMetrics struct {

	// From op-node (no longer used)
	//L1SourceCache *metrics.CacheMetrics
	//L2SourceCache *metrics.CacheMetrics

	// From op-supervisor
	CacheSizeVec *prometheus.GaugeVec
	CacheGetVec  *prometheus.CounterVec
	CacheAddVec  *prometheus.CounterVec
}

var _ CacheMetricer = (*CacheMetrics)(nil)

func NewCacheMetrics(ns string, factory metrics.Factory) *CacheMetrics {
	return &CacheMetrics{
		//L1SourceCache: metrics.NewCacheMetrics(factory, ns, "l1_source_cache", "L1 Source cache"),
		//L2SourceCache: metrics.NewCacheMetrics(factory, ns, "l2_source_cache", "L2 Source cache"),

		CacheSizeVec: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "source_rpc_cache_size",
			Help:      "Source rpc cache cache size",
		}, []string{
			"chain",
			"type",
		}),
		CacheGetVec: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "source_rpc_cache_get",
			Help:      "Source rpc cache lookups, hitting or not",
		}, []string{
			"chain",
			"type",
			"hit",
		}),
		CacheAddVec: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "source_rpc_cache_add",
			Help:      "Source rpc cache additions, evicting previous values or not",
		}, []string{
			"chain",
			"type",
			"evicted",
		}),
	}
}

func (m *CacheMetrics) CacheAdd(chainID eth.ChainID, label string, cacheSize int, evicted bool) {
	chain := chainIDLabel(chainID)
	m.CacheSizeVec.WithLabelValues(chain, label).Set(float64(cacheSize))
	if evicted {
		m.CacheAddVec.WithLabelValues(chain, label, "true").Inc()
	} else {
		m.CacheAddVec.WithLabelValues(chain, label, "false").Inc()
	}
}

func (m *CacheMetrics) CacheGet(chainID eth.ChainID, label string, hit bool) {
	chain := chainIDLabel(chainID)
	if hit {
		m.CacheGetVec.WithLabelValues(chain, label, "true").Inc()
	} else {
		m.CacheGetVec.WithLabelValues(chain, label, "false").Inc()
	}
}

type NoopCacheMetrics struct{}

var _ CacheMetricer = NoopCacheMetrics{}

func (NoopCacheMetrics) CacheAdd(_ eth.ChainID, _ string, _ int, _ bool) {}
func (NoopCacheMetrics) CacheGet(_ eth.ChainID, _ string, _ bool)        {}
