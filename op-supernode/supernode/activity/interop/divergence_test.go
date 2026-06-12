package interop

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	coreinterop "github.com/ethereum-optimism/optimism/op-core/interop"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/engine_controller"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/resources"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// These tests guard the divergence-halt contract: when durable local state
// (logsDB, verifiedDB, or a WAL'd transition derived from them) contradicts
// the chain or itself, the verification loop must stop terminally with the
// halted gauge set and the pending transition preserved — never spin on the
// error-backoff retry loop, where observeRound (and with it all reorg
// detection) is starved by the pending transition.

// gatheredGauge reads a label-less gauge value directly from the registry.
func gatheredGauge(t *testing.T, m *resources.SupernodeMetrics, metricName string) (float64, bool) {
	t.Helper()
	families, err := m.Registry().Gather()
	require.NoError(t, err)
	for _, mf := range families {
		if mf.GetName() != metricName {
			continue
		}
		for _, metric := range mf.GetMetric() {
			return metric.GetGauge().GetValue(), true
		}
	}
	return 0, false
}

func requireHalted(t *testing.T, h *interopTestHarness) {
	t.Helper()
	v, found := gatheredGauge(t, h.interop.metrics, "supernode_interop_activity_state")
	require.True(t, found, "activity state gauge must exist")
	require.Equal(t, float64(InteropStateHalted), v, "activity state gauge must be halted")
}

// TestIsDivergenceError pins the terminal-vs-transient classification: every
// durable-divergence sentinel (including op-core/interop.ErrConflict from a
// logsDB rewind hash mismatch) is terminal, survives wrapping, and ordinary
// errors are not terminal.
func TestIsDivergenceError(t *testing.T) {
	terminal := []error{
		ErrStaleLogsDB,
		ErrParentHashMismatch,
		ErrHeadRegression,
		ErrRewindChainSetMismatch,
		coreinterop.ErrConflict,
	}
	for _, e := range terminal {
		require.Truef(t, isDivergenceError(e), "%v should be terminal", e)
		require.Truef(t, isDivergenceError(fmt.Errorf("rewind logsDB: %w", e)), "%v should be terminal when wrapped", e)
	}
	for _, e := range []error{errors.New("connection refused"), coreinterop.ErrFuture} {
		require.Falsef(t, isDivergenceError(e), "%v should be transient/non-terminal", e)
	}
}

// TestProgress_HaltsOnStaleLogsDBConflict: a WAL'd Advance whose block
// conflicts with already-sealed logsDB content at the same height (possible
// only when the determinism assumption breaks — reorg under canonical L1,
// foreign data dir, derivation bug) must halt terminally, preserving the WAL.
func TestProgress_HaltsOnStaleLogsDBConflict(t *testing.T) {
	h := newInteropTestHarness(t).WithChain(10, nil).Build()
	chainID := eth.ChainIDFromUInt64(10)

	sealedHash := common.HexToHash("0xaaaa")
	conflictingHash := common.HexToHash("0xbbbb")

	// Sealed history says block 1001 is sealedHash.
	require.NoError(t, h.interop.logsDBs[chainID].SealBlock(
		common.Hash{}, eth.BlockID{Number: 1001, Hash: sealedHash}, 1001))

	// The WAL'd Advance claims block 1001 is conflictingHash.
	pending := PendingTransition{
		Decision: DecisionAdvance,
		Result: &Result{
			Timestamp:   1001,
			L1Inclusion: eth.BlockID{Number: 5, Hash: common.HexToHash("0x11")},
			L2Heads:     map[eth.ChainID]eth.BlockID{chainID: {Number: 1001, Hash: conflictingHash}},
		},
	}
	require.NoError(t, h.interop.verifiedDB.SetPendingTransition(pending))

	sleep, err := h.interop.progress()
	require.Error(t, err, "stale logsDB conflict must terminate the loop")
	require.ErrorIs(t, err, ErrStaleLogsDB)
	require.Zero(t, sleep)
	requireHalted(t, h)

	// The conflicting transition must be preserved for the operator.
	got, perr := h.interop.verifiedDB.GetPendingTransition()
	require.NoError(t, perr)
	require.NotNil(t, got, "halt must preserve the pending transition")
}

// TestProgress_HaltsOnParentHashMismatch: a WAL'd Advance extending sealed
// history with a block whose parent hash disagrees with the sealed tip is the
// next-height variant of the same divergence and must also halt.
func TestProgress_HaltsOnParentHashMismatch(t *testing.T) {
	h := newInteropTestHarness(t).WithChain(10, nil).Build()
	chainID := eth.ChainIDFromUInt64(10)

	// Sealed tip at 1000 with a hash that cannot be the parent of the WAL'd
	// block: the mock derives the new block's parent hash from its number
	// (BigToHash(1000)), so any other tip hash mismatches.
	require.NoError(t, h.interop.logsDBs[chainID].SealBlock(
		common.Hash{}, eth.BlockID{Number: 1000, Hash: common.HexToHash("0xcccc")}, 1000))

	pending := PendingTransition{
		Decision: DecisionAdvance,
		Result: &Result{
			Timestamp:   1001,
			L1Inclusion: eth.BlockID{Number: 5, Hash: common.HexToHash("0x11")},
			L2Heads:     map[eth.ChainID]eth.BlockID{chainID: {Number: 1001, Hash: common.HexToHash("0xdddd")}},
		},
	}
	require.NoError(t, h.interop.verifiedDB.SetPendingTransition(pending))

	sleep, err := h.interop.progress()
	require.Error(t, err, "parent hash mismatch must terminate the loop")
	require.ErrorIs(t, err, ErrParentHashMismatch)
	require.Zero(t, sleep)
	requireHalted(t, h)
}

// TestHandoffGap_MetricReflectsWindow verifies the startup-handoff gap gauge:
// firstVerifiableTimestamp - activationTimestamp. The window is reported
// "verified" without being verified, so its size must be observable.
func TestHandoffGap_MetricReflectsWindow(t *testing.T) {
	t.Run("zero when verification starts at activation", func(t *testing.T) {
		h := newInteropTestHarness(t).WithActivation(1000).WithChain(10, nil).Build()
		// Harness with chains sets verificationStartTimestamp = activation+1.
		// Override to exactly activation to model a clean start.
		h.interop.verificationStartTimestamp = 1000
		h.interop.recordHandoffGap()
		v, found := gatheredGauge(t, h.interop.metrics, "supernode_interop_handoff_gap_seconds")
		require.True(t, found)
		require.Equal(t, float64(0), v)
	})

	t.Run("reflects a wide window from a late first SafeDB entry", func(t *testing.T) {
		h := newInteropTestHarness(t).WithActivation(1000).WithChain(10, nil).Build()
		h.interop.verificationStartTimestamp = 5000 // e.g. reseeded node
		h.interop.recordHandoffGap()
		v, found := gatheredGauge(t, h.interop.metrics, "supernode_interop_handoff_gap_seconds")
		require.True(t, found)
		require.Equal(t, float64(4000), v)
	})
}

// TestPruneLogsDBs_DropsBelowRetentionHorizon exercises the verifier-level
// pruning wiring against the real raftwallogdb logsDB the harness opens: it
// computes horizon = lastVerifiedTimestamp - retentionWindow and prunes each
// chain's logsDB below it, tracking the pruned counter.
func TestPruneLogsDBs_DropsBelowRetentionHorizon(t *testing.T) {
	h := newInteropTestHarness(t).WithChain(10, nil).Build()
	chainID := eth.ChainIDFromUInt64(10)
	db := h.interop.logsDBs[chainID]

	// Seal blocks 1..10 with timestamps 100..1000.
	prev := common.Hash{}
	for n := uint64(1); n <= 10; n++ {
		blk := eth.BlockID{Number: n, Hash: common.BigToHash(new(big.Int).SetUint64(n))}
		require.NoError(t, db.SealBlock(prev, blk, n*100))
		prev = blk.Hash
	}

	// Frontier at ts 1000, retention window 400 → horizon 600. Blocks with
	// ts<600 (1..5) prune; block 6 (ts 600) and the tip stay.
	require.NoError(t, h.interop.verifiedDB.Commit(VerifiedResult{
		Timestamp: 1000,
		L2Heads:   map[eth.ChainID]eth.BlockID{chainID: {Number: 1000, Hash: common.HexToHash("0xfeed")}},
	}))
	h.interop.logsDBRetentionWindow = 400

	h.interop.pruneLogsDBs()

	first, err := db.FirstSealedBlock()
	require.NoError(t, err)
	require.Equal(t, uint64(6), first.Number, "blocks below the horizon should be pruned")
	latest, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, uint64(10), latest.Number, "tip must survive")

	horizon, found := gatheredGauge(t, h.interop.metrics, "supernode_logsdb_prune_horizon_timestamp")
	require.True(t, found)
	require.Equal(t, float64(600), horizon)

	pruned, found := gatheredCounter(t, h.interop.metrics, "supernode_logsdb_pruned_total", "chain_id", chainID.String())
	require.True(t, found)
	require.Equal(t, float64(5), pruned)
}

// TestPruneLogsDBs_NoOpBelowWindow: with insufficient verified history above the
// retention window, nothing is pruned (and no underflow).
func TestPruneLogsDBs_NoOpBelowWindow(t *testing.T) {
	h := newInteropTestHarness(t).WithChain(10, nil).Build()
	chainID := eth.ChainIDFromUInt64(10)
	db := h.interop.logsDBs[chainID]

	prev := common.Hash{}
	for n := uint64(1); n <= 5; n++ {
		blk := eth.BlockID{Number: n, Hash: common.BigToHash(new(big.Int).SetUint64(n))}
		require.NoError(t, db.SealBlock(prev, blk, n*100))
		prev = blk.Hash
	}
	require.NoError(t, h.interop.verifiedDB.Commit(VerifiedResult{
		Timestamp: 500,
		L2Heads:   map[eth.ChainID]eth.BlockID{chainID: {Number: 500, Hash: common.HexToHash("0xfeed")}},
	}))
	h.interop.logsDBRetentionWindow = 1000 // window exceeds lastTS → no prune

	h.interop.pruneLogsDBs()

	first, err := db.FirstSealedBlock()
	require.NoError(t, err)
	require.Equal(t, uint64(1), first.Number, "nothing should be pruned")
}

// TestProgress_HaltsOnRewindOverFinalized: an invalidation whose engine rewind
// is rejected because the target is at/below the finalized head is permanent —
// the finalized head never moves backward — so replaying the WAL'd transition
// can only fail again. The loop must halt instead of retrying every backoff
// forever. Reachable via transitive invalidation (a block finalized in an
// earlier round becoming invalid in a later one).
func TestProgress_HaltsOnRewindOverFinalized(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(10)
	h := newInteropTestHarness(t).WithChain(10, func(m *mockChainContainer) {
		// InvalidateBlock surfaces the engine controller's permanent error,
		// wrapped as the real path does (chain_container wraps with %w).
		m.invalidateBlockErr = fmt.Errorf("failed to rewind engine: %w", engine_controller.ErrRewindOverFinalizedHead)
	}).Build()

	parent := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		BlockNumber: 1000,
		BlockHash:   common.HexToHash("0xparent"),
	}}
	pending := PendingTransition{
		Decision: DecisionInvalidate,
		Result: &Result{
			Timestamp: 1001,
			InvalidHeads: map[eth.ChainID]InvalidHead{
				chainID: {BlockID: eth.BlockID{Number: 1001, Hash: common.HexToHash("0xbad")}},
			},
		},
		InvalidationParentPayloads: map[eth.ChainID]*eth.ExecutionPayloadEnvelope{chainID: parent},
	}
	require.NoError(t, h.interop.verifiedDB.SetPendingTransition(pending))

	sleep, err := h.interop.progress()
	require.Error(t, err, "rewind over finalized head must terminate the loop")
	require.ErrorIs(t, err, engine_controller.ErrRewindOverFinalizedHead)
	require.Zero(t, sleep)
	requireHalted(t, h)

	// Permanent failure must preserve the transition for operator inspection.
	got, perr := h.interop.verifiedDB.GetPendingTransition()
	require.NoError(t, perr)
	require.NotNil(t, got)
}

// TestProgress_HaltsOnRewindRejected: when the engine declares a rewind payload
// invalid (deterministic — a replay re-submits identical inputs), the loop must
// halt rather than retry forever. Subsumes the rare synthetic-block header-
// validation rejection.
func TestProgress_HaltsOnRewindRejected(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(10)
	h := newInteropTestHarness(t).WithChain(10, func(m *mockChainContainer) {
		m.invalidateBlockErr = fmt.Errorf("failed to rewind engine: %w", engine_controller.ErrRewindSyntheticPayloadRejected)
	}).Build()

	parent := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		BlockNumber: 1000,
		BlockHash:   common.HexToHash("0xparent"),
	}}
	pending := PendingTransition{
		Decision: DecisionInvalidate,
		Result: &Result{
			Timestamp: 1001,
			InvalidHeads: map[eth.ChainID]InvalidHead{
				chainID: {BlockID: eth.BlockID{Number: 1001, Hash: common.HexToHash("0xbad")}},
			},
		},
		InvalidationParentPayloads: map[eth.ChainID]*eth.ExecutionPayloadEnvelope{chainID: parent},
	}
	require.NoError(t, h.interop.verifiedDB.SetPendingTransition(pending))

	sleep, err := h.interop.progress()
	require.Error(t, err)
	require.ErrorIs(t, err, engine_controller.ErrRewindSyntheticPayloadRejected)
	require.Zero(t, sleep)
	requireHalted(t, h)
}

// TestCommit_HeadRegression guards the defense-in-depth commit guard mirroring
// the Dafny model's Commit precondition: per-chain heads are monotone across
// consecutive entries and an unchanged height keeps an unchanged hash, while
// chains may join or leave the set (dependency set changes).
func TestCommit_HeadRegression(t *testing.T) {
	t.Parallel()
	chainA := eth.ChainIDFromUInt64(10)
	chainB := eth.ChainIDFromUInt64(20)
	head := func(n uint64, h string) eth.BlockID {
		return eth.BlockID{Number: n, Hash: common.HexToHash(h)}
	}

	tests := []struct {
		name    string
		next    map[eth.ChainID]eth.BlockID
		wantErr bool
	}{
		{
			name:    "head number regression rejected",
			next:    map[eth.ChainID]eth.BlockID{chainA: head(99, "0xa99")},
			wantErr: true,
		},
		{
			name:    "hash change at unchanged height rejected",
			next:    map[eth.ChainID]eth.BlockID{chainA: head(100, "0xdead")},
			wantErr: true,
		},
		{
			name: "unchanged head accepted",
			next: map[eth.ChainID]eth.BlockID{chainA: head(100, "0xa100")},
		},
		{
			name: "advance by one accepted",
			next: map[eth.ChainID]eth.BlockID{chainA: head(101, "0xa101")},
		},
		{
			name: "chain joining the set accepted",
			next: map[eth.ChainID]eth.BlockID{chainA: head(101, "0xa101"), chainB: head(7, "0xb7")},
		},
		{
			name: "chain leaving the set accepted",
			next: map[eth.ChainID]eth.BlockID{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			db, err := OpenVerifiedDB(t.TempDir())
			require.NoError(t, err)
			defer db.Close()

			require.NoError(t, db.Commit(VerifiedResult{
				Timestamp: 1000,
				L2Heads:   map[eth.ChainID]eth.BlockID{chainA: head(100, "0xa100")},
			}))

			err = db.Commit(VerifiedResult{Timestamp: 1001, L2Heads: tc.next})
			if tc.wantErr {
				require.ErrorIs(t, err, ErrHeadRegression)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestBuildRewindPlan_ChainSetMismatch: an engine-reset rewind targeting a
// verified entry whose chain set differs from the configured set must fail at
// build time with the halt-classified sentinel instead of producing a plan
// that wedges forever in resetChainEnginesIfNeeded. Without an engine reset,
// chain-set drift stays tolerated.
func TestBuildRewindPlan_ChainSetMismatch(t *testing.T) {
	commit := func(t *testing.T, h *interopTestHarness, ts uint64, chains ...uint64) {
		heads := make(map[eth.ChainID]eth.BlockID, len(chains))
		for _, id := range chains {
			heads[eth.ChainIDFromUInt64(id)] = eth.BlockID{
				Number: ts, Hash: common.BigToHash(new(big.Int).SetUint64(ts))}
		}
		require.NoError(t, h.interop.verifiedDB.Commit(VerifiedResult{Timestamp: ts, L2Heads: heads}))
	}
	// Keyed at the rewind timestamp so HasDeniedAtOrAfterTimestamp(lastTS)
	// reports true and buildRewindPlan takes the engine-reset path.
	denied := map[uint64][]common.Hash{2001: {common.HexToHash("0xbad")}}

	t.Run("configured chain missing from target entry", func(t *testing.T) {
		h := newInteropTestHarness(t).
			WithChain(10, func(m *mockChainContainer) { m.pruneDeniedResult = denied }).
			WithChain(20, nil).
			Build()
		commit(t, h, 2000, 10) // chain 20 configured but absent at the rewind target
		commit(t, h, 2001, 10, 20)

		_, err := h.interop.buildRewindPlan(2001)
		require.ErrorIs(t, err, ErrRewindChainSetMismatch)
	})

	t.Run("target entry chain no longer configured", func(t *testing.T) {
		h := newInteropTestHarness(t).
			WithChain(10, func(m *mockChainContainer) { m.pruneDeniedResult = denied }).
			Build()
		commit(t, h, 2000, 10, 30) // chain 30 in history but not configured
		commit(t, h, 2001, 10, 30)

		_, err := h.interop.buildRewindPlan(2001)
		require.ErrorIs(t, err, ErrRewindChainSetMismatch)
	})

	t.Run("drift tolerated without engine reset", func(t *testing.T) {
		h := newInteropTestHarness(t).
			WithChain(10, nil). // no deny-list entries: plain rewind, no reset
			WithChain(20, nil).
			Build()
		commit(t, h, 2000, 10)
		commit(t, h, 2001, 10, 20)

		plan, err := h.interop.buildRewindPlan(2001)
		require.NoError(t, err)
		require.Nil(t, plan.ResetAllChainsTo)
		require.Len(t, plan.TargetHeads, 1)
	})
}
