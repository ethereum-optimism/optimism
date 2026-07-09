package dsl

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// programmableSyncProvider is a SyncStatusProvider whose returned BlockID per
// (chainID, level) can be updated at runtime from the test goroutine.
type programmableSyncProvider struct {
	name string

	mu   sync.Mutex
	byID map[safety.Level]eth.BlockID
}

func newProgrammableSyncProvider(name string) *programmableSyncProvider {
	return &programmableSyncProvider{
		name: name,
		byID: map[safety.Level]eth.BlockID{},
	}
}

func (p *programmableSyncProvider) set(lvl safety.Level, num uint64, hash common.Hash) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.byID[lvl] = eth.BlockID{Number: num, Hash: hash}
}

func (p *programmableSyncProvider) ChainSyncStatus(chainID eth.ChainID, lvl safety.Level) eth.BlockID {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.byID[lvl]
}

func (p *programmableSyncProvider) String() string { return p.name }

func TestMatchedWithProgressFn_SucceedsOnMatch(t *testing.T) {
	t.Parallel()
	base := newProgrammableSyncProvider("base")
	ref := newProgrammableSyncProvider("ref")
	chainID := eth.ChainIDFromUInt64(901)
	h := common.HexToHash("0xabc")
	base.set(safety.CrossSafe, 10, h)
	base.set(safety.LocalUnsafe, 10, h)
	ref.set(safety.CrossSafe, 10, h)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	check := MatchedWithProgressFn(base, ref, testlog.Logger(t, log.LevelDebug), ctx,
		safety.CrossSafe, safety.LocalUnsafe, chainID,
		5*time.Second, 1*time.Second)

	require.NoError(t, check())
}

func TestMatchedWithProgressFn_FailsOnStall(t *testing.T) {
	t.Parallel()
	base := newProgrammableSyncProvider("base")
	ref := newProgrammableSyncProvider("ref")
	chainID := eth.ChainIDFromUInt64(901)
	h1 := common.HexToHash("0x1")
	h2 := common.HexToHash("0x2")
	// Base CrossSafe never matches; LocalUnsafe never advances either.
	base.set(safety.CrossSafe, 0, common.Hash{})
	base.set(safety.LocalUnsafe, 5, h1)
	ref.set(safety.CrossSafe, 10, h2)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	check := MatchedWithProgressFn(base, ref, testlog.Logger(t, log.LevelDebug), ctx,
		safety.CrossSafe, safety.LocalUnsafe, chainID,
		30*time.Second, 3*time.Second)

	start := time.Now()
	err := check()
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "stalled"), "expected stall error, got: %v", err)
	// Stall should fire well before the 30s max wait.
	require.Less(t, time.Since(start), 8*time.Second, "stall detector took too long: %s", time.Since(start))
}

func TestMatchedWithProgressFn_KeepsWaitingWhileProgressing(t *testing.T) {
	t.Parallel()
	base := newProgrammableSyncProvider("base")
	ref := newProgrammableSyncProvider("ref")
	chainID := eth.ChainIDFromUInt64(901)
	h := common.HexToHash("0xabc")
	// Base CrossSafe stalls at 0; LocalUnsafe steadily advances; ref CrossSafe at 10.
	base.set(safety.CrossSafe, 0, common.Hash{})
	base.set(safety.LocalUnsafe, 5, common.HexToHash("0x5"))
	ref.set(safety.CrossSafe, 10, h)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Drive LocalUnsafe progress in the background past where a fixed-attempt
	// budget would have given up, and then converge CrossSafe.
	var driverWG sync.WaitGroup
	driverWG.Add(1)
	go func() {
		defer driverWG.Done()
		for i := 6; i <= 12; i++ {
			if err := clock.SystemClock.SleepCtx(ctx, 500*time.Millisecond); err != nil {
				return
			}
			base.set(safety.LocalUnsafe, uint64(i), common.BigToHash(common.Big1))
		}
		// Now let CrossSafe match.
		base.set(safety.CrossSafe, 10, h)
	}()

	// Stall timeout is short (1s) but the driver advances LocalUnsafe every
	// 500ms, so the stall detector should never trip.
	check := MatchedWithProgressFn(base, ref, testlog.Logger(t, log.LevelDebug), ctx,
		safety.CrossSafe, safety.LocalUnsafe, chainID,
		30*time.Second, 1*time.Second)
	require.NoError(t, check())
	driverWG.Wait()
}

func TestMatchedWithProgressFn_FailsOnMaxWait(t *testing.T) {
	t.Parallel()
	base := newProgrammableSyncProvider("base")
	ref := newProgrammableSyncProvider("ref")
	chainID := eth.ChainIDFromUInt64(901)
	h := common.HexToHash("0xabc")
	// Base CrossSafe never matches; LocalUnsafe keeps advancing so the stall
	// detector never trips — the deadline should be the failure path.
	base.set(safety.CrossSafe, 0, common.Hash{})
	base.set(safety.LocalUnsafe, 0, common.Hash{})
	ref.set(safety.CrossSafe, 10, h)

	driverCtx, driverCancel := context.WithCancel(context.Background())
	var driverWG sync.WaitGroup
	driverWG.Add(1)
	go func() {
		defer driverWG.Done()
		var i uint64
		for {
			select {
			case <-driverCtx.Done():
				return
			case <-time.After(200 * time.Millisecond):
				i++
				base.set(safety.LocalUnsafe, i, common.BigToHash(common.Big1))
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	check := MatchedWithProgressFn(base, ref, testlog.Logger(t, log.LevelDebug), ctx,
		safety.CrossSafe, safety.LocalUnsafe, chainID,
		5*time.Second, 30*time.Second)

	start := time.Now()
	err := check()
	driverCancel()
	driverWG.Wait()

	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "timeout"), "expected timeout error, got: %v", err)
	require.GreaterOrEqual(t, time.Since(start), 5*time.Second)
}
