package engine

import (
	"context"
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
)

// Tests for invalidation-recovery gating in insertUnsafePayload: while any
// denied height is above the finalized head, all unsafe payloads are dropped
// and the chain advances via derivation only.

func newUnsafeIngestionEC(t *testing.T, cfg *rollup.Config, engine *testutils.MockEngine, sa rollup.SuperAuthority, emitter *testutils.MockEmitter, unsafeHead eth.L2BlockRef) *EngineController {
	t.Helper()
	ec := NewEngineController(
		context.Background(),
		engine,
		testlog.Logger(t, 0),
		metrics.NoopMetrics,
		cfg,
		&sync.Config{SyncMode: sync.CLSync},
		&testutils.MockL1Source{},
		emitter,
		sa,
	)
	ec.SetUnsafeHead(unsafeHead)
	ec.SetLocalSafeHead(unsafeHead)
	ec.SetFinalizedHead(unsafeHead)
	return ec
}

// expectGapFillSyncing sets up the stock gap-fill interaction: NewPayload then
// an FCU that kicks the EL into sync, both answering SYNCING.
func expectGapFillSyncing(engine *testutils.MockEngine, payload *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef, safeHash, finalizedHash common.Hash) {
	engine.ExpectNewPayload(payload.ExecutionPayload, nil, &eth.PayloadStatusV1{Status: eth.ExecutionSyncing}, nil)
	engine.ExpectForkchoiceUpdate(&eth.ForkchoiceState{
		HeadBlockHash:      ref.Hash,
		SafeBlockHash:      safeHash,
		FinalizedBlockHash: finalizedHash,
	}, nil, &eth.ForkchoiceUpdatedResult{
		PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionSyncing},
	}, nil)
}

func gapPayload(refA0 eth.L2BlockRef, cfg *rollup.Config) (*eth.ExecutionPayloadEnvelope, eth.L2BlockRef) {
	ref5 := eth.L2BlockRef{
		Hash:       common.Hash{0x05},
		Number:     refA0.Number + 5,
		ParentHash: common.Hash{0x04},
		Time:       refA0.Time + 5*cfg.BlockTime,
	}
	payload5 := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		ParentHash:  ref5.ParentHash,
		BlockNumber: eth.Uint64Quantity(ref5.Number),
		Timestamp:   eth.Uint64Quantity(ref5.Time),
		BlockHash:   ref5.Hash,
	}}
	return payload5, ref5
}

func TestInsertUnsafePayloadDroppedDuringRecovery(t *testing.T) {
	cfg, refA0, refA1, payloadA1 := buildSimpleCfgAndPayload(t)
	engine := &testutils.MockEngine{}
	emitter := &testutils.MockEmitter{}
	sa := newMockSuperAuthority()
	// A denied height above finalized (0) puts the controller in recovery.
	sa.denyBlock(2, common.Hash{0xba, 0xd0})

	ec := newUnsafeIngestionEC(t, cfg, engine, sa, emitter, refA0)

	// Every unsafe payload is dropped without engine interaction — even a
	// contiguous extension of the current head.
	require.NoError(t, ec.InsertUnsafePayload(context.Background(), payloadA1, refA1))
	require.Equal(t, refA0, ec.UnsafeL2Head(), "unsafe head must not advance during recovery")

	// A far-ahead gap payload (the EL-sync trigger) is dropped too.
	payload5, ref5 := gapPayload(refA0, cfg)
	require.NoError(t, ec.InsertUnsafePayload(context.Background(), payload5, ref5))
	require.Equal(t, refA0, ec.UnsafeL2Head(), "unsafe head must not advance during recovery")

	engine.AssertExpectations(t)
	emitter.AssertExpectations(t)
}

func TestAddUnsafePayloadNotQueuedDuringRecovery(t *testing.T) {
	cfg, refA0, _, payloadA1 := buildSimpleCfgAndPayload(t)
	emitter := &testutils.MockEmitter{}
	sa := newMockSuperAuthority()
	sa.denyBlock(2, common.Hash{0xba, 0xd0})

	ec := newUnsafeIngestionEC(t, cfg, &testutils.MockEngine{}, sa, emitter, refA0)

	// A gated payload must not even enter the queue — a queued denied-branch
	// payload would be gap-filled into the engine once the window closes.
	ec.AddUnsafePayload(context.Background(), payloadA1)
	payload, _ := ec.PeekUnsafePayload()
	require.Nil(t, payload, "payload must not be queued during recovery")

	emitter.AssertExpectations(t)
}

func TestInsertUnsafePayloadProceedsWithoutDenials(t *testing.T) {
	cfg, refA0, _, _ := buildSimpleCfgAndPayload(t)
	engine := &testutils.MockEngine{}
	emitter := &testutils.MockEmitter{}
	sa := newMockSuperAuthority()

	ec := newUnsafeIngestionEC(t, cfg, engine, sa, emitter, refA0)

	// With an empty deny list, the normal gap-fill path runs. SYNCING on the
	// FCU stays a temporary error in CLSync mode.
	payload5, ref5 := gapPayload(refA0, cfg)
	expectGapFillSyncing(engine, payload5, ref5, refA0.Hash, refA0.Hash)

	err := ec.InsertUnsafePayload(context.Background(), payload5, ref5)
	require.ErrorIs(t, err, derive.ErrTemporary)

	engine.AssertExpectations(t)
	emitter.AssertExpectations(t)
}

func TestInsertUnsafePayloadGatingClearsAfterFinality(t *testing.T) {
	cfg, refA0, _, _ := buildSimpleCfgAndPayload(t)
	engine := &testutils.MockEngine{}
	emitter := &testutils.MockEmitter{}
	sa := newMockSuperAuthority()
	// The only denied height is at or below finalized: the denied branch is
	// dead by construction, the recovery window is over, and unsafe sync must
	// behave stock again.
	sa.denyBlock(2, common.Hash{0xba, 0xd0})

	ec := newUnsafeIngestionEC(t, cfg, engine, sa, emitter, refA0)
	finalized := eth.L2BlockRef{
		Hash:   common.Hash{0x02},
		Number: 2,
		Time:   refA0.Time + 2*cfg.BlockTime,
	}
	ec.SetFinalizedHead(finalized)

	payload5, ref5 := gapPayload(refA0, cfg)
	expectGapFillSyncing(engine, payload5, ref5, refA0.Hash, finalized.Hash)

	err := ec.InsertUnsafePayload(context.Background(), payload5, ref5)
	require.ErrorIs(t, err, derive.ErrTemporary)

	engine.AssertExpectations(t)
	emitter.AssertExpectations(t)
}

func TestInsertUnsafePayloadNilAuthorityUnaffected(t *testing.T) {
	cfg, refA0, refA1, payloadA1 := buildSimpleCfgAndPayload(t)
	engine := &testutils.MockEngine{}
	emitter := &testutils.MockEmitter{}

	// Plain op-nodes (no SuperAuthority) keep stock behavior.
	ec := newUnsafeIngestionEC(t, cfg, engine, nil, emitter, refA0)

	engine.ExpectNewPayload(payloadA1.ExecutionPayload, nil, &eth.PayloadStatusV1{Status: eth.ExecutionValid}, nil)
	engine.ExpectForkchoiceUpdate(&eth.ForkchoiceState{
		HeadBlockHash:      refA1.Hash,
		SafeBlockHash:      refA0.Hash,
		FinalizedBlockHash: refA0.Hash,
	}, nil, &eth.ForkchoiceUpdatedResult{
		PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid},
	}, nil)
	emitter.ExpectOnceType("UnsafeUpdateEvent")
	emitter.ExpectOnceType("ForkchoiceUpdateEvent")

	require.NoError(t, ec.InsertUnsafePayload(context.Background(), payloadA1, refA1))
	require.Equal(t, refA1, ec.UnsafeL2Head())

	engine.AssertExpectations(t)
	emitter.AssertExpectations(t)
}
