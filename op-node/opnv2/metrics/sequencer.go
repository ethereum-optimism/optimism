package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
)

type SequencerMetricer interface {
	SetSequencerState(active bool)
	RecordSequencingError()
	RecordPublishingError()

	RecordSequencerInconsistentL1Origin(from eth.BlockID, to eth.BlockID)
	RecordSequencerReset()

	CountSequencedTxsInBlock(chainID eth.ChainID, txns int, deposits int)
	RecordSequencerBuildingDiffTime(chainID eth.ChainID, duration time.Duration)
	RecordSequencerSealingTime(chainID eth.ChainID, duration time.Duration)
}

type SequencerMetrics struct {
	SequencerInconsistentL1Origin *metrics.Event
	SequencerResets               *metrics.Event

	SequencerBuildingDiffDurationSeconds *prometheus.HistogramVec
	SequencerBuildingDiffTotal           *prometheus.CounterVec

	SequencerSealingDurationSeconds *prometheus.HistogramVec
	SequencerSealingTotal           *prometheus.CounterVec

	TransactionsSequencedTotal *prometheus.CounterVec

	SequencingErrors *metrics.Event
	PublishingErrors *metrics.Event

	SequencerActive prometheus.Gauge
}

var _ SequencerMetricer = (*SequencerMetrics)(nil)

func NewSequencerMetrics(ns string, factory metrics.Factory) *SequencerMetrics {
	return &SequencerMetrics{
		SequencingErrors: metrics.NewEvent(factory, ns, "", "sequencing_errors", "sequencing errors"),
		PublishingErrors: metrics.NewEvent(factory, ns, "", "publishing_errors", "p2p publishing errors"),
		SequencerActive: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "sequencer_active",
			Help:      "1 if sequencer active, 0 otherwise",
		}),

		SequencerInconsistentL1Origin: metrics.NewEvent(factory, ns, "", "sequencer_inconsistent_l1_origin", "events when the sequencer selects an inconsistent L1 origin"),
		SequencerResets:               metrics.NewEvent(factory, ns, "", "sequencer_resets", "sequencer resets"),

		TransactionsSequencedTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "transactions_sequenced_total",
			Help:      "Count of total transactions sequenced",
		}, []string{"type", "chainID"}),

		SequencerBuildingDiffDurationSeconds: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "sequencer_building_diff_seconds",
			Buckets: []float64{
				-10, -5, -2.5, -1, -.5, -.25, -.1, -0.05, -0.025, -0.01, -0.005,
				.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			Help: "Histogram of Sequencer building time, minus block time",
		}, []string{"chainID"}),
		SequencerBuildingDiffTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "sequencer_building_diff_total",
			Help:      "Number of sequencer block building jobs",
		}, []string{"chainID"}),
		SequencerSealingDurationSeconds: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "sequencer_sealing_seconds",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			Help:      "Histogram of Sequencer block sealing time",
		}, []string{"chainID"}),
		SequencerSealingTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "sequencer_sealing_total",
			Help:      "Number of sequencer block sealing jobs",
		}, []string{"chainID"}),
	}
}

func (m *SequencerMetrics) SetSequencerState(active bool) {
	var val float64
	if active {
		val = 1
	}
	m.SequencerActive.Set(val)
}

func (m *SequencerMetrics) RecordSequencingError() {
	m.SequencingErrors.Record()
}

func (m *SequencerMetrics) RecordPublishingError() {
	m.PublishingErrors.Record()
}

func (m *SequencerMetrics) CountSequencedTxsInBlock(chainID eth.ChainID, txns int, deposits int) {
	m.TransactionsSequencedTotal.WithLabelValues("deposits", chainIDLabel(chainID)).Add(float64(deposits))
	m.TransactionsSequencedTotal.WithLabelValues("txns", chainIDLabel(chainID)).Add(float64(txns - deposits))
}

func (m *SequencerMetrics) RecordSequencerInconsistentL1Origin(from eth.BlockID, to eth.BlockID) {
	m.SequencerInconsistentL1Origin.Record()
}

func (m *SequencerMetrics) RecordSequencerReset() {
	m.SequencerResets.Record()
}

// RecordSequencerBuildingDiffTime tracks the amount of time the sequencer was allowed between
// start to finish, incl. sealing, minus the block time.
// Ideally this is 0, realistically the sequencer scheduler may be busy with other jobs like syncing sometimes.
func (m *SequencerMetrics) RecordSequencerBuildingDiffTime(chainID eth.ChainID, duration time.Duration) {
	m.SequencerBuildingDiffTotal.WithLabelValues(chainIDLabel(chainID)).Inc()
	m.SequencerBuildingDiffDurationSeconds.WithLabelValues(chainIDLabel(chainID)).Observe(float64(duration) / float64(time.Second))
}

// RecordSequencerSealingTime tracks the amount of time the sequencer took to finish sealing the block.
// Ideally this is 0, realistically it may take some time.
func (m *SequencerMetrics) RecordSequencerSealingTime(chainID eth.ChainID, duration time.Duration) {
	m.SequencerSealingTotal.WithLabelValues(chainIDLabel(chainID)).Inc()
	m.SequencerSealingDurationSeconds.WithLabelValues(chainIDLabel(chainID)).Observe(float64(duration) / float64(time.Second))
}

type NoopSequencerMetrics struct{}

var _ SequencerMetricer = NoopSequencerMetrics{}

func (NoopSequencerMetrics) SetSequencerState(active bool) {}

func (NoopSequencerMetrics) RecordSequencingError() {}

func (NoopSequencerMetrics) RecordPublishingError() {}

func (NoopSequencerMetrics) CountSequencedTxsInBlock(chainID eth.ChainID, txns int, deposits int) {}

func (NoopSequencerMetrics) RecordSequencerInconsistentL1Origin(from eth.BlockID, to eth.BlockID) {}

func (NoopSequencerMetrics) RecordSequencerReset() {}

func (NoopSequencerMetrics) RecordSequencerBuildingDiffTime(chainID eth.ChainID, duration time.Duration) {
}

func (NoopSequencerMetrics) RecordSequencerSealingTime(chainID eth.ChainID, duration time.Duration) {}
