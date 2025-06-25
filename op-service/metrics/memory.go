package metrics

import (
	"context"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/clock"
)

// LaunchMemoryMetrics starts a periodic collection of Go runtime memory stats
// and records the allocated heap bytes into the provided gauge. The gauge is
// expected to have Help text describing the metric. The returned closer stops
// the background loop.
func LaunchMemoryMetrics(log log.Logger, r *prometheus.Registry, ns string, g prometheus.Gauge) *clock.LoopFn {
	if g == nil {
		g = promauto.With(r).NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "memory_alloc_bytes",
			Help:      "Number of bytes allocated and still in use by the go runtime.",
		})
	}
	return clock.NewLoopFn(clock.SystemClock, func(ctx context.Context) {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		g.Set(float64(ms.Alloc))
	}, func() error {
		log.Info("memory metrics shutting down")
		return nil
	}, 10*time.Second)
}
