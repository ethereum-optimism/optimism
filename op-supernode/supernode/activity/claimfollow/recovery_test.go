package claimfollow

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestRecoveryBlockRequiresCanonicalDepositOnlyInputs(t *testing.T) {
	h := newHarness(t)
	h.r.fill(1, 4, "a", 0)
	h.r.safe = 4
	require.NoError(t, h.step())
	target, err := derive.PayloadToBlockRef(testRollupCfg(), h.r.blocks[4].env.ExecutionPayload)
	require.NoError(t, err)
	h.f.recoveryTarget = target
	api := NewAPI(h.f)
	ref, err := api.RecoveryBlock(t.Context(), 2, target.ID())
	require.NoError(t, err)
	require.Equal(t, uint64(2), ref.Number)
	require.Equal(t, renderHash(2, "a"), ref.Hash)
	_, err = api.RecoveryBlock(t.Context(), 0, target.ID())
	require.ErrorContains(t, err, "outside")
	_, err = api.RecoveryBlock(t.Context(), 5, target.ID())
	require.ErrorContains(t, err, "outside")
	_, err = api.RecoveryBlock(t.Context(), 2, eth.BlockID{Hash: common.Hash{0xff}, Number: 4})
	require.ErrorContains(t, err, "target changed")
	h.r.set(2, "a", 0, claimTx(t, 0, 2, 4))
	_, err = api.RecoveryBlock(t.Context(), 2, target.ID())
	require.ErrorContains(t, err, "sequencer transactions")
}

func TestPartialClaimRecoverySurvivesRestart(t *testing.T) {
	for _, invalidated := range []uint64{1, 4, 8} {
		t.Run(fmt.Sprintf("block_%d", invalidated), func(t *testing.T) {
			h := newHarness(t)
			h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
			h.r.fill(2, 8, "a", 0)
			h.r.safe = 8
			require.NoError(t, h.step())
			denied := h.r.blocks[invalidated].env.ExecutionPayload.BlockHash
			h.r.denied[invalidated] = []common.Hash{denied}
			h.r.fill(invalidated, 8, "b", invalidated)
			require.NoError(t, h.step())
			require.Equal(t, wantGenesisRef(), h.status().LocalSafeL2)
			check := func(m *Module) {
				t.Helper()
				status, err := NewAPI(m).SyncStatus(t.Context())
				require.NoError(t, err)
				require.NotNil(t, status.Recovery)
				if invalidated == 1 {
					require.Nil(t, status.Recovery.Prefix, "an erased carrier authenticates no prefix")
				} else {
					require.NotNil(t, status.Recovery.Prefix)
					require.Equal(t, invalidated-1, status.Recovery.Prefix.Last.Number)
					require.Equal(t, privHash(8), status.Recovery.Prefix.Terminal.Hash)
				}
			}
			check(h.f)
			// All volatile scan state is lost. The retained canonical carrier and
			// persisted denial identities must reconstruct the same recovery plan.
			restarted := New(h.f.cfg, testRollupCfg(), h.f.log, nil)
			restarted.Attach(h.r)
			require.NoError(t, restarted.Step(t.Context()))
			check(restarted)
		})
	}
}

func TestClaimFollowSeparatesLocalAndCrossSafety(t *testing.T) {
	h := newHarness(t)
	h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
	h.r.fill(2, 8, "a", 0)
	h.r.localSafe, h.r.safe = 8, 3
	require.NoError(t, h.step())
	require.Equal(t, wantRef(8), h.status().LocalSafeL2)
	require.Equal(t, wantGenesisRef(), h.status().SafeL2)
	h.r.safe = 8
	require.NoError(t, h.step())
	require.Equal(t, wantRef(8), h.status().SafeL2)
	h.r.safe = 3
	require.NoError(t, h.step())
	require.Equal(t, wantRef(8), h.status().LocalSafeL2)
	require.Equal(t, wantGenesisRef(), h.status().SafeL2)
}

func TestBoundedCatchupRetainsCursorBelowPublicFinality(t *testing.T) {
	h := newHarnessWithConfig(t, Config{Registry: registryAddr, GenesisHash: privateGenesisHash(), MaxBlocksPerPoll: 3})
	h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
	h.r.fill(2, 8, "a", 0)
	h.r.safe, h.r.finalized = 8, 8
	for range 3 {
		require.NoError(t, h.step())
	}
	require.Equal(t, wantRef(8), h.status().FinalizedL2)
}

func TestReplacementRecoveryRequiresAvailableDeniedAncestry(t *testing.T) {
	h := newHarness(t)
	h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
	h.r.fill(2, 8, "a", 0)
	old := h.r.blocks[4].env.ExecutionPayload.BlockHash
	h.r.denied[4] = []common.Hash{old}
	h.r.fill(4, 8, "b", 4)
	h.r.safe = 8
	delete(h.r.byHash, old)
	require.ErrorContains(t, h.step(), "denied projection header")
	require.Equal(t, wantGenesisRef(), h.status().LocalSafeL2)
}

func TestTemporarySafetyRetreatCanRestoreSameClaim(t *testing.T) {
	for _, initiallyScanned := range []uint64{5, 8} {
		t.Run(fmt.Sprint(initiallyScanned), func(t *testing.T) {
			h := newHarness(t)
			h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
			h.r.fill(2, 8, "a", 0)
			h.r.safe = initiallyScanned
			require.NoError(t, h.step())
			h.r.safe = 3
			require.NoError(t, h.step())
			require.Equal(t, wantGenesisRef(), h.status().LocalSafeL2)
			h.r.safe = 8
			require.NoError(t, h.step())
			require.Equal(t, wantRef(8), h.status().LocalSafeL2)
		})
	}
}

type resettingRendering struct {
	*fakeRendering
	reset func()
}

func (r *resettingRendering) PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error) {
	if r.reset != nil {
		reset := r.reset
		r.reset = nil
		reset()
	}
	return r.fakeRendering.PayloadByNumber(ctx, number)
}
func TestRecoveryBlockRejectsRevokedSnapshotWithUnchangedEL(t *testing.T) {
	h := newHarness(t)
	h.r.fill(1, 4, "a", 0)
	h.r.safe = 4
	require.NoError(t, h.step())
	target, err := derive.PayloadToBlockRef(testRollupCfg(), h.r.blocks[4].env.ExecutionPayload)
	require.NoError(t, err)
	h.f.recoveryTarget = target
	h.f.Attach(&resettingRendering{fakeRendering: h.r, reset: func() { h.f.rewind(1) }})
	_, err = NewAPI(h.f).RecoveryBlock(t.Context(), 2, target.ID())
	require.ErrorContains(t, err, "snapshot was revoked")
}

func TestTemporaryRetreatPreservesConfirmedReplacementBoundary(t *testing.T) {
	h := newHarness(t)
	h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
	h.r.fill(2, 8, "a", 0)
	h.r.safe = 8
	require.NoError(t, h.step())
	h.r.denied[6] = []common.Hash{h.r.blocks[6].env.ExecutionPayload.BlockHash}
	h.r.fill(6, 8, "b", 6)
	require.NoError(t, h.step())
	require.Equal(t, uint64(6), h.f.pending[0].replacementFrom)
	h.r.safe = 3
	require.NoError(t, h.step())
	h.r.safe = 8
	require.NoError(t, h.step())
	status, err := NewAPI(h.f).SyncStatus(t.Context())
	require.NoError(t, err)
	require.NotNil(t, status.Recovery.Prefix)
	require.Equal(t, uint64(5), status.Recovery.Prefix.Last.Number)
	require.Equal(t, uint64(6), h.f.pending[0].invalidFrom)
	require.Equal(t, wantGenesisRef(), status.LocalSafeL2)
}

type resettingStatusRendering struct {
	*fakeRendering
	reset func()
}

func (r *resettingStatusRendering) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	status, err := r.fakeRendering.SyncStatus(ctx)
	if r.reset != nil {
		reset := r.reset
		r.reset = nil
		reset()
	}
	return status, err
}
func TestClaimFollowRejectsStatusFromBeforeReset(t *testing.T) {
	h := newHarness(t)
	h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
	h.r.fill(2, 8, "a", 0)
	h.r.safe = 8
	require.NoError(t, h.step())
	h.f.Attach(&resettingStatusRendering{fakeRendering: h.r, reset: func() {
		h.f.rewind(3)
		h.r.fill(4, 8, "b", 4)
	}})
	require.ErrorContains(t, h.step(), "frontier changed")
	require.Zero(t, h.f.recoveryTarget)
	require.Equal(t, wantGenesisRef(), h.status().LocalSafeL2)
}
