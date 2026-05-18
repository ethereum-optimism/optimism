package interop

import (
	"context"
	"errors"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestFastInit_ResumesFromVerifiedDB asserts a node with any committed entry
// resumes at LastTimestamp+1 without consulting SafeDB or wall-clock.
func TestFastInit_ResumesFromVerifiedDB(t *testing.T) {

	dataDir := t.TempDir()
	db, err := OpenVerifiedDB(dataDir)
	require.NoError(t, err)
	require.NoError(t, db.Commit(VerifiedResult{
		Timestamp:   500,
		L1Inclusion: eth.BlockID{Number: 1},
		L2Heads:     map[eth.ChainID]eth.BlockID{eth.ChainIDFromUInt64(10): {Number: 50}},
	}))
	require.NoError(t, db.Close())

	interop := New(testLogger(), 100, 0, nil, dataDir, nil, 0, nil)
	require.NotNil(t, interop)
	defer func() { require.NoError(t, interop.Stop(context.Background())) }()

	interop.tryInitFromVerifiedDB()
	require.True(t, interop.initialized.Load())
	require.False(t, interop.waitingForSync)
	require.Equal(t, uint64(501), interop.verificationStartTimestamp)
}

// TestFastInit_ResumeBelowActivationIsAllowed exercises the property that a
// pre-activation resume timestamp is valid: verification iterates harmlessly
// over rounds where no executing messages exist, and verifiedDB stays
// gap-free.
func TestFastInit_ResumeBelowActivationIsAllowed(t *testing.T) {

	dataDir := t.TempDir()
	db, err := OpenVerifiedDB(dataDir)
	require.NoError(t, err)
	require.NoError(t, db.Commit(VerifiedResult{
		Timestamp:   50,
		L1Inclusion: eth.BlockID{Number: 1},
		L2Heads:     map[eth.ChainID]eth.BlockID{eth.ChainIDFromUInt64(10): {Number: 5}},
	}))
	require.NoError(t, db.Close())

	interop := New(testLogger(), 1000, 0, nil, dataDir, nil, 0, nil)
	require.NotNil(t, interop)
	defer func() { require.NoError(t, interop.Stop(context.Background())) }()

	interop.tryInitFromVerifiedDB()
	require.True(t, interop.initialized.Load())
	require.Equal(t, uint64(51), interop.verificationStartTimestamp,
		"resume always uses LastTimestamp+1, never clamps to activation")
}

// TestFastInit_ColdStartDefersToLoop confirms that with no verifiedDB entry
// tryInitFromVerifiedDB sets waitingForSync without touching SafeDB or wall-clock.
func TestFastInit_ColdStartDefersToLoop(t *testing.T) {

	dataDir := t.TempDir()

	interop := New(testLogger(), 1000, 0, nil, dataDir, nil, 0, nil)
	require.NotNil(t, interop)
	defer func() { require.NoError(t, interop.Stop(context.Background())) }()

	interop.tryInitFromVerifiedDB()
	require.False(t, interop.initialized.Load())
	require.True(t, interop.waitingForSync)
	require.Zero(t, interop.verificationStartTimestamp)
}

// TestAdvanceColdStartInit_WaitsWhenAnyChainEmpty exercises the per-iteration
// gate: if any chain has no SafeDB entries yet, advanceColdStartInit returns
// (false, nil) so the loop backs off.
func TestAdvanceColdStartInit_WaitsWhenAnyChainEmpty(t *testing.T) {

	h := newInteropTestHarness(t).
		WithActivation(1000).
		WithChain(10, func(m *mockChainContainer) {
			m.firstSafeHeadTimestamp = 1234
			m.firstSafeHeadTimestampSet = true
		}).
		WithChain(20, func(m *mockChainContainer) {
			// Default: returns ErrSafeDBEmpty.
		}).
		Build()
	// Harness pre-sets initialized=true for tests that drive the verify
	// path; we're exercising cold start, so reset.
	h.interop.initialized.Store(false)
	h.interop.verificationStartTimestamp = 0

	advanced, err := h.interop.advanceColdStartInit()
	require.NoError(t, err)
	require.False(t, advanced, "must wait when any chain reports ErrSafeDBEmpty")
	require.False(t, h.interop.initialized.Load())
}

// TestAdvanceColdStartInit_PicksMaxClampedToActivation: with all chains
// reporting first SafeDB entries, verificationStartTimestamp is the max of
// (activation, T_c).
func TestAdvanceColdStartInit_PicksMaxClampedToActivation(t *testing.T) {

	t.Run("activation higher than chain timestamps", func(t *testing.T) {
		h := newInteropTestHarness(t).
			WithActivation(5000).
			WithChain(10, func(m *mockChainContainer) {
				m.firstSafeHeadTimestamp = 100
				m.firstSafeHeadTimestampSet = true
			}).
			WithChain(20, func(m *mockChainContainer) {
				m.firstSafeHeadTimestamp = 200
				m.firstSafeHeadTimestampSet = true
			}).
			Build()
		// logBackfillDepth=0 so backfill is a no-op.
		advanced, err := h.interop.advanceColdStartInit()
		require.NoError(t, err)
		require.True(t, advanced)
		require.Equal(t, uint64(5000), h.interop.verificationStartTimestamp)
	})

	t.Run("max chain timestamp higher than activation", func(t *testing.T) {
		h := newInteropTestHarness(t).
			WithActivation(1000).
			WithChain(10, func(m *mockChainContainer) {
				m.firstSafeHeadTimestamp = 1500
				m.firstSafeHeadTimestampSet = true
			}).
			WithChain(20, func(m *mockChainContainer) {
				m.firstSafeHeadTimestamp = 1750
				m.firstSafeHeadTimestampSet = true
			}).
			Build()
		advanced, err := h.interop.advanceColdStartInit()
		require.NoError(t, err)
		require.True(t, advanced)
		require.Equal(t, uint64(1750), h.interop.verificationStartTimestamp)
	})
}

// TestAdvanceColdStartInit_PropagatesNonEmptyErrors confirms that
// FirstSafeHeadTimestamp errors other than ErrSafeDBEmpty are fatal.
func TestAdvanceColdStartInit_PropagatesNonEmptyErrors(t *testing.T) {

	fault := errors.New("vn not running")
	h := newInteropTestHarness(t).
		WithActivation(1000).
		WithChain(10, func(m *mockChainContainer) {
			m.firstSafeHeadTimestampErr = fault
		}).
		Build()

	advanced, err := h.interop.advanceColdStartInit()
	require.Error(t, err)
	require.ErrorIs(t, err, fault)
	require.False(t, advanced)
}

// TestRunLoop_ColdStartTransition drives the loop from waitingForSync to
// initialized via a SafeDB entry appearing after a few iterations.
func TestRunLoop_ColdStartTransition(t *testing.T) {

	h := newInteropTestHarness(t).
		WithActivation(1000).
		WithChain(10, func(m *mockChainContainer) {
			m.blockAtTimestamp = eth.L2BlockRef{Number: 100, Hash: common.HexToHash("0x1")}
		}).
		Build()
	// Reset the harness-faked initialization so we can drive cold start.
	h.interop.initialized.Store(false)
	h.interop.verificationStartTimestamp = 0
	h.interop.waitingForSync = true

	mock := h.Mock(10)

	// First two iterations: SafeDB empty. Third: populated.
	var iterCount atomic.Int32
	go func() {
		// Background flipper: after a short delay, populate the chain's
		// first safe head timestamp so advanceColdStartInit can complete.
		time.Sleep(20 * time.Millisecond)
		mock.mu.Lock()
		mock.firstSafeHeadTimestamp = 1500
		mock.firstSafeHeadTimestampSet = true
		mock.mu.Unlock()
	}()

	// Loop a few times waiting for transition.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	h.interop.ctx = ctx
	for !h.interop.initialized.Load() {
		select {
		case <-ctx.Done():
			t.Fatal("cold start did not complete")
		default:
		}
		advanced, err := h.interop.advanceColdStartInit()
		require.NoError(t, err)
		if advanced {
			h.interop.initialized.Store(true)
			h.interop.waitingForSync = false
		} else {
			time.Sleep(10 * time.Millisecond)
		}
		iterCount.Add(1)
	}
	require.True(t, h.interop.initialized.Load())
	require.Equal(t, uint64(1500), h.interop.verificationStartTimestamp)
	require.GreaterOrEqual(t, iterCount.Load(), int32(2),
		"should have backed off at least once before SafeDB was populated")
}

// TestColdStartBackfill_NoOpWhenDepthZero confirms backfill is skipped when
// the operator disables it.
func TestColdStartBackfill_NoOpWhenDepthZero(t *testing.T) {

	h := newInteropTestHarness(t).
		WithActivation(1000).
		WithLogBackfillDepth(0).
		WithChain(10, func(m *mockChainContainer) {
			m.firstSafeHeadTimestamp = 1500
			m.firstSafeHeadTimestampSet = true
		}).
		Build()

	advanced, err := h.interop.advanceColdStartInit()
	require.NoError(t, err)
	require.True(t, advanced)
	require.Equal(t, uint64(1500), h.interop.verificationStartTimestamp)

	// logsDB must be empty: no backfill ran.
	_, has := h.interop.logsDBs[eth.ChainIDFromUInt64(10)].LatestSealedBlock()
	require.False(t, has, "no blocks should be sealed when logBackfillDepth=0")
}

// TestColdStartBackfill_NoOpWhenNoChains exercises the empty-chains short
// circuit so advanceColdStartInit completes against an empty depset.
func TestColdStartBackfill_NoOpWhenNoChains(t *testing.T) {

	h := newInteropTestHarness(t).
		WithActivation(1000).
		Build()
	require.Empty(t, h.interop.chains)

	advanced, err := h.interop.advanceColdStartInit()
	require.NoError(t, err)
	require.True(t, advanced, "empty depset means advance immediately")
	require.Equal(t, uint64(1000), h.interop.verificationStartTimestamp)
}

// TestColdStartBackfill_GenesisClamp exercises the per-chain genesis clamp.
// activationTimestamp=0, depth=1000s, verificationStart=2000 would naively
// yield start=1000; but the chain's genesis time is 1500, so backfill must
// not fetch any block whose timestamp falls below genesis.
func TestColdStartBackfill_GenesisClamp(t *testing.T) {

	depth := 1000 * time.Second
	var minFetched atomic.Uint64
	minFetched.Store(^uint64(0))
	h := newInteropTestHarness(t).
		WithActivation(0).
		WithLogBackfillDepth(depth).
		WithChain(10, func(m *mockChainContainer) {
			m.firstSafeHeadTimestamp = 2000
			m.firstSafeHeadTimestampSet = true
			m.blockNumberToTimestampOverride = func(_ context.Context, n uint64) (uint64, error) {
				if n == 0 {
					return 1500, nil
				}
				return n, nil
			}
			m.timestampToBlockNumberOverride = func(_ context.Context, ts uint64) (uint64, error) {
				return ts, nil
			}
			m.outputV0Override = func(_ context.Context, num uint64) (*eth.OutputV0, error) {
				for {
					prev := minFetched.Load()
					if num >= prev || minFetched.CompareAndSwap(prev, num) {
						break
					}
				}
				return &eth.OutputV0{
					BlockHash: common.BigToHash(new(big.Int).SetUint64(num)),
				}, nil
			}
		}).
		Build()
	h.interop.initialized.Store(false)
	h.interop.verificationStartTimestamp = 0

	advanced, err := h.interop.advanceColdStartInit()
	require.NoError(t, err)
	require.True(t, advanced)
	require.Equal(t, uint64(2000), h.interop.verificationStartTimestamp)

	require.GreaterOrEqual(t, minFetched.Load(), uint64(1500),
		"backfill must not fetch blocks before genesis")
}

// TestColdStartBackfill_MisalignedActivation: with blockTime=3 and
// activation=1000, TimestampToBlockNumber floors so the first sealed block's
// Time() is strictly pre-activation. sealBlockDataIntoLogsDB accepts this
// via the backfill exception (ts within one blockTime of activation).
func TestColdStartBackfill_MisalignedActivation(t *testing.T) {
	const (
		blockTime uint64 = 3
		act       uint64 = 1000
		safeTs    uint64 = 1020 // block 340 at blockTime=3
	)
	depth := 60 * time.Second
	blockNumToTime := func(num uint64) uint64 { return num * blockTime }

	h := newInteropTestHarness(t).
		WithActivation(act).
		WithLogBackfillDepth(depth).
		WithChain(10, func(m *mockChainContainer) {
			m.firstSafeHeadTimestamp = safeTs
			m.firstSafeHeadTimestampSet = true
			m.blockTimeOverride = blockTime
			m.blockInfoTimeFn = blockNumToTime
			m.timestampToBlockNumberOverride = func(_ context.Context, ts uint64) (uint64, error) {
				return ts / blockTime, nil
			}
			m.blockNumberToTimestampOverride = func(_ context.Context, n uint64) (uint64, error) {
				return blockNumToTime(n), nil
			}
		}).
		Build()
	h.interop.initialized.Store(false)
	h.interop.verificationStartTimestamp = 0

	advanced, err := h.interop.advanceColdStartInit()
	require.NoError(t, err)
	require.True(t, advanced)
	require.Equal(t, safeTs, h.interop.verificationStartTimestamp)
	require.Equal(t, act, h.interop.activationTimestamp,
		"protocol activation must not change")

	db := h.interop.logsDBs[eth.ChainIDFromUInt64(10)]
	first, err := db.FirstSealedBlock()
	require.NoError(t, err)
	require.Less(t, first.Timestamp, act,
		"anchor is strictly pre-activation: TimestampToBlockNumber(activation) floors")
	require.Greater(t, first.Timestamp+blockTime, act,
		"anchor must still be within one blockTime of activation")

	latest, has := db.LatestSealedBlock()
	require.True(t, has)
	require.Equal(t, safeTs/blockTime-1, latest.Number,
		"backfill seals up to and including TimestampToBlockNumber(verificationStart-1)")
}

// TestColdStartBackfill_RecoversFromOfflineReorg covers the crash-then-reorg
// case: a prior cold-start sealed blocks before any verifiedDB commit, then
// the chain reorged while supernode was offline. On restart, cold-start
// re-runs and reconcileLogsDBTail must rewind (or clear) the stale tail so
// backfill doesn't loop on ErrParentHashMismatch.
func TestColdStartBackfill_RecoversFromOfflineReorg(t *testing.T) {
	const (
		act    uint64 = 100
		safeTs uint64 = 110
	)
	depth := 20 * time.Second

	h := newInteropTestHarness(t).
		WithActivation(act).
		WithLogBackfillDepth(depth).
		WithChain(10, func(m *mockChainContainer) {
			m.firstSafeHeadTimestamp = safeTs
			m.firstSafeHeadTimestampSet = true
		}).
		Build()
	h.interop.initialized.Store(false)
	h.interop.verificationStartTimestamp = 0

	db := h.interop.logsDBs[eth.ChainIDFromUInt64(10)]
	v1Hash := func(n uint64) common.Hash {
		return common.BigToHash(new(big.Int).SetUint64(n | 0xdead0000))
	}
	canonicalHash := func(n uint64) common.Hash {
		return common.BigToHash(new(big.Int).SetUint64(n))
	}
	require.NoError(t, db.SealBlock(common.Hash{},
		eth.BlockID{Number: act - 1, Hash: v1Hash(act - 1)}, act-1))
	require.NoError(t, db.SealBlock(v1Hash(act-1),
		eth.BlockID{Number: act, Hash: v1Hash(act)}, act))
	for n := act + 1; n <= 105; n++ {
		require.NoError(t, db.SealBlock(v1Hash(n-1),
			eth.BlockID{Number: n, Hash: v1Hash(n)}, n))
	}

	advanced, err := h.interop.advanceColdStartInit()
	require.NoError(t, err)
	require.True(t, advanced)

	latest, has := db.LatestSealedBlock()
	require.True(t, has)
	require.Equal(t, safeTs-1, latest.Number,
		"backfill seals up to verificationStart-1 after recovering from the reorg")
	for n := act; n < safeTs; n++ {
		seal, err := db.FindSealedBlock(n)
		require.NoError(t, err, "block %d must be sealed", n)
		require.Equal(t, canonicalHash(n), seal.Hash,
			"block %d must hold the canonical hash, not the stale v1 hash", n)
	}
}

// TestColdStartBackfill_LeavesAheadLogsDBUnchanged: when a partial prior run
// left the logsDB tip past endTime and the tail is still canonical, reconcile
// is a no-op and backfill's startNum > endNum short-circuit leaves it alone.
func TestColdStartBackfill_LeavesAheadLogsDBUnchanged(t *testing.T) {
	const (
		act        uint64 = 100
		safeTs     uint64 = 110
		preSeedTip uint64 = 120
	)
	depth := 60 * time.Second

	h := newInteropTestHarness(t).
		WithActivation(act).
		WithLogBackfillDepth(depth).
		WithChain(10, func(m *mockChainContainer) {
			m.firstSafeHeadTimestamp = safeTs
			m.firstSafeHeadTimestampSet = true
		}).
		Build()
	h.interop.initialized.Store(false)
	h.interop.verificationStartTimestamp = 0

	db := h.interop.logsDBs[eth.ChainIDFromUInt64(10)]
	canonicalHash := func(n uint64) common.Hash {
		return common.BigToHash(new(big.Int).SetUint64(n))
	}
	require.NoError(t, db.SealBlock(common.Hash{},
		eth.BlockID{Number: act - 1, Hash: canonicalHash(act - 1)}, act-1))
	require.NoError(t, db.SealBlock(canonicalHash(act-1),
		eth.BlockID{Number: act, Hash: canonicalHash(act)}, act))
	for n := act + 1; n <= preSeedTip; n++ {
		require.NoError(t, db.SealBlock(canonicalHash(n-1),
			eth.BlockID{Number: n, Hash: canonicalHash(n)}, n))
	}

	advanced, err := h.interop.advanceColdStartInit()
	require.NoError(t, err)
	require.True(t, advanced)

	latest, has := db.LatestSealedBlock()
	require.True(t, has)
	require.Equal(t, preSeedTip, latest.Number,
		"logsDB tip must be unchanged when already past endTime and canonical")
	require.Equal(t, canonicalHash(preSeedTip), latest.Hash)
}

// TestColdStartBackfill_TrimsNonCanonicalAheadLogsDBAndCatchesUp: when a prior
// partial run left the logsDB ahead of endTime but the tail diverged from
// canonical, reconcile must rewind to the last canonical block and backfill
// must then catch up to endTime.
func TestColdStartBackfill_TrimsNonCanonicalAheadLogsDBAndCatchesUp(t *testing.T) {
	const (
		act          uint64 = 100
		safeTs       uint64 = 110
		lastCanonNum uint64 = 108
		preSeedTip   uint64 = 120
	)
	depth := 60 * time.Second

	h := newInteropTestHarness(t).
		WithActivation(act).
		WithLogBackfillDepth(depth).
		WithChain(10, func(m *mockChainContainer) {
			m.firstSafeHeadTimestamp = safeTs
			m.firstSafeHeadTimestampSet = true
		}).
		Build()
	h.interop.initialized.Store(false)
	h.interop.verificationStartTimestamp = 0

	db := h.interop.logsDBs[eth.ChainIDFromUInt64(10)]
	canonicalHash := func(n uint64) common.Hash {
		return common.BigToHash(new(big.Int).SetUint64(n))
	}
	v1Hash := func(n uint64) common.Hash {
		return common.BigToHash(new(big.Int).SetUint64(n | 0xdead0000))
	}
	require.NoError(t, db.SealBlock(common.Hash{},
		eth.BlockID{Number: act - 1, Hash: canonicalHash(act - 1)}, act-1))
	require.NoError(t, db.SealBlock(canonicalHash(act-1),
		eth.BlockID{Number: act, Hash: canonicalHash(act)}, act))
	for n := act + 1; n <= lastCanonNum; n++ {
		require.NoError(t, db.SealBlock(canonicalHash(n-1),
			eth.BlockID{Number: n, Hash: canonicalHash(n)}, n))
	}
	require.NoError(t, db.SealBlock(canonicalHash(lastCanonNum),
		eth.BlockID{Number: lastCanonNum + 1, Hash: v1Hash(lastCanonNum + 1)}, lastCanonNum+1))
	for n := lastCanonNum + 2; n <= preSeedTip; n++ {
		require.NoError(t, db.SealBlock(v1Hash(n-1),
			eth.BlockID{Number: n, Hash: v1Hash(n)}, n))
	}

	advanced, err := h.interop.advanceColdStartInit()
	require.NoError(t, err)
	require.True(t, advanced)

	latest, has := db.LatestSealedBlock()
	require.True(t, has)
	require.Equal(t, safeTs-1, latest.Number,
		"after reconcile + seal, logsDB tip must equal endTime")
	for n := act; n < safeTs; n++ {
		seal, err := db.FindSealedBlock(n)
		require.NoError(t, err, "block %d must remain sealed", n)
		require.Equal(t, canonicalHash(n), seal.Hash,
			"block %d must hold the canonical hash after reconcile", n)
	}
}

// TestFirstVerifiableTimestamp_PrefersVerifiedDB locks the contract that
// verifiedDB.FirstTimestamp takes precedence over any later
// verificationStartTimestamp set by init.
func TestFirstVerifiableTimestamp_PrefersVerifiedDB(t *testing.T) {

	dataDir := t.TempDir()
	db, err := OpenVerifiedDB(dataDir)
	require.NoError(t, err)
	require.NoError(t, db.Commit(VerifiedResult{
		Timestamp:   200,
		L1Inclusion: eth.BlockID{Number: 1},
		L2Heads:     map[eth.ChainID]eth.BlockID{eth.ChainIDFromUInt64(10): {Number: 20}},
	}))
	require.NoError(t, db.Close())

	interop := New(testLogger(), 100, 0, nil, dataDir, nil, 0, nil)
	require.NotNil(t, interop)
	defer func() { require.NoError(t, interop.Stop(context.Background())) }()

	// Resume picks verificationStart=201, but RPC accessor returns 200
	// (the first committed timestamp) for the firstVerifiable boundary.
	interop.tryInitFromVerifiedDB()
	require.Equal(t, uint64(201), interop.verificationStartTimestamp)

	got, err := interop.firstVerifiableTimestamp()
	require.NoError(t, err)
	require.Equal(t, uint64(200), got)
}

// TestFirstVerifiableTimestamp_ErrNotStartedBeforeInit confirms RPC accessors
// return ErrNotStarted while cold-start init is in progress.
func TestFirstVerifiableTimestamp_ErrNotStartedBeforeInit(t *testing.T) {

	dataDir := t.TempDir()
	interop := New(testLogger(), 1000, 0, nil, dataDir, nil, 0, nil)
	require.NotNil(t, interop)
	defer func() { require.NoError(t, interop.Stop(context.Background())) }()

	_, err := interop.firstVerifiableTimestamp()
	require.ErrorIs(t, err, ErrNotStarted)
}

// _ ensures the cc import is retained even if helpers shift.
var _ = cc.ErrSafeDBEmpty
