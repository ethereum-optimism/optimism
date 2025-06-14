package metrics

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type SuperRefMetricer interface {
	// from supervisor
	RecordCrossUnsafe(chainID eth.ChainID, seal types.BlockSeal)
	RecordCrossSafe(chainID eth.ChainID, seal types.BlockSeal)
	RecordLocalSafe(chainID eth.ChainID, seal types.BlockSeal)
	RecordLocalUnsafe(chainID eth.ChainID, seal types.BlockSeal)
	RecordFinalized(chainID eth.ChainID, seal types.BlockSeal)

	RecordL1Derived(chainID eth.ChainID, seal types.BlockSeal)

	RecordReceivedUnsafePayloadRef(chainID eth.ChainID, r eth.BlockRef)
	RecordUnsafePayloadsBufferRef(chainID eth.ChainID, next eth.BlockRef)
}

type SuperRefMetrics struct {
	refMetrics opmetrics.RefMetricsWithChainID
}

var _ SuperRefMetricer = (*SuperRefMetrics)(nil)

func NewSuperRefMetrics(ns string, factory opmetrics.Factory) *SuperRefMetrics {
	return &SuperRefMetrics{
		refMetrics: opmetrics.MakeRefMetricsWithChainID(ns, factory),
	}
}

// TODO: some of the metric names here need to be updated to reflect original op-node metrics, or alias them.

func (m *SuperRefMetrics) RecordCrossUnsafe(chainID eth.ChainID, seal types.BlockSeal) {
	m.refMetrics.RecordRef("l2", "cross_unsafe", seal.Number, seal.Timestamp, seal.Hash, chainID)
}

func (m *SuperRefMetrics) RecordCrossSafe(chainID eth.ChainID, seal types.BlockSeal) {
	m.refMetrics.RecordRef("l2", "cross_safe", seal.Number, seal.Timestamp, seal.Hash, chainID)
}

func (m *SuperRefMetrics) RecordLocalUnsafe(chainID eth.ChainID, seal types.BlockSeal) {
	m.refMetrics.RecordRef("l2", "local_unsafe", seal.Number, seal.Timestamp, seal.Hash, chainID)
}

func (m *SuperRefMetrics) RecordLocalSafe(chainID eth.ChainID, seal types.BlockSeal) {
	m.refMetrics.RecordRef("l2", "local_safe", seal.Number, seal.Timestamp, seal.Hash, chainID)
}

func (m *SuperRefMetrics) RecordFinalized(chainID eth.ChainID, seal types.BlockSeal) {
	m.refMetrics.RecordRef("l2", "finalized", seal.Number, seal.Timestamp, seal.Hash, chainID)
}

func (m *SuperRefMetrics) RecordL1Derived(chainID eth.ChainID, seal types.BlockSeal) {
	m.refMetrics.RecordRef("l1", "l1_derived", seal.Number, seal.Timestamp, seal.Hash, chainID)
}

func (m *SuperRefMetrics) RecordReceivedUnsafePayloadRef(chainID eth.ChainID, ref eth.BlockRef) {
	m.refMetrics.RecordRef("l2", "received_payload", ref.Number, ref.Time, ref.Hash, chainID)
}

func (m *SuperRefMetrics) RecordUnsafePayloadsBufferRef(chainID eth.ChainID, next eth.BlockRef) {
	m.refMetrics.RecordRef("l2", "l2_buffer_unsafe", next.Number, next.Time, next.Hash, chainID)
}

type NoopSuperRefMetrics struct{}

var _ SuperRefMetricer = NoopSuperRefMetrics{}

func (NoopSuperRefMetrics) RecordCrossUnsafe(chainID eth.ChainID, seal types.BlockSeal)          {}
func (NoopSuperRefMetrics) RecordCrossSafe(chainID eth.ChainID, seal types.BlockSeal)            {}
func (NoopSuperRefMetrics) RecordLocalSafe(chainID eth.ChainID, seal types.BlockSeal)            {}
func (NoopSuperRefMetrics) RecordLocalUnsafe(chainID eth.ChainID, seal types.BlockSeal)          {}
func (NoopSuperRefMetrics) RecordFinalized(chainID eth.ChainID, seal types.BlockSeal)            {}
func (NoopSuperRefMetrics) RecordL1Derived(chainID eth.ChainID, seal types.BlockSeal)            {}
func (NoopSuperRefMetrics) RecordReceivedUnsafePayloadRef(chainID eth.ChainID, ref eth.BlockRef) {}
func (NoopSuperRefMetrics) RecordUnsafePayloadsBufferRef(chainID eth.ChainID, next eth.BlockRef) {}
