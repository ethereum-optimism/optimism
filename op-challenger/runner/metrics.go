package runner

import (
	"time"

	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

const Namespace = "op_challenger_runner"

type Metrics struct {
	ns       string
	registry *prometheus.Registry
	factory  opmetrics.Factory
	*contractMetrics.ContractMetrics

	vmExecutionTime *prometheus.HistogramVec
	successTotal    *prometheus.CounterVec
	failuresTotal   *prometheus.CounterVec
	invalidTotal    *prometheus.CounterVec
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
	m.vmExecutionTime.WithLabelValues(vmType).Observe(dur.Seconds())
}

func (m *Metrics) RecordSuccess(vmType types.TraceType) {
	m.successTotal.WithLabelValues(vmType.String()).Inc()
}

func (m *Metrics) RecordFailure(vmType types.TraceType) {
	m.failuresTotal.WithLabelValues(vmType.String()).Inc()
}

func (m *Metrics) RecordInvalid(vmType types.TraceType) {
	m.invalidTotal.WithLabelValues(vmType.String()).Inc()
}
