package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
)

type Metricer interface {
	Document() []opmetrics.DocumentedMetric

	opmetrics.RPCMetricer
	event.Metrics

	CacheMetricer
	DBMetricer
	DerivationMetricer
	L1Metricer
	P2PMetricer
	PayloadsMetricer
	SuperRefMetricer
	SequencerMetricer
	ServiceMetricer
	SuperMetricer
}

type Metrics struct {
	registry *prometheus.Registry
	factory  opmetrics.Factory

	*event.EventMetricsTracker
	opmetrics.RPCMetrics

	*CacheMetrics
	*DBMetrics
	*DerivationMetrics
	*L1Metrics
	*P2PMetrics
	*PayloadsMetrics
	*SuperRefMetrics
	*SequencerMetrics
	*ServiceMetrics
	*SuperMetrics
}

var _ Metricer = (*Metrics)(nil)

// implements the Registry getter, for metrics HTTP server to hook into
var _ = (*Metrics)(nil)

func NewMetrics() *Metrics {
	ns := "op_node_default"

	registry := opmetrics.NewRegistry()
	factory := opmetrics.With(registry)

	return &Metrics{
		registry: registry,
		factory:  factory,

		EventMetricsTracker: event.NewMetricsTracker(ns, factory),
		RPCMetrics:          opmetrics.MakeRPCMetrics(ns, factory),

		CacheMetrics:      NewCacheMetrics(ns, factory),
		DBMetrics:         NewDBMetrics(ns, factory),
		DerivationMetrics: NewDerivationMetrics(ns, factory),
		L1Metrics:         NewL1Metrics(ns, factory),
		P2PMetrics:        NewP2PMetrics(ns, factory),
		PayloadsMetrics:   NewPayloadsMetrics(ns, factory),
		SuperRefMetrics:   NewSuperRefMetrics(ns, factory),
		SequencerMetrics:  NewSequencerMetrics(ns, factory),
		ServiceMetrics:    NewServiceMetrics(ns, factory),
		SuperMetrics:      NewSuperMetrics(ns, factory),
	}
}

func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

func (m *Metrics) Document() []opmetrics.DocumentedMetric {
	return m.factory.Document()
}

func chainIDLabel(chainID eth.ChainID) string {
	return chainID.String()
}

func (m *Metrics) RecordReceivedUnsafePayload(chainID eth.ChainID, payload *eth.ExecutionPayloadEnvelope) {
	m.SuperRefMetrics.RecordReceivedUnsafePayloadRef(chainID, payload.ExecutionPayload.BlockRef())
	m.PayloadsMetrics.RecordReceivedUnsafePayload(chainID, payload)
}

func (m *Metrics) RecordUnsafePayloadsBuffer(chainID eth.ChainID, length uint64, memSize uint64, next eth.BlockRef) {
	m.SuperRefMetrics.RecordUnsafePayloadsBufferRef(chainID, next)
	m.PayloadsMetrics.RecordUnsafePayloadsBuffer(chainID, length, memSize, next)
}

type NoopMetrics struct {
	opmetrics.NoopRPCMetrics
	event.NoopMetrics

	NoopCacheMetrics
	NoopDBMetrics
	NoopDerivationMetrics
	NoopL1Metrics
	NoopP2PMetrics
	NoopPayloadsMetrics
	NoopSuperRefMetrics
	NoopSequencerMetrics
	NoopServiceMetrics
	NoopSuperMetrics
}

var _ Metricer = NoopMetrics{}

func (NoopMetrics) Document() []opmetrics.DocumentedMetric { return nil }
