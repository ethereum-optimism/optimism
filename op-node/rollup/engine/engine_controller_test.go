package engine

import (
	"context"
	"math/big"
	mrand "math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func TestInvalidPayloadDropsHead(t *testing.T) {
	emitter := &testutils.MockEmitter{}
	ec := NewEngineController(context.Background(), nil, testlog.Logger(t, 0), metrics.NoopMetrics, &rollup.Config{}, &sync.Config{}, &testutils.MockL1Source{}, emitter)

	payload := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		BlockHash: common.Hash{0x01},
	}}

	emitter.ExpectOnce(PayloadInvalidEvent{})
	emitter.ExpectOnce(ForkchoiceUpdateEvent{})

	// Add an unsafe payload requests a forkchoice update via engine controller
	ec.AddUnsafePayload(context.Background(), payload)

	require.NotNil(t, ec.unsafePayloads.Peek())

	// Mark it invalid; it should be dropped if it matches the queue head
	ec.OnEvent(context.Background(), PayloadInvalidEvent{Envelope: payload})
	require.Nil(t, ec.unsafePayloads.Peek())
}

// buildSimpleCfgAndPayload creates a minimal rollup config and a valid payload (A1) on top of A0.
func buildSimpleCfgAndPayload(t *testing.T) (*rollup.Config, eth.L2BlockRef, eth.L2BlockRef, *eth.ExecutionPayloadEnvelope) {
	t.Helper()
	rng := mrand.New(mrand.NewSource(1234))
	refA := testutils.RandomBlockRef(rng)

	refA0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         0,
		ParentHash:     common.Hash{},
		Time:           refA.Time,
		L1Origin:       refA.ID(),
		SequenceNumber: 0,
	}

	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L1:     refA.ID(),
			L2:     refA0.ID(),
			L2Time: refA0.Time,
			SystemConfig: eth.SystemConfig{
				BatcherAddr: common.Address{42},
				Overhead:    [32]byte{123},
				Scalar:      [32]byte{42},
				GasLimit:    20_000_000,
			},
		},
		BlockTime:     1,
		SeqWindowSize: 2,
	}

	refA1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refA0.Number + 1,
		ParentHash:     refA0.Hash,
		Time:           refA0.Time + cfg.BlockTime,
		L1Origin:       refA.ID(),
		SequenceNumber: 1,
	}

	// Populate necessary L1 info fields
	aL1Info := &testutils.MockBlockInfo{
		InfoParentHash:  refA.ParentHash,
		InfoNum:         refA.Number,
		InfoTime:        refA.Time,
		InfoHash:        refA.Hash,
		InfoBaseFee:     big.NewInt(1),
		InfoBlobBaseFee: big.NewInt(1),
		InfoReceiptRoot: gethtypes.EmptyRootHash,
		InfoRoot:        testutils.RandomHash(rng),
		InfoGasUsed:     rng.Uint64(),
	}
	a1L1Info, err := derive.L1InfoDepositBytes(cfg, params.SepoliaChainConfig, cfg.Genesis.SystemConfig, refA1.SequenceNumber, aL1Info, refA1.Time)
	require.NoError(t, err)

	payloadA1 := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		ParentHash:   refA1.ParentHash,
		BlockNumber:  eth.Uint64Quantity(refA1.Number),
		Timestamp:    eth.Uint64Quantity(refA1.Time),
		BlockHash:    refA1.Hash,
		Transactions: []eth.Data{a1L1Info},
	}}
	return cfg, refA0, refA1, payloadA1
}

func TestOnUnsafePayload_EnqueueEmit(t *testing.T) {
	cfg, _, _, payloadA1 := buildSimpleCfgAndPayload(t)

	emitter := &testutils.MockEmitter{}
	ec := NewEngineController(context.Background(), nil, testlog.Logger(t, 0), metrics.NoopMetrics, cfg, &sync.Config{}, &testutils.MockL1Source{}, emitter)

	emitter.ExpectOnce(PayloadInvalidEvent{})
	emitter.ExpectOnce(ForkchoiceUpdateEvent{})

	ec.AddUnsafePayload(context.Background(), payloadA1)

	got := ec.unsafePayloads.Peek()
	require.NotNil(t, got)
	require.Equal(t, payloadA1, got)
}

func TestOnForkchoiceUpdate_ProcessRetryAndPop(t *testing.T) {
	cfg, refA0, refA1, payloadA1 := buildSimpleCfgAndPayload(t)

	emitter := &testutils.MockEmitter{}
	mockEngine := &testutils.MockEngine{}
	cl := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0), metrics.NoopMetrics, cfg, &sync.Config{SyncMode: sync.CLSync}, &testutils.MockL1Source{}, emitter)

	// queue payload A1
	emitter.ExpectOnceType("UnsafeUpdateEvent")
	emitter.ExpectOnceType("PayloadInvalidEvent")
	emitter.ExpectOnceType("ForkchoiceUpdateEvent")
	emitter.ExpectOnceType("ForkchoiceUpdateEvent")
	cl.AddUnsafePayload(context.Background(), payloadA1)

	// applicable forkchoice -> process once
	mockEngine.ExpectGetPayload(eth.PayloadID{}, payloadA1, nil)
	mockEngine.ExpectNewPayload(payloadA1.ExecutionPayload, nil, &eth.PayloadStatusV1{Status: eth.ExecutionValid}, nil)
	mockEngine.ExpectForkchoiceUpdate(&eth.ForkchoiceState{HeadBlockHash: refA1.Hash, SafeBlockHash: common.Hash{}, FinalizedBlockHash: common.Hash{}}, nil, &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}}, nil)
	cl.OnEvent(context.Background(), ForkchoiceUpdateEvent{UnsafeL2Head: refA0, SafeL2Head: refA0, FinalizedL2Head: refA0})
	require.NotNil(t, cl.unsafePayloads.Peek(), "should not pop yet")

	// same forkchoice -> retry
	cl.OnEvent(context.Background(), ForkchoiceUpdateEvent{UnsafeL2Head: refA0, SafeL2Head: refA0, FinalizedL2Head: refA0})
	require.NotNil(t, cl.unsafePayloads.Peek(), "still pending")

	// after applied (unsafe head == A1) -> pop
	cl.OnEvent(context.Background(), ForkchoiceUpdateEvent{UnsafeL2Head: refA1, SafeL2Head: refA0, FinalizedL2Head: refA0})
	require.Nil(t, cl.unsafePayloads.Peek())
}

func TestLowestQueuedUnsafeBlock(t *testing.T) {
	cfg, _, _, payloadA1 := buildSimpleCfgAndPayload(t)

	emitter := &testutils.MockEmitter{}
	ec := NewEngineController(context.Background(), nil, testlog.Logger(t, 0), metrics.NoopMetrics, cfg, &sync.Config{SyncMode: sync.CLSync}, &testutils.MockL1Source{}, emitter)

	// empty -> zero
	require.Equal(t, eth.L2BlockRef{}, ec.LowestQueuedUnsafeBlock())

	// queue -> returns derived ref
	_ = ec.unsafePayloads.Push(payloadA1)
	want, err := derive.PayloadToBlockRef(cfg, payloadA1.ExecutionPayload)
	require.NoError(t, err)
	require.Equal(t, want, ec.LowestQueuedUnsafeBlock())
}

func TestLowestQueuedUnsafeBlock_OnDeriveErrorReturnsZero(t *testing.T) {
	// missing L1-info in txs will cause derive error
	emitter := &testutils.MockEmitter{}
	ec := NewEngineController(context.Background(), nil, testlog.Logger(t, 0), metrics.NoopMetrics, &rollup.Config{}, &sync.Config{SyncMode: sync.CLSync}, &testutils.MockL1Source{}, emitter)

	bad := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{BlockNumber: 1, BlockHash: common.Hash{0xaa}}}
	_ = ec.unsafePayloads.Push(bad)
	require.Equal(t, eth.L2BlockRef{}, ec.LowestQueuedUnsafeBlock())
}

func TestInvalidPayloadForNonHead_NoDrop(t *testing.T) {
	emitter := &testutils.MockEmitter{}
	ec := NewEngineController(context.Background(), nil, testlog.Logger(t, 0), metrics.NoopMetrics, &rollup.Config{}, &sync.Config{SyncMode: sync.CLSync}, &testutils.MockL1Source{}, emitter)

	// Head payload (lower block number)
	head := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		BlockNumber: 1,
		BlockHash:   common.Hash{0x01},
	}}
	// Non-head payload (higher block number)
	other := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		BlockNumber: 2,
		BlockHash:   common.Hash{0x02},
	}}

	emitter.ExpectOnce(PayloadInvalidEvent{})
	emitter.ExpectOnce(ForkchoiceUpdateEvent{})
	ec.AddUnsafePayload(context.Background(), head)

	emitter.ExpectOnce(PayloadInvalidEvent{})
	emitter.ExpectOnce(ForkchoiceUpdateEvent{})
	ec.AddUnsafePayload(context.Background(), other)

	// Invalidate non-head should not drop head
	ec.OnEvent(context.Background(), PayloadInvalidEvent{Envelope: other})
	require.Equal(t, 2, ec.unsafePayloads.Len())
	require.Equal(t, head, ec.unsafePayloads.Peek())
}

// note: nil-envelope behavior is not tested to match current implementation
