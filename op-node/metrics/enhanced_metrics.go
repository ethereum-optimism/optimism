// Package metrics provides enhanced metrics for op-node monitoring.
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// EnhancedMetrics extends the base Metrics with additional monitoring capabilities.
type EnhancedMetrics struct {
	*Metrics
	
	// Additional latency metrics
	L1RequestLatency      *prometheus.HistogramVec
	L2RequestLatency      *prometheus.HistogramVec
	ConfigReloadDuration  prometheus.Histogram
	
	// Resource utilization metrics
	MemoryUsage           prometheus.Gauge
	CPUUsage             prometheus.Gauge
	GoroutineCount        prometheus.Gauge
	
	// Health check metrics
	HealthStatus          prometheus.Gauge
	LastHealthCheckTime   prometheus.Gauge
	
	// Performance metrics
	BlocksPerSecond       prometheus.Gauge
	TransactionsPerSecond prometheus.Gauge
	
	// Error rate metrics
	ErrorRate             prometheus.Gauge
	ErrorRate1Min         prometheus.Gauge
	ErrorRate5Min         prometheus.Gauge
}

// NewEnhancedMetrics creates a new EnhancedMetrics instance.
func NewEnhancedMetrics(base *Metrics) *EnhancedMetrics {
	if base == nil {
		base = NewMetrics("default")
	}
	
	factory := base.factory
	ns := base.factory.Namespace()
	
	return &EnhancedMetrics{
		Metrics: base,
		
		L1RequestLatency: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "l1_request_latency_seconds",
			Help:      "L1 request latency in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10),
		}, []string{"endpoint", "method"}),
		
		L2RequestLatency: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "l2_request_latency_seconds",
			Help:      "L2 request latency in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10),
		}, []string{"endpoint", "method"}),
		
		ConfigReloadDuration: factory.NewHistogram(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "config_reload_duration_seconds",
			Help:      "Configuration reload duration in seconds",
		}),
		
		MemoryUsage: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "memory_usage_bytes",
			Help:      "Memory usage in bytes",
		}),
		
		CPUUsage: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "cpu_usage_percent",
			Help:      "CPU usage percentage",
		}),
		
		GoroutineCount: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "goroutine_count",
			Help:      "Number of goroutines",
		}),
		
		HealthStatus: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "health_status",
			Help:      "Health status (1 = healthy, 0 = unhealthy)",
		}),
		
		LastHealthCheckTime: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "last_health_check_time_seconds",
			Help:      "Last health check time as Unix timestamp",
		}),
		
		BlocksPerSecond: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "blocks_per_second",
			Help:      "Blocks processed per second",
		}),
		
		TransactionsPerSecond: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "transactions_per_second",
			Help:      "Transactions processed per second",
		}),
		
		ErrorRate: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "error_rate",
			Help:      "Current error rate",
		}),
		
		ErrorRate1Min: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "error_rate_1min",
			Help:      "Error rate over 1 minute",
		}),
		
		ErrorRate5Min: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "error_rate_5min",
			Help:      "Error rate over 5 minutes",
		}),
	}
}

// RecordL1RequestLatency records the latency of an L1 request.
func (m *EnhancedMetrics) RecordL1RequestLatency(endpoint, method string, duration time.Duration) {
	m.L1RequestLatency.WithLabelValues(endpoint, method).Observe(duration.Seconds())
}

// RecordL2RequestLatency records the latency of an L2 request.
func (m *EnhancedMetrics) RecordL2RequestLatency(endpoint, method string, duration time.Duration) {
	m.L2RequestLatency.WithLabelValues(endpoint, method).Observe(duration.Seconds())
}

// RecordConfigReloadDuration records the duration of a config reload.
func (m *EnhancedMetrics) RecordConfigReloadDuration(duration time.Duration) {
	m.ConfigReloadDuration.Observe(duration.Seconds())
}

// SetHealthStatus sets the health status (1 = healthy, 0 = unhealthy).
func (m *EnhancedMetrics) SetHealthStatus(healthy bool) {
	status := 0.0
	if healthy {
		status = 1.0
	}
	m.HealthStatus.Set(status)
	m.LastHealthCheckTime.SetToCurrentTime()
}

// SetBlocksPerSecond sets the blocks per second metric.
func (m *EnhancedMetrics) SetBlocksPerSecond(blocks float64) {
	m.BlocksPerSecond.Set(blocks)
}

// SetTransactionsPerSecond sets the transactions per second metric.
func (m *EnhancedMetrics) SetTransactionsPerSecond(txs float64) {
	m.TransactionsPerSecond.Set(txs)
}

// SetErrorRate sets the current error rate.
func (m *EnhancedMetrics) SetErrorRate(rate float64) {
	m.ErrorRate.Set(rate)
}

// SetErrorRate1Min sets the 1-minute error rate.
func (m *EnhancedMetrics) SetErrorRate1Min(rate float64) {
	m.ErrorRate1Min.Set(rate)
}

// SetErrorRate5Min sets the 5-minute error rate.
func (m *EnhancedMetrics) SetErrorRate5Min(rate float64) {
	m.ErrorRate5Min.Set(rate)
}

