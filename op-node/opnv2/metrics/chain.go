package metrics

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type ChainLegacyRefMetrics interface {
	RecordL1Ref(name string, ref eth.L1BlockRef)
}

type ChainRefMetricer interface {
	RecordCrossUnsafe(seal types.BlockSeal)
	RecordCrossSafe(seal types.BlockSeal)
	RecordLocalSafe(seal types.BlockSeal)
	RecordLocalUnsafe(seal types.BlockSeal)
	RecordFinalized(seal types.BlockSeal)
}

type ChainDBMetrics interface {
	RecordDBEntryCount(kind string, count int64)
	RecordDBSearchEntriesRead(count int64)
}

type ChainSequencingMetrics interface {
	CountSequencedTxsInBlock(txns int, deposits int)
	RecordSequencerBuildingDiffTime(duration time.Duration)
	RecordSequencerSealingTime(duration time.Duration)
}

// TODO: below derivation metrics are not yet attributed with chainID
// (or unique derivation pipeline, if we run multiple per chain)

type ChainDerivationMetrics interface {
	SetDerivationIdle(status bool)
	RecordPipelineReset()
	RecordDerivationError()
	RecordDerivedBatches(batchType string)
	RecordChannelInputBytes(num int)
	RecordHeadChannelOpened()
	RecordChannelTimedOut()
	RecordFrame()
}

type ChainMetricer interface {
	ChainRefMetricer
	caching.Metrics
	ChainDBMetrics
	ChainLegacyRefMetrics
	ChainDerivationMetrics
}

// ChainMetrics is an adapter between the metrics API expected by clients that assume there's only a single chain
// and the actual metrics implementation which requires a chain ID to identify the source chain.
type ChainMetrics struct {
	chainID  eth.ChainID
	delegate Metricer

	// TODO: it would be nice if we compose the chain-delegated metrics
}

var _ ChainMetricer = (*ChainMetrics)(nil)
var _ caching.Metrics = (*ChainMetrics)(nil)
var _ logs.Metrics = (*ChainMetrics)(nil)

func NewChainMetrics(chainID eth.ChainID, delegate Metricer) *ChainMetrics {
	return &ChainMetrics{
		chainID:  chainID,
		delegate: delegate,
	}
}

func (c *ChainMetrics) RecordCrossUnsafe(seal types.BlockSeal) {
	c.delegate.RecordCrossUnsafe(c.chainID, seal)
}

func (c *ChainMetrics) RecordCrossSafe(seal types.BlockSeal) {
	c.delegate.RecordCrossSafe(c.chainID, seal)
}

func (c *ChainMetrics) RecordLocalSafe(seal types.BlockSeal) {
	c.delegate.RecordLocalSafe(c.chainID, seal)
}

func (c *ChainMetrics) RecordLocalUnsafe(seal types.BlockSeal) {
	c.delegate.RecordLocalUnsafe(c.chainID, seal)
}

func (c *ChainMetrics) RecordFinalized(seal types.BlockSeal) {
	c.delegate.RecordFinalized(c.chainID, seal)
}

func (c *ChainMetrics) CacheAdd(label string, cacheSize int, evicted bool) {
	c.delegate.CacheAdd(c.chainID, label, cacheSize, evicted)
}

func (c *ChainMetrics) CacheGet(label string, hit bool) {
	c.delegate.CacheGet(c.chainID, label, hit)
}

func (c *ChainMetrics) RecordDBEntryCount(kind string, count int64) {
	c.delegate.RecordDBEntryCount(c.chainID, kind, count)
}

func (c *ChainMetrics) RecordDBSearchEntriesRead(count int64) {
	c.delegate.RecordDBSearchEntriesRead(c.chainID, count)
}

func (c *ChainMetrics) CountSequencedTxsInBlock(txns int, deposits int) {
	c.delegate.CountSequencedTxsInBlock(c.chainID, txns, deposits)
}

func (c *ChainMetrics) RecordSequencerBuildingDiffTime(duration time.Duration) {
	c.delegate.RecordSequencerBuildingDiffTime(c.chainID, duration)
}

func (c *ChainMetrics) RecordSequencerSealingTime(duration time.Duration) {
	c.delegate.RecordSequencerSealingTime(c.chainID, duration)
}

func (c *ChainMetrics) RecordL1Ref(name string, ref eth.L1BlockRef) {
	if name == "l1_derived" { // old derivation pipeline uses this
		c.delegate.RecordL1Derived(c.chainID, types.BlockSealFromRef(ref))
	}
}

func (c *ChainMetrics) SetDerivationIdle(status bool) {
	c.delegate.SetDerivationIdle(status)
}

func (c *ChainMetrics) RecordPipelineReset() {
	c.delegate.RecordPipelineReset()
}

func (c *ChainMetrics) RecordDerivationError() {
	c.delegate.RecordDerivationError()
}

func (c *ChainMetrics) RecordDerivedBatches(batchType string) {
	c.delegate.RecordDerivedBatches(batchType)
}

func (c *ChainMetrics) RecordChannelInputBytes(num int) {
	c.delegate.RecordChannelInputBytes(num)
}

func (c *ChainMetrics) RecordHeadChannelOpened() {
	c.delegate.RecordHeadChannelOpened()
}

func (c *ChainMetrics) RecordChannelTimedOut() {
	c.delegate.RecordChannelTimedOut()
}

func (c *ChainMetrics) RecordFrame() {
	c.delegate.RecordFrame()
}
