package runner

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
)

const Namespace = "op_challenger_runner"

type Metrics struct {
	ns       string
	registry *prometheus.Registry
	factory  opmetrics.Factory
	*contractMetrics.ContractMetrics
	*metrics.VmMetrics
	opmetrics.RPCMetrics

	up                              prometheus.Gauge
	vmLastExecutionTime             *prometheus.GaugeVec
	vmLastMemoryUsed                *prometheus.GaugeVec
	successTotal                    *prometheus.CounterVec
	setupFailuresTotal              *prometheus.CounterVec
	consecutiveSetupFailuresCurrent *prometheus.GaugeVec
	vmFailuresTotal                 *prometheus.CounterVec
}

// Reason labels for vmFailuresTotal metric
const (
	ReasonIncorrectStatus = "incorrect_status"
	ReasonPanic           = "panic"
	ReasonTimeout         = "timeout"
)

var _ Metricer = (*Metrics)(nil)

// Metrics implementation must implement RegistryMetricer to allow the metrics server to work.
var _ opmetrics.RegistryMetricer = (*Metrics)(nil)

func NewMetrics(runConfigs []RunConfig) *Metrics {
	registry := opmetrics.NewRegistry()
	factory := opmetrics.With(registry)

	metrics := &Metrics{
		ns:       Namespace,
		registry: registry,
		factory:  factory,

		ContractMetrics: contractMetrics.MakeContractMetrics(Namespace, factory),
		VmMetrics:       metrics.NewVmMetrics(Namespace, factory),
		RPCMetrics:      opmetrics.MakeRPCMetrics(Namespace, factory),

		up: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "up",
			Help:      "The VM runner has started to run",
		}),
		vmLastExecutionTime: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "vm_last_execution_time",
			Help:      "Time (in seconds) taken for the last execution of the fault proof VM",
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
		setupFailuresTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "setup_failures_total",
			Help:      "Number of setup failures before VM execution",
		}, []string{"type"}),
		consecutiveSetupFailuresCurrent: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "consecutive_setup_failures_current",
			Help:      "Number of consecutive setup failures by VM type. Resets to 0 on any complete run.",
		}, []string{"type"}),
		vmFailuresTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "vm_failures_total",
			Help:      "Number of VM execution failures by type and reason (incorrect_status, panic, timeout)",
		}, []string{"type", "reason"}),
	}

	for _, runConfig := range runConfigs {
		metrics.successTotal.WithLabelValues(runConfig.Name).Add(0)
		metrics.setupFailuresTotal.WithLabelValues(runConfig.Name).Add(0)
		metrics.consecutiveSetupFailuresCurrent.WithLabelValues(runConfig.Name).Set(0)
		metrics.vmFailuresTotal.WithLabelValues(runConfig.Name, ReasonIncorrectStatus).Add(0)
		metrics.vmFailuresTotal.WithLabelValues(runConfig.Name, ReasonPanic).Add(0)
		metrics.vmFailuresTotal.WithLabelValues(runConfig.Name, ReasonTimeout).Add(0)
		metrics.RecordUp()
	}

	return metrics
}

func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

func (m *Metrics) RecordUp() {
	m.up.Set(1)
}

func (m *Metrics) RecordVmExecutionTime(vmType string, dur time.Duration) {
	val := dur.Seconds()
	m.VmMetrics.RecordVmExecutionTime(vmType, dur)
	m.vmLastExecutionTime.WithLabelValues(vmType).Set(val)
}

func (m *Metrics) RecordVmMemoryUsed(vmType string, memoryUsed uint64) {
	m.VmMetrics.RecordVmMemoryUsed(vmType, memoryUsed)
	m.vmLastMemoryUsed.WithLabelValues(vmType).Set(float64(memoryUsed))
}

func (m *Metrics) RecordSuccess(vmType string) {
	m.successTotal.WithLabelValues(vmType).Inc()
	m.consecutiveSetupFailuresCurrent.WithLabelValues(vmType).Set(0)
}

func (m *Metrics) RecordSetupFailure(vmType string) {
	m.setupFailuresTotal.WithLabelValues(vmType).Inc()
	m.consecutiveSetupFailuresCurrent.WithLabelValues(vmType).Inc()
}

func (m *Metrics) RecordVmFailure(vmType string, reason string) {
	m.vmFailuresTotal.WithLabelValues(vmType, reason).Inc()
	m.consecutiveSetupFailuresCurrent.WithLabelValues(vmType).Set(0)
}
