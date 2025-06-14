package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
)

type DerivationMetricer interface {
	SetDerivationIdle(status bool)
	RecordPipelineReset()
	RecordDerivationError()
	RecordDerivedBatches(batchType string)
	RecordChannelInputBytes(num int)
	RecordHeadChannelOpened()
	RecordChannelTimedOut()
	RecordFrame()
}

type DerivationMetrics struct {
	DerivationIdle    prometheus.Gauge
	PipelineResets    *metrics.Event
	DerivationErrors  *metrics.Event
	DerivedBatches    metrics.EventVec
	ChannelInputBytes prometheus.Counter

	headChannelOpenedEvent *metrics.Event
	channelTimedOutEvent   *metrics.Event
	frameAddedEvent        *metrics.Event
}

var _ DerivationMetricer = (*DerivationMetrics)(nil)

func NewDerivationMetrics(ns string, factory metrics.Factory) *DerivationMetrics {
	return &DerivationMetrics{
		DerivationIdle: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "derivation_idle",
			Help:      "1 if the derivation pipeline is idle",
		}),
		PipelineResets:   metrics.NewEvent(factory, ns, "", "pipeline_resets", "derivation pipeline resets"),
		DerivationErrors: metrics.NewEvent(factory, ns, "", "derivation_errors", "derivation errors"),
		DerivedBatches:   metrics.NewEventVec(factory, ns, "", "derived_batches", "derived batches", []string{"type"}),

		headChannelOpenedEvent: metrics.NewEvent(factory, ns, "", "head_channel", "New channel at the front of the channel bank"),
		channelTimedOutEvent:   metrics.NewEvent(factory, ns, "", "channel_timeout", "Channel has timed out"),
		frameAddedEvent:        metrics.NewEvent(factory, ns, "", "frame_added", "New frame ingested in the channel bank"),

		ChannelInputBytes: factory.NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "channel_input_bytes",
			Help:      "Number of compressed bytes added to the channel",
		}),
	}
}

func (m *DerivationMetrics) SetDerivationIdle(status bool) {
	var val float64
	if status {
		val = 1
	}
	m.DerivationIdle.Set(val)
}

func (m *DerivationMetrics) RecordPipelineReset() {
	m.PipelineResets.Record()
}

func (m *DerivationMetrics) RecordDerivationError() {
	m.DerivationErrors.Record()
}

func (m *DerivationMetrics) RecordDerivedBatches(batchType string) {
	m.DerivedBatches.Record(batchType)
}

func (m *DerivationMetrics) RecordChannelInputBytes(inputCompressedBytes int) {
	m.ChannelInputBytes.Add(float64(inputCompressedBytes))
}

func (m *DerivationMetrics) RecordHeadChannelOpened() {
	m.headChannelOpenedEvent.Record()
}

func (m *DerivationMetrics) RecordChannelTimedOut() {
	m.channelTimedOutEvent.Record()
}

func (m *DerivationMetrics) RecordFrame() {
	m.frameAddedEvent.Record()
}

type NoopDerivationMetrics struct{}

var _ DerivationMetricer = NoopDerivationMetrics{}

func (NoopDerivationMetrics) SetDerivationIdle(status bool) {}

func (NoopDerivationMetrics) RecordPipelineReset() {}

func (NoopDerivationMetrics) RecordDerivationError() {}

func (NoopDerivationMetrics) RecordDerivedBatches(batchType string) {}

func (NoopDerivationMetrics) RecordChannelInputBytes(int) {}

func (NoopDerivationMetrics) RecordHeadChannelOpened() {}

func (NoopDerivationMetrics) RecordChannelTimedOut() {}

func (NoopDerivationMetrics) RecordFrame() {}
