package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
)

type PayloadsMetricer interface {
	RecordReceivedUnsafePayload(chainID eth.ChainID, payload *eth.ExecutionPayloadEnvelope)
	RecordUnsafePayloadsBuffer(chainID eth.ChainID, length uint64, memSize uint64, next eth.BlockRef)
}

type PayloadsMetrics struct {
	UnsafePayloads              *metrics.Event
	UnsafePayloadsBufferLen     *prometheus.GaugeVec
	UnsafePayloadsBufferMemSize *prometheus.GaugeVec
}

var _ PayloadsMetricer = (*PayloadsMetrics)(nil)

func NewPayloadsMetrics(ns string, factory metrics.Factory) *PayloadsMetrics {
	return &PayloadsMetrics{
		// TODO: also per chain?
		UnsafePayloads: metrics.NewEvent(factory, ns, "", "unsafe_payloads", "unsafe payloads"),

		UnsafePayloadsBufferLen: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "unsafe_payloads_buffer_len",
			Help:      "Number of buffered L2 unsafe payloads",
		}, []string{"chain"}),
		UnsafePayloadsBufferMemSize: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "unsafe_payloads_buffer_mem_size",
			Help:      "Total estimated memory size of buffered L2 unsafe payloads",
		}, []string{"chain"}),
	}
}

func (m *PayloadsMetrics) RecordReceivedUnsafePayload(chainID eth.ChainID, payload *eth.ExecutionPayloadEnvelope) {
	m.UnsafePayloads.Record()
}

func (m *PayloadsMetrics) RecordUnsafePayloadsBuffer(chainID eth.ChainID, length uint64, memSize uint64, next eth.BlockRef) {
	m.UnsafePayloadsBufferLen.WithLabelValues(chainIDLabel(chainID)).Set(float64(length))
	m.UnsafePayloadsBufferMemSize.WithLabelValues(chainIDLabel(chainID)).Set(float64(memSize))
}

type NoopPayloadsMetrics struct{}

var _ PayloadsMetricer = NoopPayloadsMetrics{}

func (NoopPayloadsMetrics) RecordReceivedUnsafePayload(chainID eth.ChainID, payload *eth.ExecutionPayloadEnvelope) {
}

func (NoopPayloadsMetrics) RecordUnsafePayloadsBuffer(chainID eth.ChainID, length uint64, memSize uint64, next eth.BlockRef) {
}
