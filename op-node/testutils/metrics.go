package testutils

import (
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

// TestDerivationMetrics implements the metrics used in the derivation pipeline as no-op operations.
// Optionally a test may hook into the metrics
type TestDerivationMetrics struct {
	FnRecordL1ReorgDepth      func(d uint64)
	FnRecordL1Ref             func(name string, ref eth.L1BlockRef)
	FnRecordL2Ref             func(name string, ref eth.L2BlockRef)
	FnRecordUnsafePayloads    func(length uint64, memSize uint64, next eth.BlockID)
	FnRecordChannelInputBytes func(inputCompresedBytes int)
	FnRecordL1Block           func(info eth.BlockInfo, txs types.Transactions)
	FnRecordL2Block           func(payload *eth.ExecutionPayload)
}

func (t *TestDerivationMetrics) RecordL1ReorgDepth(d uint64) {
	if t.FnRecordL1ReorgDepth != nil {
		t.FnRecordL1ReorgDepth(d)
	}
}

func (t *TestDerivationMetrics) RecordL1Ref(name string, ref eth.L1BlockRef) {
	if t.FnRecordL1Ref != nil {
		t.FnRecordL1Ref(name, ref)
	}
}

func (t *TestDerivationMetrics) RecordL2Ref(name string, ref eth.L2BlockRef) {
	if t.FnRecordL2Ref != nil {
		t.FnRecordL2Ref(name, ref)
	}
}

func (t *TestDerivationMetrics) RecordUnsafePayloadsBuffer(length uint64, memSize uint64, next eth.BlockID) {
	if t.FnRecordUnsafePayloads != nil {
		t.FnRecordUnsafePayloads(length, memSize, next)
	}
}

func (t *TestDerivationMetrics) RecordChannelInputBytes(inputCompresedBytes int) {
	if t.FnRecordChannelInputBytes != nil {
		t.FnRecordChannelInputBytes(inputCompresedBytes)
	}
}
func (t *TestDerivationMetrics) RecordL1Block(info eth.BlockInfo, txs types.Transactions) {
	if t.FnRecordL1Block != nil {
		t.FnRecordL1Block(info, txs)
	}
}

func (t *TestDerivationMetrics) RecordL2Block(payload *eth.ExecutionPayload) {
	if t.FnRecordL2Block != nil {
		t.FnRecordL2Block(payload)
	}
}

type TestRPCMetrics struct{}

func (n *TestRPCMetrics) RecordRPCServerRequest(method string) func() {
	return func() {}
}
