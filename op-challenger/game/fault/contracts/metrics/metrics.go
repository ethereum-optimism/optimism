package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const ContractsSubsystem = "contracts"

type EndTimer func()

type Factory interface {
	NewCounterVec(opts prometheus.CounterOpts, labelNames []string) *prometheus.CounterVec
	NewHistogramVec(opts prometheus.HistogramOpts, labelNames []string) *prometheus.HistogramVec
}

type ContractMetricer interface {
	StartContractRequest(name string) EndTimer
}

type ContractMetrics struct {
	ContractRequestsTotal          *prometheus.CounterVec
	ContractRequestDurationSeconds *prometheus.HistogramVec
}

var _ ContractMetricer = (*ContractMetrics)(nil)

func MakeContractMetrics(ns string, factory Factory) *ContractMetrics {
	return &ContractMetrics{
		ContractRequestsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: ContractsSubsystem,
			Name:      "requests_total",
			Help:      "Total requests to the contracts",
		}, []string{
			"method",
		}),
		ContractRequestDurationSeconds: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: ContractsSubsystem,
			Name:      "requests_duration_seconds",
			Help:      "Histogram of contract request durations",
		}, []string{
			"method",
		}),
	}
}

func (m *ContractMetrics) StartContractRequest(method string) EndTimer {
	m.ContractRequestsTotal.WithLabelValues(method).Inc()
	timer := prometheus.NewTimer(m.ContractRequestDurationSeconds.WithLabelValues(method))
	return func() {
		timer.ObserveDuration()
	}
}
