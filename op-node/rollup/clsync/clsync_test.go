package clsync

import (
	"context"
	"math/big"
	mrand "math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
)

type noopMetrics struct{}

func (n *noopMetrics) RecordUnsafePayloadsBuffer(length uint64, memSize uint64, next eth.BlockID) {}

type fakeEngController struct{ calls int }

func (f *fakeEngController) RequestForkchoiceUpdate(ctx context.Context) { f.calls++ }
func (f *fakeEngController) TryUpdatePendingSafe(ctx context.Context, ref eth.L2BlockRef, concluding bool, source eth.L1BlockRef) {
}
func (f *fakeEngController) TryUpdateLocalSafe(ctx context.Context, ref eth.L2BlockRef, concluding bool, source eth.L1BlockRef) {
}
func (f *fakeEngController) RequestPendingSafeUpdate(ctx context.Context) {
}

func TestCLSync_InvalidPayloadDropsHead(t *testing.T) {
	logger := testlog.Logger(t, 0)
	fe := &fakeEngController{}
	cl := NewCLSync(logger, nil, &noopMetrics{}, fe)
	emitter := &testutils.MockEmitter{}
	cl.AttachEmitter(emitter)

	payload := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		BlockHash: common.Hash{0x01},
	}}

	// Adding an unsafe payload requests a forkchoice update via engine controller
	cl.OnEvent(context.Background(), ReceivedUnsafePayloadEvent{Envelope: payload})
	require.Equal(t, 1, fe.calls)
	require.NotNil(t, cl.unsafePayloads.Peek())

	// Mark it invalid; it should be dropped if it matches the queue head
	cl.OnEvent(context.Background(), engine.PayloadInvalidEvent{Envelope: payload})
	require.Nil(t, cl.unsafePayloads.Peek())
}

type recordingMetrics struct {
	calls    int
	lastLen  uint64
	lastMem  uint64
	lastNext eth.BlockID
}

func (m *recordingMetrics) RecordUnsafePayloadsBuffer(length uint64, memSize uint64, next eth.BlockID) {
	m.calls++
	m.lastLen = length
	m.lastMem = memSize
	m.lastNext = next
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
	a1L1Info, err := derive.L1InfoDepositBytes(cfg, cfg.Genesis.SystemConfig, refA1.SequenceNumber, aL1Info, refA1.Time)
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

func TestCLSync_OnUnsafePayload_EnqueueEmitAndRecord(t *testing.T) {
	cfg, _, refA1, payloadA1 := buildSimpleCfgAndPayload(t)
	logger := testlog.Logger(t, 0)
	metrics := &recordingMetrics{}
	emitter := &testutils.MockEmitter{}
	fe := &fakeEngController{}
	cl := NewCLSync(logger, cfg, metrics, fe)
	cl.AttachEmitter(emitter)

	cl.OnEvent(context.Background(), ReceivedUnsafePayloadEvent{Envelope: payloadA1})
	require.Equal(t, 1, fe.calls)

	// queued and metrics recorded
	got := cl.unsafePayloads.Peek()
	require.NotNil(t, got)
	require.Equal(t, payloadA1, got)
	require.Equal(t, 1, metrics.calls)
	require.EqualValues(t, 1, metrics.lastLen)
	require.Equal(t, refA1.Hash, metrics.lastNext.Hash)
	require.Equal(t, cl.unsafePayloads.MemSize(), metrics.lastMem)
}

func TestCLSync_OnForkchoiceUpdate_ProcessRetryAndPop(t *testing.T) {
	cfg, refA0, refA1, payloadA1 := buildSimpleCfgAndPayload(t)
	logger := testlog.Logger(t, 0)
	metrics := &recordingMetrics{}
	emitter := &testutils.MockEmitter{}
	fe := &fakeEngController{}
	cl := NewCLSync(logger, cfg, metrics, fe)
	cl.AttachEmitter(emitter)

	// queue payload A1
	cl.OnEvent(context.Background(), ReceivedUnsafePayloadEvent{Envelope: payloadA1})
	require.Equal(t, 1, fe.calls)

	// applicable forkchoice -> process once
	emitter.ExpectOnce(engine.ProcessUnsafePayloadEvent{Envelope: payloadA1})
	cl.OnEvent(context.Background(), engine.ForkchoiceUpdateEvent{UnsafeL2Head: refA0, SafeL2Head: refA0, FinalizedL2Head: refA0})
	emitter.AssertExpectations(t)
	require.NotNil(t, cl.unsafePayloads.Peek(), "should not pop yet")

	// same forkchoice -> retry
	emitter.ExpectOnce(engine.ProcessUnsafePayloadEvent{Envelope: payloadA1})
	cl.OnEvent(context.Background(), engine.ForkchoiceUpdateEvent{UnsafeL2Head: refA0, SafeL2Head: refA0, FinalizedL2Head: refA0})
	emitter.AssertExpectations(t)
	require.NotNil(t, cl.unsafePayloads.Peek(), "still pending")

	// after applied (unsafe head == A1) -> pop
	cl.OnEvent(context.Background(), engine.ForkchoiceUpdateEvent{UnsafeL2Head: refA1, SafeL2Head: refA0, FinalizedL2Head: refA0})
	require.Nil(t, cl.unsafePayloads.Peek())
}

func TestCLSync_LowestQueuedUnsafeBlock(t *testing.T) {
	cfg, _, _, payloadA1 := buildSimpleCfgAndPayload(t)
	logger := testlog.Logger(t, 0)
	cl := NewCLSync(logger, cfg, &noopMetrics{}, &fakeEngController{})
	// empty -> zero
	require.Equal(t, eth.L2BlockRef{}, cl.LowestQueuedUnsafeBlock())

	// queue -> returns derived ref
	_ = cl.unsafePayloads.Push(payloadA1)
	want, err := derive.PayloadToBlockRef(cfg, payloadA1.ExecutionPayload)
	require.NoError(t, err)
	require.Equal(t, want, cl.LowestQueuedUnsafeBlock())
}

func TestCLSync_LowestQueuedUnsafeBlock_OnDeriveErrorReturnsZero(t *testing.T) {
	// missing L1-info in txs will cause derive error
	logger := testlog.Logger(t, 0)
	cl := NewCLSync(logger, &rollup.Config{}, &noopMetrics{}, &fakeEngController{})
	bad := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{BlockNumber: 1, BlockHash: common.Hash{0xaa}}}
	_ = cl.unsafePayloads.Push(bad)
	require.Equal(t, eth.L2BlockRef{}, cl.LowestQueuedUnsafeBlock())
}

func TestCLSync_InvalidPayloadForNonHead_NoDrop(t *testing.T) {
	logger := testlog.Logger(t, 0)
	fe := &fakeEngController{}
	cl := NewCLSync(logger, nil, &noopMetrics{}, fe)
	emitter := &testutils.MockEmitter{}
	cl.AttachEmitter(emitter)

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

	cl.OnEvent(context.Background(), ReceivedUnsafePayloadEvent{Envelope: head})
	cl.OnEvent(context.Background(), ReceivedUnsafePayloadEvent{Envelope: other})
	require.Equal(t, 2, fe.calls)

	// Invalidate non-head should not drop head
	cl.OnEvent(context.Background(), engine.PayloadInvalidEvent{Envelope: other})
	require.Equal(t, 2, cl.unsafePayloads.Len())
	require.Equal(t, head, cl.unsafePayloads.Peek())
}

// note: nil-envelope behavior is not tested to match current implementation
