package derive

import (
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
)

var _ Engine = (*testutils.MockEngine)(nil)

var _ L1Fetcher = (*testutils.MockL1Source)(nil)

// TestMetrics implements the metrics used in the derivation pipeline as no-op operations.
// Optionally a test may hook into the metrics
type TestMetrics struct {
	recordL1Ref          func(name string, ref eth.L1BlockRef)
	recordL2Ref          func(name string, ref eth.L2BlockRef)
	recordUnsafePayloads func(length uint64, memSize uint64, next eth.BlockID)
}

func (t *TestMetrics) RecordL1Ref(name string, ref eth.L1BlockRef) {
	if t.recordL1Ref != nil {
		t.recordL1Ref(name, ref)
	}
}

func (t *TestMetrics) RecordL2Ref(name string, ref eth.L2BlockRef) {
	if t.recordL2Ref != nil {
		t.recordL2Ref(name, ref)
	}
}

func (t *TestMetrics) RecordUnsafePayloadsBuffer(length uint64, memSize uint64, next eth.BlockID) {
	if t.recordUnsafePayloads != nil {
		t.recordUnsafePayloads(length, memSize, next)
	}
}

var _ Metrics = (*TestMetrics)(nil)
