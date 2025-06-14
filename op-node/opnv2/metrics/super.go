package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
)

type SuperMetricer interface {
	RecordAccessListVerifyFailure(chainID eth.ChainID)
}

type SuperMetrics struct {
	AccessListVerifyFailureVec *prometheus.CounterVec
}

var _ SuperMetricer = (*SuperMetrics)(nil)

func NewSuperMetrics(ns string, factory metrics.Factory) *SuperMetrics {
	return &SuperMetrics{
		AccessListVerifyFailureVec: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "access_list_verify_failure",
			Help:      "Number of access list verify failures",
		}, []string{
			"chain",
		}),
	}
}

func (m *SuperMetrics) RecordAccessListVerifyFailure(chainID eth.ChainID) {
	m.AccessListVerifyFailureVec.WithLabelValues(chainIDLabel(chainID)).Inc()
}

type NoopSuperMetrics struct{}

var _ SuperMetricer = NoopSuperMetrics{}

func (NoopSuperMetrics) RecordAccessListVerifyFailure(_ eth.ChainID) {}
