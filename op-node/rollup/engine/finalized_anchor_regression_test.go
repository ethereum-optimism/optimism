package engine

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// TestEngineController_ForceResetSeedsSuperAuthorityFinalizedCache reproduces
// the finalized-head regression observed on interop-sdm-v2 (2026-10-01).
//
// Sequence in production:
//  1. op-supernode invalidates a block (invalid executing message), rewinds the
//     engine and restarts the chain's virtual node. The verifier activity is
//     Reset() and has no verified-DB entry at FinalizedL1 any more.
//  2. The new virtual node builds a fresh EngineController. Sync-start loads the
//     engine's persisted finalized label and the reset applies it. The
//     SuperAuthority finalized cache, the only monotonicity guard for the Anchor
//     branch, is empty because it lives in memory.
//  3. SuperAuthority.FinalizedL2Head returns Anchor{ts = L2 genesis time}
//     (verifierContribution clamps "no entry" to genesis).
//  4. FinalizedHead() resolves the anchor to block 0, finds no cache, and the
//     reset forkchoice update is sent with finalized = genesis.
//
// The existing TestEngineController_FinalizedHeadAnchorDoesNotRegressCache only
// covers a pre-populated cache; this test covers the post-restart reset path.
func TestEngineController_ForceResetSeedsSuperAuthorityFinalizedCache(t *testing.T) {
	const genesisTime = uint64(1790721048)
	genesisRef := eth.L2BlockRef{Hash: common.Hash{0x5d, 0xd4}, Number: 0, Time: genesisTime}
	cfg := &rollup.Config{
		BlockTime: 1,
		Genesis: rollup.Genesis{
			L2:     genesisRef.ID(),
			L2Time: genesisTime,
		},
	}

	// Verifier has no entry after the virtual-node restart: Anchor at genesis.
	superAuth := &mockSuperAuthority{
		finalizedL2HeadSource: rollup.VerifierHeadAnchor,
		finalizedTimestamp:    genesisTime,
	}

	// Heads as loaded by sync-start from the engine right after the restart.
	unsafe := eth.L2BlockRef{Hash: common.Hash{0x91, 0xf8}, Number: 168567, Time: genesisTime + 168567}
	safe := eth.L2BlockRef{Hash: common.Hash{0xa1, 0x65}, Number: 168564, Time: genesisTime + 168564}
	finalized := eth.L2BlockRef{Hash: common.Hash{0xf1, 0x69}, Number: 167033, Time: genesisTime + 167033}

	mockEngine := &testutils.MockEngine{}
	// The anchor resolves to genesis on every FinalizedHead() call; allow any number of lookups.
	var noErr error
	mockEngine.Mock.On("L2BlockRefByNumber", uint64(0)).Return(genesisRef, &noErr)
	// The reset forkchoice update must carry the persisted finalized head, not genesis.
	mockEngine.ExpectForkchoiceUpdate(
		&eth.ForkchoiceState{
			HeadBlockHash:      unsafe.Hash,
			SafeBlockHash:      safe.Hash,
			FinalizedBlockHash: finalized.Hash,
		},
		nil,
		&eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}},
		nil,
	)

	emitter := &testutils.MockEmitter{}
	emitter.ExpectOnceType("ForkchoiceUpdateEvent")
	emitter.ExpectOnceType("EngineResetConfirmedEvent")

	ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0), metrics.NoopMetrics,
		cfg, &sync.Config{}, &testutils.MockL1Source{}, emitter, superAuth)
	require.Equal(t, eth.L2BlockRef{}, ec.superAuthorityFinalizedHead, "precondition: fresh controller has no cache")

	// Mirror onResetEngineRequest: FindL2Heads read finalized from the engine's label.
	ec.seedSuperAuthorityFinalizedCache(&sync.FindHeadsResult{
		Unsafe: unsafe, Safe: safe, Finalized: finalized, FinalizedFromEngine: true,
	})
	ec.ForceReset(context.Background(), unsafe, safe, safe, finalized)

	require.Equal(t, finalized, ec.FinalizedHead(),
		"a verifier Anchor at genesis must not regress the finalized head below the reset finalized head")
	require.Equal(t, finalized, ec.superAuthorityFinalizedHead,
		"reset finalized head must seed the SuperAuthority finalized cache")
	mockEngine.AssertExpectations(t)
	emitter.AssertExpectations(t)
}

// TestEngineController_ResetDoesNotSeedSynthesizedFinalized covers the FindL2Heads
// EL-sync recovery branches. There Finalized is synthesized from the unsafe tip
// (minus OffsetELSafe), not read from the engine's finalized label, so it must not
// seed the SuperAuthority finalized cache: the cache is trusted without
// re-validation and would publish a block the verifier never approved as finalized.
func TestEngineController_ResetDoesNotSeedSynthesizedFinalized(t *testing.T) {
	const genesisTime = uint64(1790721048)
	genesisRef := eth.L2BlockRef{Hash: common.Hash{0x5d, 0xd4}, Number: 0, Time: genesisTime}
	cfg := &rollup.Config{
		BlockTime:     1,
		SeqWindowSize: 10,
		Genesis: rollup.Genesis{
			L2:     genesisRef.ID(),
			L2Time: genesisTime,
		},
	}
	superAuth := &mockSuperAuthority{
		fullyVerifiedL2HeadSource: rollup.VerifierHeadAnchor,
		fullyVerifiedTimestamp:    genesisTime,
		finalizedL2HeadSource:     rollup.VerifierHeadAnchor,
		finalizedTimestamp:        genesisTime,
	}

	// Engine finalized and safe labels at genesis with unsafe far ahead triggers the
	// "recovery from sync state without finality" branch of FindL2Heads.
	tip := eth.L2BlockRef{Hash: common.Hash{0x40}, Number: 400, Time: genesisTime + 400, L1Origin: eth.BlockID{Number: 300}}
	mockEngine := &testutils.MockEngine{}
	mockEngine.ExpectL2BlockRefByLabel(eth.Finalized, genesisRef, nil)
	mockEngine.ExpectL2BlockRefByLabel(eth.Safe, genesisRef, nil)
	mockEngine.ExpectL2BlockRefByLabel(eth.Unsafe, tip, nil)
	var noErr error
	mockEngine.Mock.On("L2BlockRefByNumber", uint64(0)).Return(genesisRef, &noErr)
	mockEngine.ExpectForkchoiceUpdate(
		&eth.ForkchoiceState{
			HeadBlockHash:      tip.Hash,
			SafeBlockHash:      genesisRef.Hash,
			FinalizedBlockHash: genesisRef.Hash,
		},
		nil,
		&eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}},
		nil,
	)
	emitter := &testutils.MockEmitter{}
	emitter.Mock.On("Emit", mock.Anything).Maybe()

	ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0), metrics.NoopMetrics,
		cfg, &sync.Config{}, &testutils.MockL1Source{}, emitter, superAuth)
	ec.onResetEngineRequest(context.Background())

	require.Equal(t, genesisRef, ec.superAuthorityFinalizedHead,
		"the cache must hold the SuperAuthority-gated head, not the synthesized recovery finalized head")
	require.Equal(t, genesisRef, ec.FinalizedHead(),
		"published finalized must stay gated by the SuperAuthority, not the unsafe tip")
	mockEngine.AssertExpectations(t)
}
