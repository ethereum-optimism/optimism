package driver

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// TestEngineConfirmedResetTruncatesToLocalSafe checks that history below local-safe
// survives a reset. Nothing records it again, and L1AtSafeHead answers across a gap
// without reporting one.
func TestEngineConfirmedResetTruncatesToLocalSafe(t *testing.T) {
	ctx := context.Background()
	logger := testlog.Logger(t, log.LvlInfo)

	db, err := safedb.NewSafeDB(logger, t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	l1 := func(num uint64) eth.BlockID {
		return eth.BlockID{Number: num, Hash: common.Hash{'l', '1', byte(num)}}
	}
	l2 := func(num uint64) eth.L2BlockRef {
		return eth.L2BlockRef{Number: num, Hash: common.Hash{'l', '2', byte(num)}, L1Origin: l1(num / 2)}
	}

	require.NoError(t, db.SafeHeadUpdated(l2(20), l1(10)))
	require.NoError(t, db.SafeHeadUpdated(l2(40), l1(20)))
	require.NoError(t, db.SafeHeadUpdated(l2(80), l1(40)))
	require.NoError(t, db.SafeHeadUpdated(l2(100), l1(50)))

	emitter := &testutils.MockEmitter{}
	emitter.ExpectOnceType("ConfirmPipelineResetEvent")
	s := &SyncDeriver{
		SafeHeadNotifs: db,
		Config:         &rollup.Config{Genesis: rollup.Genesis{L2: eth.BlockID{Number: 0}}},
		Emitter:        emitter,
		Log:            logger,
		Ctx:            ctx,
	}

	// Cross-safe lags local-safe, as it does under a SuperAuthority.
	s.onEngineConfirmedReset(ctx, engine.EngineResetConfirmedEvent{
		LocalUnsafe: l2(120),
		LocalSafe:   l2(100),
		CrossSafe:   l2(40),
		Finalized:   l2(40),
	})
	emitter.AssertExpectations(t)

	require.NoError(t, db.SafeHeadUpdated(l2(140), l1(70)))

	gotL1, gotSafe, err := db.L1AtSafeHead(ctx, 80)
	require.NoError(t, err)
	require.Equal(t, l1(40), gotL1)
	require.Equal(t, l2(80).ID(), gotSafe)
}

// TestEngineConfirmedResetTruncatesAboveLocalSafe checks the other half: a reset
// removes the entries that derivation re-derives.
func TestEngineConfirmedResetTruncatesAboveLocalSafe(t *testing.T) {
	ctx := context.Background()
	logger := testlog.Logger(t, log.LvlInfo)

	db, err := safedb.NewSafeDB(logger, t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	l1 := func(num uint64) eth.BlockID {
		return eth.BlockID{Number: num, Hash: common.Hash{'l', '1', byte(num)}}
	}
	l2 := func(num uint64) eth.L2BlockRef {
		return eth.L2BlockRef{Number: num, Hash: common.Hash{'l', '2', byte(num)}, L1Origin: l1(num / 2)}
	}

	require.NoError(t, db.SafeHeadUpdated(l2(40), l1(20)))
	require.NoError(t, db.SafeHeadUpdated(l2(80), l1(40)))
	require.NoError(t, db.SafeHeadUpdated(l2(100), l1(50)))

	emitter := &testutils.MockEmitter{}
	emitter.ExpectOnceType("ConfirmPipelineResetEvent")
	s := &SyncDeriver{
		SafeHeadNotifs: db,
		Config:         &rollup.Config{Genesis: rollup.Genesis{L2: eth.BlockID{Number: 0}}},
		Emitter:        emitter,
		Log:            logger,
		Ctx:            ctx,
	}

	// An L1 reorg pulled local-safe back to 80, below the unsafe head.
	s.onEngineConfirmedReset(ctx, engine.EngineResetConfirmedEvent{
		LocalUnsafe: l2(120),
		LocalSafe:   l2(80),
		CrossSafe:   l2(40),
		Finalized:   l2(40),
	})
	emitter.AssertExpectations(t)

	gotL1, gotSafe, err := db.LastEntry(ctx)
	require.NoError(t, err)
	require.Equal(t, l1(40), gotL1)
	require.Equal(t, l2(80).ID(), gotSafe, "the entry for block 100 is removed")
}
