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

func TestLogBackfillLowerBound(t *testing.T) {
	tests := []struct {
		name           string
		crossSafe, act uint64
		depth          time.Duration
		wantTlo        uint64
	}{
		{"zero depth returns crossSafe", 1000, 100, 0, 1000},
		{"clamp when raw before activation", 200, 100, 500 * time.Second, 100},
		{"raw above activation", 1000, 100, 100 * time.Second, 900},
		{"crossSafe below depth underflow then clamp", 50, 40, 100 * time.Second, 40},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LogBackfillLowerBound(tt.crossSafe, tt.act, tt.depth)
			require.Equal(t, tt.wantTlo, got)
		})
	}
}

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

func TestLogBackfill_ResumesAfterInterruption(t *testing.T) {
	const act = uint64(100)
	depth := 10 * time.Second // crossSafe 110, depth 10s -> T_lo 100; should seal 100..110

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

	require.NoError(t, h.interop.runLogBackfill())
	require.Equal(t, uint64(111), h.interop.runtimeActivationTimestamp)
	require.Equal(t, act, h.interop.activationTimestamp, "protocol activation must not change")

	latest, has = h.interop.logsDBs[chain10.id].LatestSealedBlock()
	require.True(t, has)
	require.Equal(t, uint64(110), latest.Number)

	// Should have fetched only blocks 106..110 (5 blocks), not 100..110 (11 blocks).
	require.Equal(t, int32(5), fetchCount.Load())
}

func TestLogBackfill_RetriesWhenVirtualNodesNotReady(t *testing.T) {
	const act = uint64(100)
	depth := 10 * time.Second

	// Track SyncStatus call count so we can make the first N calls fail.
	var syncStatusCalls atomic.Int32
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
			m.currentL1Err = errors.New("virtual node not ready")
			m.syncStatusOverride = func() (*eth.SyncStatus, error) {
				n := syncStatusCalls.Add(1)
				if n <= failUntil {
					return nil, errors.New("virtual node not ready")
				}
				return &eth.SyncStatus{
					CurrentL1:   eth.L1BlockRef{Number: 1, Hash: common.HexToHash("0xL1")},
					UnsafeL2:    eth.L2BlockRef{Number: 110, Time: 110},
					SafeL2:      eth.L2BlockRef{Number: 110, Time: 110},
					LocalSafeL2: eth.L2BlockRef{Number: 110, Time: 110},
				}, nil
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

	// Wait for backfill to complete: runtime activation should advance past 110.
	require.Eventually(t, func() bool {
		return h.interop.runtimeActivationTimestamp > act
	}, 5*time.Second, 20*time.Millisecond, "backfill should eventually succeed after retries")

	require.GreaterOrEqual(t, syncStatusCalls.Load(), failUntil,
		"SyncStatus should have been called at least %d times (the failing ones)", failUntil)
	require.Equal(t, uint64(111), h.interop.runtimeActivationTimestamp)
	require.Equal(t, act, h.interop.activationTimestamp, "protocol activation must not change")

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

// TestLogBackfill_AsymmetricMultiChain exercises the two cross-chain minima
// that runLogBackfill computes with three chains at different sync tips:
//
//   - T_lo is derived from min(SafeL2.Time) across chains (cross-safe).
//   - runtimeActivationTimestamp is derived from min(LocalSafeL2.Time) across
//     chains and must advance to that minimum + 1.
//
// All chains participate in backfill, and each chain's LocalSafe is at or
// beyond T_lo in both time and block number — the consistent case that
// runLogBackfill is required to support.
func TestLogBackfill_AsymmetricMultiChain(t *testing.T) {
	const act = uint64(50)
	depth := 10 * time.Second // min crossSafe 100 -> T_lo 90

	// Chain 10: crossSafe/localSafe tip at 120. Backfill range 90..120.
	// Chain 20: crossSafe 200, localSafe tip at 130. Backfill range 90..130.
	// Chain 30: crossSafe 100 (the min, so it pins T_lo), localSafe tip at
	// 110 (smallest localSafe, so it pins runtimeActivation). Backfill
	// range 90..110.
	h := newInteropTestHarness(t).
		WithActivation(act).
		WithLogBackfillDepth(depth).
		WithChain(10, func(m *mockChainContainer) {
			m.syncStatusFull = &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 1, Hash: common.HexToHash("0xL1")},
				UnsafeL2:    eth.L2BlockRef{Number: 120, Time: 120},
				SafeL2:      eth.L2BlockRef{Number: 120, Time: 120},
				LocalSafeL2: eth.L2BlockRef{Number: 120, Time: 120},
			}
		}).
		WithChain(20, func(m *mockChainContainer) {
			m.syncStatusFull = &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 1, Hash: common.HexToHash("0xL1")},
				UnsafeL2:    eth.L2BlockRef{Number: 130, Time: 130},
				SafeL2:      eth.L2BlockRef{Number: 200, Time: 200},
				LocalSafeL2: eth.L2BlockRef{Number: 130, Time: 130},
			}
		}).
		WithChain(30, func(m *mockChainContainer) {
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

	require.NoError(t, h.interop.runLogBackfill())
	require.Equal(t, act, h.interop.activationTimestamp, "protocol activation must not change")
	require.Equal(t, uint64(111), h.interop.runtimeActivationTimestamp,
		"runtime activation must advance to min(localSafe)+1 across all chains")

	chain10 := h.Mock(10)
	chain20 := h.Mock(20)
	chain30 := h.Mock(30)

	require.Equal(t, int32(31), fetchCount[chain10.id].Load(),
		"chain 10 should backfill blocks 90..120 (31 blocks)")
	require.Equal(t, int32(41), fetchCount[chain20.id].Load(),
		"chain 20 should backfill blocks 90..130 (41 blocks)")
	require.Equal(t, int32(21), fetchCount[chain30.id].Load(),
		"chain 30 should backfill blocks 90..110 (21 blocks)")

	latest10, has10 := h.interop.logsDBs[chain10.id].LatestSealedBlock()
	require.True(t, has10)
	require.Equal(t, uint64(120), latest10.Number)

	latest20, has20 := h.interop.logsDBs[chain20.id].LatestSealedBlock()
	require.True(t, has20)
	require.Equal(t, uint64(130), latest20.Number)

	latest30, has30 := h.interop.logsDBs[chain30.id].LatestSealedBlock()
	require.True(t, has30)
	require.Equal(t, uint64(110), latest30.Number)
}

// TestLogBackfill_FailsWhenChainBehindLowerBound asserts that runLogBackfill
// refuses to proceed if TimestampToBlockNumber(T_lo) for any chain lands past
// that chain's LocalSafeL2 block number. This condition is unreachable under
// a consistent SyncStatus (localSafe >= crossSafe >= minCrossSafeTime >= T_lo),
// so if it fires, the chain container is reporting inconsistent state.
// Silently skipping such a chain would leave its logs DB empty while other
// chains seal from T_lo forward, wedging the main loop's first seal attempt
// on ErrStaleLogsDB. Failing the attempt surfaces the problem to the
// operator; remediation is to clear the data dir and restart.
func TestLogBackfill_FailsWhenChainBehindLowerBound(t *testing.T) {
	const act = uint64(50)
	depth := 10 * time.Second // min crossSafe 100 -> T_lo 90

	h := newInteropTestHarness(t).
		WithActivation(act).
		WithLogBackfillDepth(depth).
		WithChain(10, func(m *mockChainContainer) {
			m.syncStatusFull = &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 1, Hash: common.HexToHash("0xL1")},
				UnsafeL2:    eth.L2BlockRef{Number: 120, Time: 120},
				SafeL2:      eth.L2BlockRef{Number: 100, Time: 100},
				LocalSafeL2: eth.L2BlockRef{Number: 120, Time: 120},
			}
		}).
		// Chain 20 reports a consistent-looking SyncStatus but a bogus
		// TimestampToBlockNumber that maps T_lo to a block well past
		// LocalSafeL2. This is the inconsistency the guard catches.
		WithChain(20, func(m *mockChainContainer) {
			m.syncStatusFull = &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 1, Hash: common.HexToHash("0xL1")},
				UnsafeL2:    eth.L2BlockRef{Number: 95, Time: 95},
				SafeL2:      eth.L2BlockRef{Number: 500, Time: 500},
				LocalSafeL2: eth.L2BlockRef{Number: 95, Time: 95},
			}
			m.timestampToBlockNumberOverride = func(ctx context.Context, ts uint64) (uint64, error) {
				return 200, nil
			}
		}).
		Build()
	h.interop.ctx = context.Background()

	err := h.interop.runLogBackfill()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrChainBehindBackfillLowerBound)
	require.Equal(t, act, h.interop.runtimeActivationTimestamp,
		"runtime activation must not advance when backfill fails")
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
// blocks 333..340 without error. runtimeActivation advances to 1021.
func TestLogBackfill_MisalignedActivation(t *testing.T) {
	const (
		blockTime    uint64 = 3
		act          uint64 = 1000
		localSafeNum uint64 = 340
		localSafeTs  uint64 = 1020 // 340 * blockTime
	)
	depth := 60 * time.Second // crossSafe 1020 - 60 = 960 < activation → T_lo clamps to 1000

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

	require.NoError(t, h.interop.runLogBackfill())
	require.Equal(t, act, h.interop.activationTimestamp, "protocol activation must not change")
	require.Equal(t, localSafeTs+1, h.interop.runtimeActivationTimestamp)

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
	depth := time.Second // crossSafe 110, depth 1s -> T_lo 109; seals 109..110; activation advances to 111

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
	h.interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
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

	require.NoError(t, h.interop.runLogBackfill())
	require.Equal(t, uint64(111), h.interop.runtimeActivationTimestamp)
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
