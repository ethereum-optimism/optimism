package interop

import (
	"context"
	"errors"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// progressInteropUntil calls progressAndRecord up to maxIters times until cond() is true.
func progressInteropUntil(t *testing.T, i *Interop, maxIters int, cond func() bool) {
	t.Helper()
	for range maxIters {
		if cond() {
			return
		}
		_, err := i.progressAndRecord()
		require.NoError(t, err)
	}
}

func requireFirstVerifiableTimestamp(t *testing.T, i *Interop, want uint64, msgAndArgs ...interface{}) {
	t.Helper()
	got, err := i.firstVerifiableTimestamp(context.Background())
	require.NoError(t, err)
	require.Equal(t, want, got, msgAndArgs...)
}

func TestLogBackfill_ResumesAfterInterruption(t *testing.T) {
	const act = uint64(100)
	depth := 10 * time.Second // EL finalized 110, depth 10s -> T_lo 100; should seal 100..110

	h := newInteropTestHarness(t).
		WithActivation(act).
		WithLogBackfillDepth(depth).
		WithChain(10, func(m *mockChainContainer) {
			m.currentL1 = eth.BlockRef{Number: 1, Hash: common.HexToHash("0xL1")}
			m.syncStatusFull = &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 1, Hash: common.HexToHash("0xL1")},
				UnsafeL2:    eth.L2BlockRef{Number: 110, Time: 110},
				SafeL2:      eth.L2BlockRef{Number: 110, Time: 110},
				LocalSafeL2: eth.L2BlockRef{Number: 110, Time: 110},
			}
		}).
		Build()
	h.interop.ctx = context.Background()

	// Simulate a previous partial run: seal blocks 100..105 into the logsDB.
	chain10 := h.Mock(10)
	for num := uint64(100); num <= 105; num++ {
		out, err := chain10.OutputV0AtBlockNumber(context.Background(), num)
		require.NoError(t, err)
		bid := eth.BlockID{Hash: out.BlockHash, Number: num}
		blockInfo, receipts, err := chain10.FetchReceipts(context.Background(), bid)
		require.NoError(t, err)
		err = h.interop.sealBlockDataIntoLogsDB(chain10.id, bid, blockInfo, receipts, blockInfo.Time(), true)
		require.NoError(t, err)
	}

	latest, has := h.interop.logsDBs[chain10.id].LatestSealedBlock()
	require.True(t, has)
	require.Equal(t, uint64(105), latest.Number)

	// Track how many OutputV0 calls happen during backfill to confirm we
	// don't re-fetch blocks 100..105.
	var fetchCount atomic.Int32
	chain10.outputV0Override = func(ctx context.Context, num uint64) (*eth.OutputV0, error) {
		fetchCount.Add(1)
		return &eth.OutputV0{
			StateRoot:                eth.Bytes32(common.HexToHash("0xmockstate")),
			MessagePasserStorageRoot: eth.Bytes32(common.HexToHash("0xmockmsg")),
			BlockHash:                common.BigToHash(new(big.Int).SetUint64(num)),
		}, nil
	}

	end, err := h.interop.runLogBackfill()
	require.NoError(t, err)
	h.interop.backfillEndTimestamp = end
	require.Equal(t, uint64(110), end,
		"runLogBackfill must return minELFinalizedTime as the end of the sealed range")
	requireFirstVerifiableTimestamp(t, h.interop, 111,
		"main loop resumes at backfillEndTimestamp+1")
	require.Equal(t, act, h.interop.activationTimestamp, "protocol activation must not change")

	latest, has = h.interop.logsDBs[chain10.id].LatestSealedBlock()
	require.True(t, has)
	require.Equal(t, uint64(110), latest.Number)

	// 5 fetches for blocks 106..110 + 1 reconcile probe at block 105.
	require.Equal(t, int32(6), fetchCount.Load())
}

func TestLogBackfill_RetriesWhenELFinalizedNotReady(t *testing.T) {
	const act = uint64(100)
	depth := 10 * time.Second

	// Track EL finalized head call count so we can make the first N calls fail.
	var elFinalizedCalls atomic.Int32
	failUntil := int32(3) // first 3 calls return error, then succeed

	h := newInteropTestHarness(t).
		WithActivation(act).
		WithLogBackfillDepth(depth).
		WithChain(10, func(m *mockChainContainer) {
			m.currentL1 = eth.BlockRef{Number: 1, Hash: common.HexToHash("0xL1")}
			m.syncStatusFull = &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 1, Hash: common.HexToHash("0xL1")},
				UnsafeL2:    eth.L2BlockRef{Number: 110, Time: 110},
				SafeL2:      eth.L2BlockRef{Number: 110, Time: 110},
				LocalSafeL2: eth.L2BlockRef{Number: 110, Time: 110},
			}
			m.elFinalizedHeadOverride = func() (eth.L2BlockRef, error) {
				n := elFinalizedCalls.Add(1)
				if n <= failUntil {
					return eth.L2BlockRef{}, errors.New("EL finalized not ready")
				}
				return eth.L2BlockRef{Number: 110, Time: 110}, nil
			}
		}).
		Build()

	// Use a shorter backoff for tests.
	origBackoff := errorBackoffPeriod
	errorBackoffPeriod = 10 * time.Millisecond
	t.Cleanup(func() { errorBackoffPeriod = origBackoff })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- h.interop.Start(ctx) }()

	// Wait for backfill to complete: backfillEndTimestamp should be set
	// to the end of the sealed range (110).
	require.Eventually(t, func() bool {
		return h.interop.backfillEndTimestamp > 0
	}, 5*time.Second, 20*time.Millisecond, "backfill should eventually succeed after retries")

	require.GreaterOrEqual(t, elFinalizedCalls.Load(), failUntil,
		"EL finalized head should have been called at least %d times (the failing ones)", failUntil)
	require.Equal(t, uint64(110), h.interop.backfillEndTimestamp)
	requireFirstVerifiableTimestamp(t, h.interop, 111)
	require.Equal(t, act, h.interop.activationTimestamp, "protocol activation must not change")

	cancel()
	<-done
}

// TestLogBackfill_RecoversFromOfflineReorg tests an L2 reorg that
// invalidates a sealed block while supernode is offline self-heals on
// restart, not loop forever on ErrParentHashMismatch.
func TestLogBackfill_RecoversFromOfflineReorg(t *testing.T) {
	const act = uint64(100)
	depth := 20 * time.Second

	h := newInteropTestHarness(t).
		WithActivation(act).
		WithLogBackfillDepth(depth).
		WithChain(10, func(m *mockChainContainer) {
			m.currentL1 = eth.BlockRef{Number: 1, Hash: common.HexToHash("0xL1")}
			m.syncStatusFull = &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 1, Hash: common.HexToHash("0xL1")},
				UnsafeL2:    eth.L2BlockRef{Number: 110, Time: 110},
				SafeL2:      eth.L2BlockRef{Number: 110, Time: 110},
				LocalSafeL2: eth.L2BlockRef{Number: 110, Time: 110},
			}
		}).
		Build()

	chain10 := h.Mock(10)
	db := h.interop.logsDBs[chain10.id]

	// Pre-seed blocks 100..105 with a stale "v1" fork hash; the mock's canonical
	// view ("v2") returns BigToHash(n).
	v1Hash := func(n uint64) common.Hash {
		return common.BigToHash(new(big.Int).SetUint64(n | 0xdead0000))
	}
	require.NoError(t, db.SealBlock(common.Hash{},
		eth.BlockID{Number: 100, Hash: v1Hash(100)}, 100))
	for n := uint64(101); n <= 105; n++ {
		require.NoError(t, db.SealBlock(v1Hash(n-1),
			eth.BlockID{Number: n, Hash: v1Hash(n)}, n))
	}
	before, has := db.LatestSealedBlock()
	require.True(t, has)
	require.Equal(t, uint64(105), before.Number)
	require.Equal(t, v1Hash(105), before.Hash)

	origBackoff := errorBackoffPeriod
	errorBackoffPeriod = 10 * time.Millisecond
	t.Cleanup(func() { errorBackoffPeriod = origBackoff })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- h.interop.Start(ctx) }()

	require.Eventually(t, func() bool {
		return h.interop.backfillCompleted.Load()
	}, 15*time.Second, 20*time.Millisecond,
		"Start must recover from an offline reorg, not loop forever on ErrParentHashMismatch")

	require.Equal(t, uint64(110), h.interop.backfillEndTimestamp)

	// Assert specific heights, not LatestSealedBlock: once backfill completes
	// the main loop may seal further blocks before we read state.
	canonicalHash := func(n uint64) common.Hash {
		return common.BigToHash(new(big.Int).SetUint64(n))
	}
	seal110, err := db.FindSealedBlock(110)
	require.NoError(t, err, "backfill tip must be sealed")
	require.Equal(t, canonicalHash(110), seal110.Hash,
		"backfill tip must hold the canonical hash, not a stale v1 hash")
	seal103, err := db.FindSealedBlock(103)
	require.NoError(t, err)
	require.Equal(t, canonicalHash(103), seal103.Hash,
		"reorged interior block must be replaced with the canonical hash")
	seal100, err := db.FindSealedBlock(100)
	require.NoError(t, err)
	require.Equal(t, canonicalHash(100), seal100.Hash,
		"activation block must be replaced with the canonical hash")

	cancel()
	<-done
}

func TestLogBackfill_RetriesStopOnContextCancel(t *testing.T) {
	const act = uint64(100)
	depth := 10 * time.Second

	h := newInteropTestHarness(t).
		WithActivation(act).
		WithLogBackfillDepth(depth).
		WithChain(10, func(m *mockChainContainer) {
			// SyncStatus always fails — backfill will retry forever.
			m.currentL1Err = errors.New("virtual node not ready")
		}).
		Build()

	origBackoff := errorBackoffPeriod
	errorBackoffPeriod = 10 * time.Millisecond
	t.Cleanup(func() { errorBackoffPeriod = origBackoff })

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- h.interop.Start(ctx) }()

	// Let it retry a few times, then cancel.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

// TestLogBackfill_AsymmetricMultiChain asserts that every chain is backfilled
// over the same [T_lo, minELFinalizedTime] window regardless of how far
// individual chains' EL finalized heads have advanced. This keeps the system
// symmetric at startup — every chain has logs sealed up to the same
// timestamp, matching the invariant the main loop observes during normal
// operation. Chains whose EL finalized head is beyond minELFinalizedTime catch
// up through the main loop, not eagerly during backfill.
//
//   - T_lo is derived from min(EL finalized head time) across chains.
//   - End of backfill for every chain is TimestampToBlockNumber(minELFinalizedTime).
//   - backfillEndTimestamp is set to minELFinalizedTime; the main loop
//     resumes at backfillEndTimestamp+1.
func TestLogBackfill_AsymmetricMultiChain(t *testing.T) {
	const act = uint64(50)
	depth := 10 * time.Second // min EL finalized 110 -> T_lo 100

	// Chain 10: EL finalized tip at 120.
	// Chain 20: EL finalized tip at 130.
	// Chain 30: EL finalized 110 (the min, pinning T_lo).
	// Every chain backfills 100..110 (the shared shape), so each seals 11
	// blocks regardless of how far its EL finalized tip is.
	h := newInteropTestHarness(t).
		WithActivation(act).
		WithLogBackfillDepth(depth).
		WithChain(10, func(m *mockChainContainer) {
			m.elFinalizedHead = eth.L2BlockRef{Number: 120, Time: 120}
			m.elFinalizedHeadSet = true
			m.syncStatusFull = &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 1, Hash: common.HexToHash("0xL1")},
				UnsafeL2:    eth.L2BlockRef{Number: 120, Time: 120},
				SafeL2:      eth.L2BlockRef{Number: 120, Time: 120},
				LocalSafeL2: eth.L2BlockRef{Number: 120, Time: 120},
			}
		}).
		WithChain(20, func(m *mockChainContainer) {
			m.elFinalizedHead = eth.L2BlockRef{Number: 130, Time: 130}
			m.elFinalizedHeadSet = true
			m.syncStatusFull = &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 1, Hash: common.HexToHash("0xL1")},
				UnsafeL2:    eth.L2BlockRef{Number: 130, Time: 130},
				SafeL2:      eth.L2BlockRef{Number: 200, Time: 200},
				LocalSafeL2: eth.L2BlockRef{Number: 130, Time: 130},
			}
		}).
		WithChain(30, func(m *mockChainContainer) {
			m.elFinalizedHead = eth.L2BlockRef{Number: 110, Time: 110}
			m.elFinalizedHeadSet = true
			m.syncStatusFull = &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 1, Hash: common.HexToHash("0xL1")},
				UnsafeL2:    eth.L2BlockRef{Number: 110, Time: 110},
				SafeL2:      eth.L2BlockRef{Number: 100, Time: 100},
				LocalSafeL2: eth.L2BlockRef{Number: 110, Time: 110},
			}
		}).
		Build()
	h.interop.ctx = context.Background()

	fetchCount := make(map[eth.ChainID]*atomic.Int32, 3)
	for _, id := range []uint64{10, 20, 30} {
		c := h.Mock(id)
		counter := new(atomic.Int32)
		fetchCount[c.id] = counter
		c.outputV0Override = func(ctx context.Context, num uint64) (*eth.OutputV0, error) {
			counter.Add(1)
			return &eth.OutputV0{
				StateRoot:                eth.Bytes32(common.HexToHash("0xmockstate")),
				MessagePasserStorageRoot: eth.Bytes32(common.HexToHash("0xmockmsg")),
				BlockHash:                common.BigToHash(new(big.Int).SetUint64(num)),
			}, nil
		}
	}

	end, err := h.interop.runLogBackfill()
	require.NoError(t, err)
	h.interop.backfillEndTimestamp = end
	require.Equal(t, act, h.interop.activationTimestamp, "protocol activation must not change")
	require.Equal(t, uint64(110), end,
		"runLogBackfill must return minELFinalizedTime as the end of the sealed range")
	requireFirstVerifiableTimestamp(t, h.interop, 111,
		"main loop resumes at backfillEndTimestamp+1")

	chain10 := h.Mock(10)
	chain20 := h.Mock(20)
	chain30 := h.Mock(30)

	// Every chain backfills the same 100..110 window (11 blocks each).
	require.Equal(t, int32(11), fetchCount[chain10.id].Load(),
		"chain 10 should backfill blocks 100..110 (11 blocks)")
	require.Equal(t, int32(11), fetchCount[chain20.id].Load(),
		"chain 20 should backfill blocks 100..110 (11 blocks)")
	require.Equal(t, int32(11), fetchCount[chain30.id].Load(),
		"chain 30 should backfill blocks 100..110 (11 blocks)")

	latest10, has10 := h.interop.logsDBs[chain10.id].LatestSealedBlock()
	require.True(t, has10)
	require.Equal(t, uint64(110), latest10.Number)

	latest20, has20 := h.interop.logsDBs[chain20.id].LatestSealedBlock()
	require.True(t, has20)
	require.Equal(t, uint64(110), latest20.Number)

	latest30, has30 := h.interop.logsDBs[chain30.id].LatestSealedBlock()
	require.True(t, has30)
	require.Equal(t, uint64(110), latest30.Number)
}

// TestLogBackfill_MisalignedActivation asserts that backfill succeeds when
// the protocol activation timestamp does not land on a (genesis + k*blockTime)
// boundary. In this configuration TargetBlockNumber(activation) floors to the
// last block whose Time() is strictly before activation: that block
// represents the chain state as of the fork and is the correct pairing
// anchor for the first post-activation block. An overly strict
// "first seal must be >= activation" check would reject this block and the
// retry loop would spin forever with a misleading "virtual nodes may not be
// ready" log line.
//
// Concrete setup: blockTime=3, genesis=0, activation=1000. Block 333 has
// Time()=999 (the pairing anchor); block 334 is at 1002; LocalSafe is at
// block 340, Time=1020. T_lo clamps to activation so backfill must seal
// blocks 333..340 without error. backfillEndTimestamp is set to 1020
// (minELFinalizedTime), so the main loop resumes at 1021.
func TestLogBackfill_MisalignedActivation(t *testing.T) {
	const (
		blockTime    uint64 = 3
		act          uint64 = 1000
		localSafeNum uint64 = 340
		localSafeTs  uint64 = 1020 // 340 * blockTime
	)
	depth := 60 * time.Second // EL finalized 1020 - 60 = 960 < activation → T_lo clamps to 1000

	blockNumToTime := func(num uint64) uint64 { return num * blockTime }
	tsToBlockNum := func(ctx context.Context, ts uint64) (uint64, error) {
		return ts / blockTime, nil // floor, matches rollup.TargetBlockNumber
	}

	h := newInteropTestHarness(t).
		WithActivation(act).
		WithLogBackfillDepth(depth).
		WithChain(10, func(m *mockChainContainer) {
			m.blockTimeOverride = blockTime
			m.blockInfoTimeFn = blockNumToTime
			m.timestampToBlockNumberOverride = tsToBlockNum
			m.syncStatusFull = &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 1, Hash: common.HexToHash("0xL1")},
				UnsafeL2:    eth.L2BlockRef{Number: localSafeNum, Time: localSafeTs},
				SafeL2:      eth.L2BlockRef{Number: localSafeNum, Time: localSafeTs},
				LocalSafeL2: eth.L2BlockRef{Number: localSafeNum, Time: localSafeTs},
			}
		}).
		Build()
	h.interop.ctx = context.Background()

	end, err := h.interop.runLogBackfill()
	require.NoError(t, err)
	h.interop.backfillEndTimestamp = end
	require.Equal(t, act, h.interop.activationTimestamp, "protocol activation must not change")
	require.Equal(t, localSafeTs, end,
		"runLogBackfill must return minELFinalizedTime as the end of the sealed range")
	requireFirstVerifiableTimestamp(t, h.interop, localSafeTs+1)

	chain10 := h.Mock(10)
	db := h.interop.logsDBs[chain10.id]

	// processBlockLogs seals a "virtual parent" before the first real backfill
	// block so subsequent blocks have a parent to link against. For the first
	// backfill block at number 333, that virtual parent is at number 332 with
	// the real block's Time() (999). FirstSealedBlock therefore returns the
	// virtual parent — the anchor — and the invariant we care about is that
	// its timestamp is strictly pre-activation but within one blockTime of it.
	first, err := db.FirstSealedBlock()
	require.NoError(t, err)
	require.Equal(t, uint64(332), first.Number, "first sealed block is the virtual parent of TargetBlockNumber(activation)")
	require.Equal(t, uint64(999), first.Timestamp,
		"anchor's Time() is the real block's time, strictly pre-activation — this is the pairing anchor, not a violation")
	require.Less(t, first.Timestamp, act, "sanity: anchor is strictly pre-activation")
	require.Greater(t, first.Timestamp+blockTime, act,
		"anchor must still be within one blockTime of activation (the anchor window)")

	latest, has := db.LatestSealedBlock()
	require.True(t, has)
	require.Equal(t, localSafeNum, latest.Number)
}

func TestLogBackfill_AdvancesActivationAndStartsVerifyAfterCeiling(t *testing.T) {
	const act = uint64(108)
	depth := time.Second // EL finalized 110, depth 1s -> T_lo 109; seals 109..110; first verifiable ts = 111

	h := newInteropTestHarness(t).
		WithActivation(act).
		WithLogBackfillDepth(depth).
		WithChain(10, func(m *mockChainContainer) {
			m.currentL1 = eth.BlockRef{Number: 1, Hash: common.HexToHash("0xL1")}
			m.syncStatusFull = &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 1, Hash: common.HexToHash("0xL1")},
				UnsafeL2:    eth.L2BlockRef{Number: 110, Time: 110},
				SafeL2:      eth.L2BlockRef{Number: 110, Time: 110},
				LocalSafeL2: eth.L2BlockRef{Number: 110, Time: 110},
			}
		}).
		Build()

	var verifyCalls atomic.Int32
	var firstVerifyTS atomic.Uint64
	h.interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID, _ map[eth.ChainID]eth.BlockID, _ *frontierVerificationView) (Result, error) {
		if verifyCalls.Add(1) == 1 {
			firstVerifyTS.Store(ts)
		}
		return Result{
			Timestamp:   ts,
			L1Inclusion: eth.BlockID{Number: 1, Hash: common.HexToHash("0xL1")},
			L2Heads:     blocks,
		}, nil
	}
	h.interop.ctx = context.Background()

	end, err := h.interop.runLogBackfill()
	require.NoError(t, err)
	h.interop.backfillEndTimestamp = end
	require.Equal(t, uint64(110), end,
		"runLogBackfill must return minELFinalizedTime as the end of the sealed range")
	requireFirstVerifiableTimestamp(t, h.interop, 111,
		"main loop resumes at backfillEndTimestamp+1")
	require.Equal(t, act, h.interop.activationTimestamp, "protocol activation must not change")

	chain10 := h.Mock(10)
	latest, has := h.interop.logsDBs[chain10.id].LatestSealedBlock()
	require.True(t, has)
	require.Equal(t, uint64(110), latest.Number)
	require.Zero(t, verifyCalls.Load())

	// Progress the main loop — first verify should be at 111 (activation after backfill).
	progressInteropUntil(t, h.interop, 10, func() bool {
		lastTS, ok := h.interop.verifiedDB.LastTimestamp()
		return ok && lastTS >= 111
	})
	lastTS, ok := h.interop.verifiedDB.LastTimestamp()
	require.True(t, ok)
	require.GreaterOrEqual(t, lastTS, uint64(111))
	require.Equal(t, int32(1), verifyCalls.Load())
	require.Equal(t, uint64(111), firstVerifyTS.Load())
}

// TestLogBackfill_NoOpWhenDepthZero asserts that runLogBackfill short-circuits
// when logBackfillDepth is zero: it must return (0, nil) without touching
// SyncStatus, TimestampToBlockNumber, or the logs DB. This is the "feature
// disabled" path — operators who don't want backfill get no work done.
func TestLogBackfill_NoOpWhenDepthZero(t *testing.T) {
	const act = uint64(100)

	var syncStatusCalls atomic.Int32
	var outputCalls atomic.Int32

	h := newInteropTestHarness(t).
		WithActivation(act).
		// no WithLogBackfillDepth → depth stays at zero-value.
		WithChain(10, func(m *mockChainContainer) {
			m.elFinalizedHead = eth.L2BlockRef{Number: 110, Time: 110}
			m.elFinalizedHeadSet = true
			m.syncStatusOverride = func() (*eth.SyncStatus, error) {
				syncStatusCalls.Add(1)
				return &eth.SyncStatus{
					CurrentL1:   eth.L1BlockRef{Number: 1, Hash: common.HexToHash("0xL1")},
					UnsafeL2:    eth.L2BlockRef{Number: 110, Time: 110},
					SafeL2:      eth.L2BlockRef{Number: 110, Time: 110},
					LocalSafeL2: eth.L2BlockRef{Number: 110, Time: 110},
				}, nil
			}
			m.outputV0Override = func(ctx context.Context, num uint64) (*eth.OutputV0, error) {
				outputCalls.Add(1)
				return &eth.OutputV0{
					StateRoot:                eth.Bytes32(common.HexToHash("0xmockstate")),
					MessagePasserStorageRoot: eth.Bytes32(common.HexToHash("0xmockmsg")),
					BlockHash:                common.BigToHash(new(big.Int).SetUint64(num)),
				}, nil
			}
		}).
		Build()
	h.interop.ctx = context.Background()

	end, err := h.interop.runLogBackfill()
	require.NoError(t, err)
	require.Zero(t, end, "depth==0 short-circuits with end=0")
	require.Zero(t, syncStatusCalls.Load(), "SyncStatus must not be called when depth is zero")
	require.Zero(t, outputCalls.Load(), "no blocks should be fetched when depth is zero")

	chain10 := h.Mock(10)
	_, has := h.interop.logsDBs[chain10.id].LatestSealedBlock()
	require.False(t, has, "logs DB must remain empty")

	// Caller sets backfillEndTimestamp; with end==0 the main loop derives the
	// first unverified timestamp from the current EL finalized head.
	h.interop.backfillEndTimestamp = end
	requireFirstVerifiableTimestamp(t, h.interop, 111,
		"with end==0 the main loop starts after the finalized head")
}

// TestLogBackfill_NoOpWhenNoChains asserts that runLogBackfill short-circuits
// when no chains are registered: no SyncStatus/TimestampToBlockNumber calls
// can happen because there's nothing to iterate, and end must be zero so the
// main loop falls back to activationTimestamp.
func TestLogBackfill_NoOpWhenNoChains(t *testing.T) {
	const act = uint64(100)

	h := newInteropTestHarness(t).
		WithActivation(act).
		WithLogBackfillDepth(10 * time.Second).
		// no WithChain calls — empty chains map.
		Build()
	require.NotNil(t, h.interop, "Interop must initialize with zero chains")
	h.interop.ctx = context.Background()

	end, err := h.interop.runLogBackfill()
	require.NoError(t, err)
	require.Zero(t, end, "empty chains map short-circuits with end=0")

	h.interop.backfillEndTimestamp = end
	requireFirstVerifiableTimestamp(t, h.interop, act,
		"with end==0 the main loop resumes at activationTimestamp")
}

// TestLogBackfill_ActivationInFuture asserts the edge case where the
// configured activation is ahead of every chain's EL finalized tip.
// firstVerifiableTimestamp clamps to activation, and backfill must no-op
// instead of sealing beyond the current EL finalized head.
func TestLogBackfill_ActivationInFuture(t *testing.T) {
	const act = uint64(2000)
	depth := 100 * time.Second

	var outputCalls atomic.Int32

	h := newInteropTestHarness(t).
		WithActivation(act).
		WithLogBackfillDepth(depth).
		WithChain(10, func(m *mockChainContainer) {
			// EL finalized tip at 1000 — well below activation 2000.
			m.syncStatusFull = &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 1, Hash: common.HexToHash("0xL1")},
				UnsafeL2:    eth.L2BlockRef{Number: 1000, Time: 1000},
				SafeL2:      eth.L2BlockRef{Number: 1000, Time: 1000},
				LocalSafeL2: eth.L2BlockRef{Number: 1000, Time: 1000},
			}
			m.outputV0Override = func(ctx context.Context, num uint64) (*eth.OutputV0, error) {
				outputCalls.Add(1)
				return &eth.OutputV0{
					StateRoot:                eth.Bytes32(common.HexToHash("0xmockstate")),
					MessagePasserStorageRoot: eth.Bytes32(common.HexToHash("0xmockmsg")),
					BlockHash:                common.BigToHash(new(big.Int).SetUint64(num)),
				}, nil
			}
		}).
		Build()
	h.interop.ctx = context.Background()

	end, err := h.interop.runLogBackfill()
	require.NoError(t, err)
	require.Zero(t, end)
	require.Zero(t, outputCalls.Load(),
		"no blocks fetched: backfill no-ops when activation is ahead of EL finalized")

	chain10 := h.Mock(10)
	_, has := h.interop.logsDBs[chain10.id].LatestSealedBlock()
	require.False(t, has, "logs DB must remain empty while backfill waits")

	require.Equal(t, act, h.interop.activationTimestamp,
		"protocol activation must not change")

	requireFirstVerifiableTimestamp(t, h.interop, act,
		"main loop resumes at activation when EL finalized is still pre-activation")
}

// TestLogBackfill_ClampsStartToGenesis asserts that the per-chain start is
// clamped up to the chain's genesis timestamp. Without this clamp,
// runLogBackfill would ask TimestampToBlockNumber for a pre-genesis timestamp
// and try to seal blocks before the chain existed. Both subcases assert the
// same shape: backfill seals exactly the per-chain range [genesisBlock, endBlock].
func TestLogBackfill_ClampsStartToGenesis(t *testing.T) {
	type genesisCase struct {
		name           string
		act            uint64
		depth          time.Duration
		genesisTime    uint64
		elFinalizedTip uint64
		// timestampToBlockNum maps a unix timestamp back to the block number the
		// chain would return. nil means use the harness default (identity).
		timestampToBlockNum func(ctx context.Context, ts uint64) (uint64, error)
		// blockNumberToTimestamp maps a block number to its unix timestamp. Only
		// block 0 (genesis) needs to differ from the identity default.
		blockNumberToTimestamp func(ctx context.Context, num uint64) (uint64, error)
		// blockInfoTime keeps FetchReceipts' reported block timestamp consistent
		// with blockNumberToTimestamp when they diverge from identity.
		blockInfoTime    func(num uint64) uint64
		wantEndBlock     uint64
		wantSealedBlocks int32
	}

	cases := []genesisCase{
		{
			// idealStart = 110-50 = 60; startTime = max(60, act=50) = 60.
			// Chain's genesis time is 100, which is > 60, so per-chain start
			// clamps to genesis and seals blocks 100..110 (11 blocks).
			name:           "activation before genesis",
			act:            50,
			depth:          50 * time.Second,
			genesisTime:    100,
			elFinalizedTip: 110,
			blockNumberToTimestamp: func(ctx context.Context, num uint64) (uint64, error) {
				if num == 0 {
					return 100, nil
				}
				return num, nil
			},
			wantEndBlock:     110,
			wantSealedBlocks: 11,
		},
		{
			// activation == genesis time. idealStart = 110-60 = 50;
			// startTime = max(50, act=100) = 100. genesisTime (100) is NOT
			// strictly greater than startTime (100), so the per-chain clamp
			// is a no-op and chainStartTime stays at activation=100.
			// TimestampToBlockNumber(100) returns block 0; the seal range is
			// blocks 0..10 (11 blocks) — the genesis block at the activation
			// boundary is included and has no logs, which is acceptable.
			name:           "activation equals genesis",
			act:            100,
			depth:          60 * time.Second,
			genesisTime:    100,
			elFinalizedTip: 110,
			timestampToBlockNum: func(ctx context.Context, ts uint64) (uint64, error) {
				return ts - 100, nil
			},
			blockNumberToTimestamp: func(ctx context.Context, num uint64) (uint64, error) {
				return num + 100, nil
			},
			blockInfoTime:    func(num uint64) uint64 { return num + 100 },
			wantEndBlock:     10,
			wantSealedBlocks: 11,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var outputCalls atomic.Int32

			h := newInteropTestHarness(t).
				WithActivation(tc.act).
				WithLogBackfillDepth(tc.depth).
				WithChain(10, func(m *mockChainContainer) {
					m.elFinalizedHead = eth.L2BlockRef{Number: tc.elFinalizedTip, Time: tc.elFinalizedTip}
					m.elFinalizedHeadSet = true
					m.syncStatusFull = &eth.SyncStatus{
						CurrentL1:   eth.L1BlockRef{Number: 1, Hash: common.HexToHash("0xL1")},
						UnsafeL2:    eth.L2BlockRef{Number: tc.elFinalizedTip, Time: tc.elFinalizedTip},
						SafeL2:      eth.L2BlockRef{Number: tc.elFinalizedTip, Time: tc.elFinalizedTip},
						LocalSafeL2: eth.L2BlockRef{Number: tc.elFinalizedTip, Time: tc.elFinalizedTip},
					}
					m.blockNumberToTimestampOverride = tc.blockNumberToTimestamp
					if tc.timestampToBlockNum != nil {
						m.timestampToBlockNumberOverride = tc.timestampToBlockNum
					}
					if tc.blockInfoTime != nil {
						m.blockInfoTimeFn = tc.blockInfoTime
					}
					m.outputV0Override = func(ctx context.Context, num uint64) (*eth.OutputV0, error) {
						outputCalls.Add(1)
						return &eth.OutputV0{
							StateRoot:                eth.Bytes32(common.HexToHash("0xmockstate")),
							MessagePasserStorageRoot: eth.Bytes32(common.HexToHash("0xmockmsg")),
							BlockHash:                common.BigToHash(new(big.Int).SetUint64(num)),
						}, nil
					}
				}).
				Build()
			h.interop.ctx = context.Background()

			end, err := h.interop.runLogBackfill()
			require.NoError(t, err)
			require.Equal(t, tc.elFinalizedTip, end,
				"return value is still minELFinalizedTime regardless of the genesis clamp")

			chain10 := h.Mock(10)
			latest, has := h.interop.logsDBs[chain10.id].LatestSealedBlock()
			require.True(t, has)
			require.Equal(t, tc.wantEndBlock, latest.Number)

			require.Equal(t, tc.wantSealedBlocks, outputCalls.Load(),
				"backfill must seal exactly [genesisBlock, endBlock]")
		})
	}
}

// TestLogBackfill_UsesVerifiedDBWhenInitializedAndSyncStatusStale simulates startup while
// StatusTracker still reports a stale SafeL2 block and local-safe has moved
// beyond the persisted EL finalized label. With an initialized verifiedDB,
// backfill should cap at verifiedDB.LastTimestamp instead of sampling moving
// SyncStatus state or extending past verifiedDB.
func TestLogBackfill_UsesVerifiedDBWhenInitializedAndSyncStatusStale(t *testing.T) {
	const (
		act           uint64 = 100
		staleCross    uint64 = 100
		elFinalized   uint64 = 200
		localSafe     uint64 = 200
		lastVerified  uint64 = 195
		backfillDepth        = 60 * time.Second
	)

	h := newInteropTestHarness(t).
		WithActivation(act).
		WithLogBackfillDepth(backfillDepth).
		WithChain(10, func(m *mockChainContainer) {
			m.elFinalizedHead = eth.L2BlockRef{Number: elFinalized, Time: elFinalized}
			m.elFinalizedHeadSet = true
			m.syncStatusFull = &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 1, Hash: common.HexToHash("0xL1")},
				UnsafeL2:    eth.L2BlockRef{Number: localSafe, Time: localSafe},
				SafeL2:      eth.L2BlockRef{Number: 0, Time: staleCross},
				LocalSafeL2: eth.L2BlockRef{Number: localSafe, Time: localSafe},
			}
		}).
		Build()
	h.interop.ctx = context.Background()

	chain10 := h.Mock(10)
	for ts := act + 1; ts <= lastVerified; ts++ {
		require.NoError(t, h.interop.verifiedDB.Commit(VerifiedResult{
			Timestamp:   ts,
			L1Inclusion: eth.BlockID{Number: 1, Hash: common.HexToHash("0xL1")},
			L2Heads:     map[eth.ChainID]eth.BlockID{chain10.id: {Number: ts, Hash: common.BigToHash(new(big.Int).SetUint64(ts))}},
		}))
	}

	end, err := h.interop.runLogBackfill()
	require.NoError(t, err)
	h.interop.backfillEndTimestamp = end

	require.Equal(t, lastVerified, end,
		"backfill must derive endTime from verifiedDB.LastTimestamp when verifiedDB is initialized")

	latest, has := h.interop.logsDBs[chain10.id].LatestSealedBlock()
	require.True(t, has)
	require.Equal(t, lastVerified, latest.Number)
}

// verifiedDB.LastTimestamp typically exceeds min EL finalized; backfill must
// still resume from verifiedDB.
func TestLogBackfill_UsesVerifiedDBWhenAheadOfELFinalized(t *testing.T) {
	const (
		act          uint64 = 100
		elFinalized  uint64 = 190
		lastVerified uint64 = 195
	)

	h := newInteropTestHarness(t).
		WithActivation(act).
		WithLogBackfillDepth(60*time.Second).
		WithChain(10, func(m *mockChainContainer) {
			m.elFinalizedHead = eth.L2BlockRef{Number: elFinalized, Time: elFinalized}
			m.elFinalizedHeadSet = true
			m.syncStatusFull = &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 1, Hash: common.HexToHash("0xL1")},
				UnsafeL2:    eth.L2BlockRef{Number: 200, Time: 200},
				SafeL2:      eth.L2BlockRef{Number: 0, Time: act},
				LocalSafeL2: eth.L2BlockRef{Number: 200, Time: 200},
			}
		}).
		Build()
	h.interop.ctx = context.Background()

	chain10 := h.Mock(10)
	for ts := act + 1; ts <= lastVerified; ts++ {
		require.NoError(t, h.interop.verifiedDB.Commit(VerifiedResult{
			Timestamp:   ts,
			L1Inclusion: eth.BlockID{Number: 1, Hash: common.HexToHash("0xL1")},
			L2Heads:     map[eth.ChainID]eth.BlockID{chain10.id: {Number: ts, Hash: common.BigToHash(new(big.Int).SetUint64(ts))}},
		}))
	}

	end, err := h.interop.runLogBackfill()
	require.NoError(t, err)
	require.Equal(t, lastVerified, end,
		"backfill end derives from verifiedDB.LastTimestamp regardless of EL finalized")

	latest, has := h.interop.logsDBs[chain10.id].LatestSealedBlock()
	require.True(t, has)
	require.Equal(t, lastVerified, latest.Number)
}

// TestLogBackfill_LeavesAheadLogsDBUnchanged asserts that when a chain's
// logsDB already holds canonical blocks past the computed backfill endTime,
// runLogBackfill does not rewrite, trim, or extend it. reconcileLogsDBTail
// sees the tip hash match canonical and returns without touching state; the
// seal loop then no-ops because startNum (latest+1) > endNum.
//
// Setup: cold start (empty verifiedDB), act=100, depth=60s,
// EL finalized=110 → endTime=110. Pre-seal blocks 100..120 with canonical
// hashes for chain 10. After backfill the logsDB tip must still be 120.
func TestLogBackfill_LeavesAheadLogsDBUnchanged(t *testing.T) {
	const (
		act         uint64 = 100
		elFinalized uint64 = 110
		preSeedTip  uint64 = 120
	)
	depth := 60 * time.Second

	h := newInteropTestHarness(t).
		WithActivation(act).
		WithLogBackfillDepth(depth).
		WithChain(10, func(m *mockChainContainer) {
			m.elFinalizedHead = eth.L2BlockRef{Number: elFinalized, Time: elFinalized}
			m.elFinalizedHeadSet = true
			m.syncStatusFull = &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 1, Hash: common.HexToHash("0xL1")},
				UnsafeL2:    eth.L2BlockRef{Number: preSeedTip, Time: preSeedTip},
				SafeL2:      eth.L2BlockRef{Number: preSeedTip, Time: preSeedTip},
				LocalSafeL2: eth.L2BlockRef{Number: preSeedTip, Time: preSeedTip},
			}
		}).
		Build()
	h.interop.ctx = context.Background()

	chain10 := h.Mock(10)
	db := h.interop.logsDBs[chain10.id]
	canonicalHash := func(n uint64) common.Hash {
		return common.BigToHash(new(big.Int).SetUint64(n))
	}
	require.NoError(t, db.SealBlock(common.Hash{},
		eth.BlockID{Number: act, Hash: canonicalHash(act)}, act))
	for n := act + 1; n <= preSeedTip; n++ {
		require.NoError(t, db.SealBlock(canonicalHash(n-1),
			eth.BlockID{Number: n, Hash: canonicalHash(n)}, n))
	}

	end, err := h.interop.runLogBackfill()
	require.NoError(t, err)
	require.Equal(t, elFinalized, end,
		"backfill end must equal minELFinalizedTime, independent of the ahead-of-end logsDB tip")

	latest, has := db.LatestSealedBlock()
	require.True(t, has)
	require.Equal(t, preSeedTip, latest.Number,
		"logsDB must be left untouched when it is already past endTime and canonical")
	require.Equal(t, canonicalHash(preSeedTip), latest.Hash,
		"logsDB tip hash must be unchanged")
}

// TestLogBackfill_TrimsNonCanonicalAheadLogsDBAndCatchesUp asserts the
// reconcile-then-backfill behavior when a chain's logsDB sits ahead of
// endTime but its tail has diverged from canonical (e.g. an L2 reorg landed
// while supernode was offline). reconcileLogsDBTail must walk back to the
// last canonical block, after which backfill seals forward up to endTime.
//
// Setup: cold start, act=100, depth=60s, EL finalized=110 → endTime=110.
// Pre-seal blocks 100..108 with canonical hashes, then 109..120 with a v1
// fork hash. Expect reconcile to rewind to 108, and the seal loop to seal
// 109..110 with canonical hashes. Final tip is endTime (110).
func TestLogBackfill_TrimsNonCanonicalAheadLogsDBAndCatchesUp(t *testing.T) {
	const (
		act          uint64 = 100
		elFinalized  uint64 = 110
		lastCanonNum uint64 = 108
		preSeedTip   uint64 = 120
	)
	depth := 60 * time.Second

	h := newInteropTestHarness(t).
		WithActivation(act).
		WithLogBackfillDepth(depth).
		WithChain(10, func(m *mockChainContainer) {
			m.elFinalizedHead = eth.L2BlockRef{Number: elFinalized, Time: elFinalized}
			m.elFinalizedHeadSet = true
			m.syncStatusFull = &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 1, Hash: common.HexToHash("0xL1")},
				UnsafeL2:    eth.L2BlockRef{Number: preSeedTip, Time: preSeedTip},
				SafeL2:      eth.L2BlockRef{Number: preSeedTip, Time: preSeedTip},
				LocalSafeL2: eth.L2BlockRef{Number: preSeedTip, Time: preSeedTip},
			}
		}).
		Build()
	h.interop.ctx = context.Background()

	chain10 := h.Mock(10)
	db := h.interop.logsDBs[chain10.id]
	canonicalHash := func(n uint64) common.Hash {
		return common.BigToHash(new(big.Int).SetUint64(n))
	}
	v1Hash := func(n uint64) common.Hash {
		return common.BigToHash(new(big.Int).SetUint64(n | 0xdead0000))
	}

	require.NoError(t, db.SealBlock(common.Hash{},
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

	end, err := h.interop.runLogBackfill()
	require.NoError(t, err)
	require.Equal(t, elFinalized, end,
		"backfill end must equal minELFinalizedTime, independent of the ahead-of-end logsDB tip")

	latest, has := db.LatestSealedBlock()
	require.True(t, has)
	require.Equal(t, elFinalized, latest.Number,
		"after reconcile + seal, logsDB tip must equal endTime")
	require.Equal(t, canonicalHash(elFinalized), latest.Hash,
		"final tip must hold the canonical hash, not a stale v1 hash")

	for n := act; n <= elFinalized; n++ {
		seal, err := db.FindSealedBlock(n)
		require.NoError(t, err, "block %d must remain sealed", n)
		require.Equal(t, canonicalHash(n), seal.Hash,
			"block %d must hold the canonical hash after reconcile", n)
	}
}
