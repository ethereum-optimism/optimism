package actions

import (
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

// TODO: deduplicate with derive.TestMetrics by moving it into the test utils package

type TestMetrics struct{}

func (t TestMetrics) RecordUnsafePayloadsBuffer(length uint64, memSize uint64, next eth.BlockID) {}
func (t TestMetrics) RecordL1Ref(name string, ref eth.L1BlockRef)                                {}
func (t TestMetrics) RecordL2Ref(name string, ref eth.L2BlockRef)                                {}

var _ derive.Metrics = (*TestMetrics)(nil)
