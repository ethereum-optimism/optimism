package claimfollow

import (
	"github.com/prometheus/client_golang/prometheus"

	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
)

// Metrics is what the follow module reports about itself.
//
// It is an interface rather than a struct of prometheus handles so that the state machine's tests
// never need a registry, and so that each thing an operator watches is a named method rather than a
// label somebody could rename.
type Metrics interface {
	// RecordClaim counts claims accepted from the rendering chain.
	RecordClaim()
	// RecordRejectedClaim counts registry-addressed transactions that yielded no usable claim, by
	// reason. "selector" and "decode" are a stranger's bytes; "reverted" is the operator's own
	// transaction refused by the operator's own registry, which under snap-to-commitment is a skip
	// rather than a halt — so this counter is the ONLY signal that it happened, and any increase in
	// the reverted series is an operator incident.
	RecordRejectedClaim(reason string)
	// RecordRenderingReorg counts rewinds forced by the rendering chain moving under the cursor.
	RecordRenderingReorg()
	// RecordSafe and RecordFinalized report the served labels' heights.
	RecordSafe(height uint64)
	RecordFinalized(height uint64)
}

// NoopMetrics discards everything. It is the default when New is passed a nil Metrics.
type NoopMetrics struct{}

func (NoopMetrics) RecordClaim()               {}
func (NoopMetrics) RecordRejectedClaim(string) {}
func (NoopMetrics) RecordRenderingReorg()      {}
func (NoopMetrics) RecordSafe(uint64)          {}
func (NoopMetrics) RecordFinalized(uint64)     {}

var _ Metrics = NoopMetrics{}

// PromMetrics is the prometheus-backed Metrics.
type PromMetrics struct {
	claims      prometheus.Counter
	rejected    *prometheus.CounterVec
	reorgs      prometheus.Counter
	safeHeight  prometheus.Gauge
	finalHeight prometheus.Gauge
}

var _ Metrics = (*PromMetrics)(nil)

// NewPromMetrics registers the module's metrics under the given namespace.
func NewPromMetrics(factory opmetrics.Factory, ns string) *PromMetrics {
	m := &PromMetrics{
		claims: factory.NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "claim_follow_claims_total",
			Help:      "Claims read from the rendering chain and accepted.",
		}),
		rejected: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "claim_follow_rejected_claims_total",
			Help:      "Registry-addressed transactions that yielded no usable claim, by reason. Any 'reverted' is an operator incident.",
		}, []string{"reason"}),
		reorgs: factory.NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "claim_follow_rendering_reorgs_total",
			Help:      "Times the rendering chain moved under the scan cursor and forced a rewind.",
		}),
		safeHeight: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "claim_follow_safe_height",
			Help:      "Private-chain block number served as local_safe_l2 and safe_l2.",
		}),
		finalHeight: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "claim_follow_finalized_height",
			Help:      "Private-chain block number served as finalized_l2.",
		}),
	}
	// Pre-create every rejection series so a dashboard shows a flat zero rather than a gap for a
	// reason that has not happened yet. "reverted" is the one that matters.
	for _, reason := range []string{"selector", "decode", "reverted"} {
		m.rejected.WithLabelValues(reason)
	}
	return m
}

func (m *PromMetrics) RecordClaim()                      { m.claims.Inc() }
func (m *PromMetrics) RecordRejectedClaim(reason string) { m.rejected.WithLabelValues(reason).Inc() }
func (m *PromMetrics) RecordRenderingReorg()             { m.reorgs.Inc() }
func (m *PromMetrics) RecordSafe(height uint64)          { m.safeHeight.Set(float64(height)) }
func (m *PromMetrics) RecordFinalized(height uint64)     { m.finalHeight.Set(float64(height)) }
