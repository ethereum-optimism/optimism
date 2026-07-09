// Tests for the safe/finalized head decision at EL-sync completion (insertUnsafePayload):
// offset retraction, and preserving / trimming / clearing the safedb.
package engine

import (
	"context"
	"math/big"
	mrand "math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type discardEmitter struct{}

func (discardEmitter) Emit(context.Context, event.Event) {}

type noopSyncDeriver struct{}

func (noopSyncDeriver) ResetSafeDB(eth.L2BlockRef) {}

func (noopSyncDeriver) SafeDBTip(context.Context) (eth.BlockID, bool, error) {
	return eth.BlockID{}, false, nil
}

func (noopSyncDeriver) SafeDBHeadAtOrAboveL2(context.Context, uint64) (eth.BlockID, bool, error) {
	return eth.BlockID{}, false, nil
}

// fakeSyncDeriver mimics the production SyncDeriver: SafeDBTip reports the recorded tip,
// SafeDBHeadAtOrAboveL2 reports the recorded safe head near the offset head (checked for
// canonicity), and ResetSafeDB records the most recent reset target (and applies it to db if set).
type fakeSyncDeriver struct {
	db        *safedb.SafeDB
	tip       eth.BlockID
	hasTip    bool
	anchor    eth.BlockID
	hasAnchor bool
	resetTo   *eth.L2BlockRef
}

func (w *fakeSyncDeriver) ResetSafeDB(ref eth.L2BlockRef) {
	w.resetTo = &ref
	if w.db != nil {
		_ = w.db.SafeHeadReset(ref)
	}
}

func (w *fakeSyncDeriver) SafeDBTip(context.Context) (eth.BlockID, bool, error) {
	return w.tip, w.hasTip, nil
}

func (w *fakeSyncDeriver) SafeDBHeadAtOrAboveL2(context.Context, uint64) (eth.BlockID, bool, error) {
	return w.anchor, w.hasAnchor, nil
}

// buildELSyncTipChain returns genesis L2 b0 .. b3 (tip) with BlockTime=2 and a valid payload for b3.
func buildELSyncTipChain(t *testing.T) (*rollup.Config, eth.L2BlockRef, eth.L2BlockRef, eth.L2BlockRef, eth.L2BlockRef, *eth.ExecutionPayloadEnvelope) {
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
		BlockTime:     2,
		SeqWindowSize: 2,
	}
	refA1 := eth.L2BlockRef{
		Hash:           common.Hash{'1'},
		Number:         1,
		ParentHash:     refA0.Hash,
		Time:           refA0.Time + cfg.BlockTime,
		L1Origin:       refA.ID(),
		SequenceNumber: 1,
	}
	refA2 := eth.L2BlockRef{
		Hash:           common.Hash{'2'},
		Number:         2,
		ParentHash:     refA1.Hash,
		Time:           refA1.Time + cfg.BlockTime,
		L1Origin:       refA.ID(),
		SequenceNumber: 2,
	}
	refA3 := eth.L2BlockRef{
		Hash:           common.Hash{'3'},
		Number:         3,
		ParentHash:     refA2.Hash,
		Time:           refA2.Time + cfg.BlockTime,
		L1Origin:       refA.ID(),
		SequenceNumber: 3,
	}
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
	l1Bytes, err := derive.L1InfoDepositBytes(cfg, params.SepoliaChainConfig, cfg.Genesis.SystemConfig, refA3.SequenceNumber, aL1Info, refA3.Time)
	require.NoError(t, err)
	payload := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		ParentHash:   refA3.ParentHash,
		BlockNumber:  eth.Uint64Quantity(refA3.Number),
		Timestamp:    eth.Uint64Quantity(refA3.Time),
		BlockHash:    refA3.Hash,
		Transactions: []eth.Data{l1Bytes},
	}}
	return cfg, refA0, refA1, refA2, refA3, payload
}

func TestInsertUnsafePayload_ELSync_offsetDerived(t *testing.T) {
	cfg, refA0, _, _, refA3, payload := buildELSyncTipChain(t)

	tests := []struct {
		name   string
		offset time.Duration
		want   eth.L2BlockRef // local safe + finalized
		stub   func(eng *testutils.MockEngine)
	}{
		{
			name:   "zero sets safe and finalized to tip",
			offset: 0,
			want:   refA3,
			stub: func(eng *testutils.MockEngine) {
				eng.ExpectL2BlockRefByLabel(eth.Finalized, refA0, nil)
				eng.ExpectNewPayload(payload.ExecutionPayload, nil, &eth.PayloadStatusV1{Status: eth.ExecutionValid}, nil)
				eng.ExpectForkchoiceUpdate(&eth.ForkchoiceState{
					HeadBlockHash:      refA3.Hash,
					SafeBlockHash:      refA3.Hash,
					FinalizedBlockHash: refA3.Hash,
				}, nil, &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}}, nil)
			},
		},
		{
			name:   "non-zero retracts safe and finalized by ceil(offset/BlockTime)",
			offset: 5 * time.Second,
			want:   refA0,
			stub: func(eng *testutils.MockEngine) {
				eng.ExpectL2BlockRefByLabel(eth.Finalized, refA0, nil)
				eng.ExpectNewPayload(payload.ExecutionPayload, nil, &eth.PayloadStatusV1{Status: eth.ExecutionValid}, nil)
				eng.ExpectL2BlockRefByNumber(0, refA0, nil)
				eng.ExpectForkchoiceUpdate(&eth.ForkchoiceState{
					HeadBlockHash:      refA3.Hash,
					SafeBlockHash:      refA0.Hash,
					FinalizedBlockHash: refA0.Hash,
				}, nil, &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEngine := &testutils.MockEngine{}
			tt.stub(mockEngine)
			ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0), metrics.NoopMetrics, cfg,
				&sync.Config{SyncMode: sync.ELSync, OffsetELSafe: tt.offset}, &testutils.MockL1Source{}, discardEmitter{}, nil)
			ec.SyncDeriver = noopSyncDeriver{}

			err := ec.InsertUnsafePayload(context.Background(), payload, refA3)
			require.NoError(t, err)
			require.Equal(t, refA3, ec.unsafeHead)
			require.Equal(t, tt.want, ec.localSafeHead)
			require.Equal(t, tt.want, ec.FinalizedHead())
		})
	}
}

// TestInsertUnsafePayload_ELSync_preservesSafeDB reproduces issue #21089: an op-reth
// node (SupportsPostFinalizationELSync) restarting with --syncmode=execution-layer
// re-enters EL sync even though it already has a finalized head, and previously that
// wiped a populated safedb. The fix preserves the safedb and resumes from its tip; for a
// synced node the restart is a no-op for the heads.
func TestInsertUnsafePayload_ELSync_preservesSafeDB(t *testing.T) {
	cfg, _, refA1, refA2, refA3, payload := buildELSyncTipChain(t)

	db, err := safedb.NewSafeDB(testlog.Logger(t, 0), t.TempDir())
	require.NoError(t, err)
	defer db.Close()
	// Prior derivation recorded safe heads up to the tip.
	l1A := eth.BlockID{Hash: common.Hash{0xa1}, Number: 100}
	l1B := eth.BlockID{Hash: common.Hash{0xb2}, Number: 200}
	require.NoError(t, db.SafeHeadUpdated(refA1, l1A))
	require.NoError(t, db.SafeHeadUpdated(refA3, l1B))

	mockEngine := &testutils.MockEngine{}
	// A non-genesis finalized head: without SupportsPostFinalizationELSync the controller
	// would skip EL sync entirely. With it (op-reth), EL sync is (re-)entered.
	mockEngine.ExpectL2BlockRefByLabel(eth.Finalized, refA1, nil)
	mockEngine.ExpectNewPayload(payload.ExecutionPayload, nil, &eth.PayloadStatusV1{Status: eth.ExecutionValid}, nil)
	// The fix resolves the safedb tip against the synced chain before reusing it.
	mockEngine.ExpectL2BlockRefByHash(refA3.Hash, refA3, nil)
	// Restart is a no-op for the heads: safe stays at the tip, finalized stays where it was.
	mockEngine.ExpectForkchoiceUpdate(&eth.ForkchoiceState{
		HeadBlockHash:      refA3.Hash,
		SafeBlockHash:      refA3.Hash,
		FinalizedBlockHash: refA2.Hash,
	}, nil, &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}}, nil)

	ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0), metrics.NoopMetrics, cfg,
		&sync.Config{SyncMode: sync.ELSync, SupportsPostFinalizationELSync: true}, &testutils.MockL1Source{}, discardEmitter{}, nil)
	ec.SyncDeriver = &fakeSyncDeriver{db: db, tip: refA3.ID(), hasTip: true}
	// A synced node on restart: safe at the tip (refA3), finalized lagging by one (refA2).
	ec.SetLocalSafeHead(refA3)
	ec.SetFinalizedHead(refA2)

	require.NoError(t, ec.InsertUnsafePayload(context.Background(), payload, refA3))

	require.Equal(t, refA3, ec.unsafeHead)
	require.Equal(t, refA3, ec.localSafeHead)
	require.Equal(t, refA2, ec.FinalizedHead())

	// The earlier safedb entry must still be queryable: a restart must not wipe it.
	_, l2, err := db.SafeHeadAtL1(context.Background(), l1A.Number)
	require.NoError(t, err, "safedb was wiped on EL-sync restart")
	require.Equal(t, refA1.ID(), l2)
}

// TestInsertUnsafePayload_ELSync_resumesFromSafeDBBelowFinalized covers restoring a safedb
// whose tip is older than the node's finalized head (e.g. a backup a few hours old). Rather
// than wiping it, derivation resumes from the safedb tip: both safe and finalized are reset
// back to it (finalized moves backward and is re-advanced as derivation progresses), and the
// safedb is preserved gap-free.
func TestInsertUnsafePayload_ELSync_resumesFromSafeDBBelowFinalized(t *testing.T) {
	cfg, _, refA1, refA2, refA3, payload := buildELSyncTipChain(t)

	db, err := safedb.NewSafeDB(testlog.Logger(t, 0), t.TempDir())
	require.NoError(t, err)
	defer db.Close()
	// Restored safedb whose tip (refA1) is below the node's finalized head (refA2).
	l1A := eth.BlockID{Hash: common.Hash{0xa1}, Number: 100}
	require.NoError(t, db.SafeHeadUpdated(refA1, l1A))

	mockEngine := &testutils.MockEngine{}
	mockEngine.ExpectL2BlockRefByLabel(eth.Finalized, refA1, nil)
	mockEngine.ExpectNewPayload(payload.ExecutionPayload, nil, &eth.PayloadStatusV1{Status: eth.ExecutionValid}, nil)
	mockEngine.ExpectL2BlockRefByHash(refA1.Hash, refA1, nil)
	// Safe and finalized are reset back to the safedb tip (refA1); unsafe stays at the synced tip.
	mockEngine.ExpectForkchoiceUpdate(&eth.ForkchoiceState{
		HeadBlockHash:      refA3.Hash,
		SafeBlockHash:      refA1.Hash,
		FinalizedBlockHash: refA1.Hash,
	}, nil, &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}}, nil)

	ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0), metrics.NoopMetrics, cfg,
		&sync.Config{SyncMode: sync.ELSync, SupportsPostFinalizationELSync: true}, &testutils.MockL1Source{}, discardEmitter{}, nil)
	ec.SyncDeriver = &fakeSyncDeriver{db: db, tip: refA1.ID(), hasTip: true}
	// The node already finalized refA2, which is ahead of the restored safedb tip.
	ec.SetFinalizedHead(refA2)
	ec.SetLocalSafeHead(refA2)

	require.NoError(t, ec.InsertUnsafePayload(context.Background(), payload, refA3))

	// Finalized was moved back to the safedb tip so derivation can re-fill from there.
	require.Equal(t, refA1, ec.localSafeHead)
	require.Equal(t, refA1, ec.FinalizedHead())

	// The safedb is preserved, not wiped.
	_, l2, err := db.SafeHeadAtL1(context.Background(), l1A.Number)
	require.NoError(t, err, "safedb below finalized should be preserved, resuming derivation from its tip")
	require.Equal(t, refA1.ID(), l2)
}

// TestInsertUnsafePayload_ELSync_trimsReorgedSafeDBToOffset covers a safedb whose tip was
// dropped by a recent reorg (its tip isn't on the synced chain). Rather than wiping the whole db,
// the recorded safe head at/above the offset head is looked up; it's on the synced chain, so the
// db is trimmed to it and derivation resumes from there — only the recent window is re-derived.
func TestInsertUnsafePayload_ELSync_trimsReorgedSafeDBToOffset(t *testing.T) {
	cfg, refA0, refA1, _, refA3, payload := buildELSyncTipChain(t)

	// A safedb tip that isn't part of the synced chain (e.g. reorged out); the recorded safe head
	// at/above the offset head is refA1, which is on the synced chain.
	orphanTip := eth.BlockID{Hash: common.Hash{0xde, 0xad}, Number: 2}

	mockEngine := &testutils.MockEngine{}
	mockEngine.ExpectL2BlockRefByLabel(eth.Finalized, refA1, nil)
	mockEngine.ExpectNewPayload(payload.ExecutionPayload, nil, &eth.PayloadStatusV1{Status: eth.ExecutionValid}, nil)
	// offset=4s, BlockTime=2 → retract 2 blocks from the tip (refA3) to the offset head refA1.
	mockEngine.ExpectL2BlockRefByNumber(refA1.Number, refA1, nil)
	mockEngine.ExpectL2BlockRefByHash(orphanTip.Hash, eth.L2BlockRef{}, ethereum.NotFound)
	// The recorded safe head at the offset (refA1) is on the synced chain.
	mockEngine.ExpectL2BlockRefByHash(refA1.Hash, refA1, nil)
	// Safe resumes from the recorded head; finalized stays at the (lower) current finalized.
	mockEngine.ExpectForkchoiceUpdate(&eth.ForkchoiceState{
		HeadBlockHash:      refA3.Hash,
		SafeBlockHash:      refA1.Hash,
		FinalizedBlockHash: refA0.Hash,
	}, nil, &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}}, nil)

	ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0), metrics.NoopMetrics, cfg,
		&sync.Config{SyncMode: sync.ELSync, SupportsPostFinalizationELSync: true, OffsetELSafe: 4 * time.Second},
		&testutils.MockL1Source{}, discardEmitter{}, nil)
	deriver := &fakeSyncDeriver{tip: orphanTip, hasTip: true, anchor: refA1.ID(), hasAnchor: true}
	ec.SyncDeriver = deriver
	ec.SetFinalizedHead(refA0)
	ec.SetLocalSafeHead(refA0)

	require.NoError(t, ec.InsertUnsafePayload(context.Background(), payload, refA3))

	// Trimmed back to the recorded head and resumed from there — not fully wiped.
	require.NotNil(t, deriver.resetTo, "safedb should have been trimmed")
	require.Equal(t, refA1, *deriver.resetTo)
	require.Equal(t, refA1, ec.localSafeHead)
	require.Equal(t, refA0, ec.FinalizedHead())
}

// TestInsertUnsafePayload_ELSync_clearsInconsistentSafeDB covers a safedb that can't be trusted:
// its tip isn't on the synced chain, and the recorded safe head at the offset head isn't either
// (e.g. a foreign restore). It is cleared entirely and derivation falls back to the offset head.
func TestInsertUnsafePayload_ELSync_clearsInconsistentSafeDB(t *testing.T) {
	cfg, _, refA1, _, refA3, payload := buildELSyncTipChain(t)

	orphanTip := eth.BlockID{Hash: common.Hash{0xde, 0xad}, Number: 2}
	foreignAnchor := eth.BlockID{Hash: common.Hash{0xfe}, Number: 1}

	mockEngine := &testutils.MockEngine{}
	mockEngine.ExpectL2BlockRefByLabel(eth.Finalized, refA1, nil)
	mockEngine.ExpectNewPayload(payload.ExecutionPayload, nil, &eth.PayloadStatusV1{Status: eth.ExecutionValid}, nil)
	mockEngine.ExpectL2BlockRefByNumber(refA1.Number, refA1, nil)
	// Neither the tip nor the recorded head at the offset is on the synced chain.
	mockEngine.ExpectL2BlockRefByHash(orphanTip.Hash, eth.L2BlockRef{}, ethereum.NotFound)
	mockEngine.ExpectL2BlockRefByHash(foreignAnchor.Hash, eth.L2BlockRef{}, ethereum.NotFound)
	mockEngine.ExpectForkchoiceUpdate(&eth.ForkchoiceState{
		HeadBlockHash:      refA3.Hash,
		SafeBlockHash:      refA1.Hash,
		FinalizedBlockHash: refA1.Hash,
	}, nil, &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}}, nil)

	ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0), metrics.NoopMetrics, cfg,
		&sync.Config{SyncMode: sync.ELSync, SupportsPostFinalizationELSync: true, OffsetELSafe: 4 * time.Second},
		&testutils.MockL1Source{}, discardEmitter{}, nil)
	deriver := &fakeSyncDeriver{tip: orphanTip, hasTip: true, anchor: foreignAnchor, hasAnchor: true}
	ec.SyncDeriver = deriver

	require.NoError(t, ec.InsertUnsafePayload(context.Background(), payload, refA3))

	// Cleared entirely (reset to a zero ref), then resumed from the offset head.
	require.NotNil(t, deriver.resetTo, "safedb should have been reset")
	require.Equal(t, eth.L2BlockRef{}, *deriver.resetTo)
	require.Equal(t, refA1, ec.localSafeHead)
}
