package altda

import (
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type Metricer interface {
	RecordActiveChallenge(commBlock uint64, startBlock uint64, hash []byte)
	RecordResolvedChallenge(hash []byte)
	RecordExpiredChallenge(hash []byte)
	RecordChallengesHead(name string, num uint64)
	RecordStorageError()
}

type Metrics struct {
	ChallengesStatus *prometheus.GaugeVec
	ChallengesHead   *prometheus.GaugeVec

	StorageErrors *metrics.Event
}

var _ Metricer = (*Metrics)(nil)

func MakeMetrics(ns string, factory metrics.Factory) *Metrics {
	return &Metrics{
		ChallengesStatus: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "challenges_status",
			Help:      "Gauge representing the status of challenges synced",
		}, []string{"status"}),
		ChallengesHead: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "challenges_head",
			Help:      "Gauge representing the l1 heads of challenges synced",
		}, []string{"type"}),
		StorageErrors: metrics.NewEvent(factory, ns, "", "storage_errors", "errors when fetching or uploading to storage service"),
	}
}

func (m *Metrics) RecordChallenge(status string) {
	m.ChallengesStatus.WithLabelValues(status).Inc()
}

// RecordActiveChallenge records when a commitment is challenged including the block where the commitment
// is included, the block where the commitment was challenged and the commitment hash.
func (m *Metrics) RecordActiveChallenge(commBlock uint64, startBlock uint64, hash []byte) {
	m.RecordChallenge("active")
}

func (m *Metrics) RecordResolvedChallenge(hash []byte) {
	m.RecordChallenge("resolved")
}

func (m *Metrics) RecordExpiredChallenge(hash []byte) {
	m.RecordChallenge("expired")
}

func (m *Metrics) RecordStorageError() {
	m.StorageErrors.Record()
}

func (m *Metrics) RecordChallengesHead(name string, num uint64) {
	m.ChallengesHead.WithLabelValues(name).Set(float64(num))
}

type NoopMetrics struct{}

func (m *NoopMetrics) RecordActiveChallenge(commBlock uint64, startBlock uint64, hash []byte) {}
func (m *NoopMetrics) RecordResolvedChallenge(hash []byte)                                    {}
func (m *NoopMetrics) RecordExpiredChallenge(hash []byte)                                     {}
func (m *NoopMetrics) RecordChallengesHead(name string, num uint64)                           {}
func (m *NoopMetrics) RecordStorageError()                                                    {}
