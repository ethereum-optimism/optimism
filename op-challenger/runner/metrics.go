package runner

import (
	"time"

	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

const Namespace = "op_challenger_runner"

type Metrics struct {
	ns       string
	registry *prometheus.Registry
	factory  opmetrics.Factory
	*contractMetrics.ContractMetrics

	vmExecutionTime     *prometheus.HistogramVec
	vmLastExecutionTime *prometheus.GaugeVec
	vmMemoryUsed        *prometheus.HistogramVec
	vmLastMemoryUsed    *prometheus.GaugeVec
	successTotal        *prometheus.CounterVec
	failuresTotal       *prometheus.CounterVec
	invalidTotal        *prometheus.CounterVec
}

var _ Metricer = (*Metrics)(nil)

// Metrics implementation must implement RegistryMetricer to allow the metrics server to work.
var _ opmetrics.RegistryMetricer = (*Metrics)(nil)

func NewMetrics() *Metrics {
	registry := opmetrics.NewRegistry()
	factory := opmetrics.With(registry)

	return &Metrics{
		ns:       Namespace,
		registry: registry,
		factory:  factory,

		ContractMetrics: contractMetrics.MakeContractMetrics(Namespace, factory),

		vmExecutionTime: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: Namespace,
			Name:      "vm_execution_time",
			Help:      "Time (in seconds) to execute the fault proof VM",
			Buckets: append(
				[]float64{1.0, 10.0},
				prometheus.ExponentialBuckets(30.0, 2.0, 14)...),
		}, []string{"vm"}),
		vmLastExecutionTime: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "vm_last_execution_time",
			Help:      "Time (in seconds) taken for the last execution of the fault proof VM",
		}, []string{"vm"}),
		vmMemoryUsed: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: Namespace,
			Name:      "vm_memory_used",
			Help:      "Memory used (in bytes) to execute the fault proof VM",
			// 100MiB increments from 0 to 1.5GiB
			Buckets: prometheus.LinearBuckets(0, 1024*1024*100, 15),
		}, []string{"vm"}),
		vmLastMemoryUsed: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "vm_last_memory_used",
			Help:      "Memory used (in bytes) for the last execution of the fault proof VM",
		}, []string{"vm"}),
		successTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "success_total",
			Help:      "Number of VM executions that successfully verified the output root",
		}, []string{"type"}),
		failuresTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "failures_total",
			Help:      "Number of failures to execute a VM",
		}, []string{"type"}),
		invalidTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "invalid_total",
			Help:      "Number of runs that determined the output root was invalid",
		}, []string{"type"}),
	}
}

func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

func (m *Metrics) RecordVmExecutionTime(vmType string, dur time.Duration) {
	val := dur.Seconds()
	m.vmExecutionTime.WithLabelValues(vmType).Observe(val)
	m.vmLastExecutionTime.WithLabelValues(vmType).Set(val)
}

func (m *Metrics) RecordVmMemoryUsed(vmType string, memoryUsed uint64) {
	m.vmMemoryUsed.WithLabelValues(vmType).Observe(float64(memoryUsed))
	m.vmLastMemoryUsed.WithLabelValues(vmType).Set(float64(memoryUsed))
}

func (m *Metrics) RecordSuccess(vmType string) {
	m.successTotal.WithLabelValues(vmType).Inc()
}

func (m *Metrics) RecordFailure(vmType string) {
	m.failuresTotal.WithLabelValues(vmType).Inc()
}

func (m *Metrics) RecordInvalid(vmType string) {
	m.invalidTotal.WithLabelValues(vmType).Inc()
}
