package helpers

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// rethSyncFixture is a reth sequencer engine with a short chain, and the setup to create further
// engines next to it.
type rethSyncFixture struct {
	t      StatefulTesting
	sd     *e2eutils.SetupData
	logger log.Logger
	logs   *testlog.CapturingHandler
	seqEng *L2Engine
}

func newRethSyncFixture(gt *testing.T, blocks int) *rethSyncFixture {
	gt.Setenv(ELSelectorEnv, elRethTestEngine)
	t := SubTest(gt)
	dp := e2eutils.MakeDeployParams(t, DefaultRollupTestParams())
	sd := e2eutils.Setup(t, dp, DefaultAlloc)
	logger, logs := testlog.CaptureLogger(t, log.LevelInfo)
	miner, seqEng, sequencer := SetupSequencerTest(t, sd, logger)
	miner.ActEmptyBlock(t)
	sequencer.ActL2PipelineFull(t)
	for range blocks {
		sequencer.ActL2StartBlock(t)
		sequencer.ActL2EndBlock(t)
	}
	return &rethSyncFixture{t: t, sd: sd, logger: logger, logs: logs, seqEng: seqEng}
}

// requestSync points eng's forkchoice at a head it lacks, which makes it report SYNCING and record
// the head as its sync target.
func (f *rethSyncFixture) requestSync(eng *L2Engine, head common.Hash) {
	var res eth.ForkchoiceUpdatedResult
	require.NoError(f.t, eng.reth.client.CallContext(f.t.Ctx(), &res, "engine_forkchoiceUpdatedV3",
		eth.ForkchoiceState{HeadBlockHash: head}, nil))
	require.Equal(f.t, eth.ExecutionSyncing, res.PayloadStatus.Status)
}

// TestRethSyncPumpStopsAtTarget checks that a backfill pass imports up to the sync target and not
// beyond it to the peer's tip.
func TestRethSyncPumpStopsAtTarget(gt *testing.T) {
	f := newRethSyncFixture(gt, 6)
	fresh := newRethL2Engine(f.t, f.logger, f.sd.L2Cfg)
	target := f.seqEng.BlockByNumber(f.t, 3).Hash()
	f.requestSync(fresh, target)

	fresh.reth.pumpOnce(context.Background(), f.seqEng.reth)

	head := fresh.LatestHeader(f.t)
	require.EqualValues(f.t, 3, bigs.Uint64Strict(head.Number), "backfilled past the sync target")
	require.Equal(f.t, target, head.Hash())
}

// buildLocalBlocks makes eng build n empty blocks on its head through the engine API and makes each
// the head. Their timestamps step by three seconds, which the sequencer's two-second blocks never
// share, so the blocks fork off the sequencer's chain.
func (f *rethSyncFixture) buildLocalBlocks(eng *L2Engine, n int) {
	cl, err := sources.NewEngineClient(eng.RPCClient(), f.logger, nil, sources.EngineClientDefaultConfig(f.sd.RollupCfg))
	require.NoError(f.t, err)
	for range n {
		parent := eng.LatestHeader(f.t)
		ts := parent.Time + 3
		attrs := &eth.PayloadAttributes{
			Timestamp: eth.Uint64Quantity(ts),
			GasLimit:  (*eth.Uint64Quantity)(&f.sd.RollupCfg.Genesis.SystemConfig.GasLimit),
		}
		if f.sd.RollupCfg.IsCanyon(ts) {
			attrs.Withdrawals = &types.Withdrawals{}
		}
		if f.sd.RollupCfg.IsEcotone(ts) {
			attrs.ParentBeaconBlockRoot = &common.Hash{}
		}
		if f.sd.RollupCfg.IsHolocene(ts) {
			attrs.EIP1559Params = &eth.Bytes8{}
		}
		if f.sd.RollupCfg.IsJovian(ts) {
			attrs.MinBaseFee = new(uint64)
		}
		res, err := cl.ForkchoiceUpdate(f.t.Ctx(), &eth.ForkchoiceState{HeadBlockHash: parent.Hash()}, attrs)
		require.NoError(f.t, err)
		require.NotNil(f.t, res.PayloadID)
		envelope, err := cl.GetPayload(f.t.Ctx(), eth.PayloadInfo{ID: *res.PayloadID, Timestamp: ts})
		require.NoError(f.t, err)
		status, err := cl.NewPayload(f.t.Ctx(), envelope.ExecutionPayload, envelope.ParentBeaconBlockRoot)
		require.NoError(f.t, err)
		require.Equal(f.t, eth.ExecutionValid, status.Status)
		res, err = cl.ForkchoiceUpdate(f.t.Ctx(), &eth.ForkchoiceState{HeadBlockHash: envelope.ExecutionPayload.BlockHash}, nil)
		require.NoError(f.t, err)
		require.Equal(f.t, eth.ExecutionValid, res.PayloadStatus.Status)
	}
}

// requireSyncedTo checks that a forkchoice update to target, which reported SYNCING before the
// backfill, now canonicalizes it on eng.
func (f *rethSyncFixture) requireSyncedTo(eng *L2Engine, target common.Hash) {
	var res eth.ForkchoiceUpdatedResult
	require.NoError(f.t, eng.reth.client.CallContext(f.t.Ctx(), &res, "engine_forkchoiceUpdatedV3",
		eth.ForkchoiceState{HeadBlockHash: target}, nil))
	require.Equal(f.t, eth.ExecutionValid, res.PayloadStatus.Status, "backfill left the target unreachable")
	require.Equal(f.t, target, eng.LatestHeader(f.t).Hash())
}

// TestRethSyncPumpBackfillsFromTheForkPoint checks that an engine whose head is on a fork below the
// sync target backfills the peer's chain from where the two chains diverge, not from its own head.
func TestRethSyncPumpBackfillsFromTheForkPoint(gt *testing.T) {
	f := newRethSyncFixture(gt, 6)
	ver := newRethL2Engine(f.t, f.logger, f.sd.L2Cfg)
	f.requestSync(ver, f.seqEng.BlockByNumber(f.t, 2).Hash())
	ver.reth.pumpOnce(context.Background(), f.seqEng.reth)
	f.requireSyncedTo(ver, f.seqEng.BlockByNumber(f.t, 2).Hash())
	f.buildLocalBlocks(ver, 1)

	target := f.seqEng.BlockByNumber(f.t, 5).Hash()
	f.requestSync(ver, target)
	ver.reth.pumpOnce(context.Background(), f.seqEng.reth)

	// The target stays set until the next forkchoice update; a further pass must not import the
	// side blocks again.
	imported := func() int {
		return len(f.logs.FindLogs(testlog.NewMessageFilter("EL sync imported block from peer")))
	}
	before := imported()
	ver.reth.pumpOnce(context.Background(), f.seqEng.reth)
	require.Equal(f.t, before, imported(), "a backfilled target was imported again")

	f.requireSyncedTo(ver, target)
}

// TestRethSyncPumpBackfillsBelowAHigherForkHead checks that an engine whose head is on a fork at
// or above the sync target's height still backfills the peer's chain up to the target.
func TestRethSyncPumpBackfillsBelowAHigherForkHead(gt *testing.T) {
	f := newRethSyncFixture(gt, 6)
	ver := newRethL2Engine(f.t, f.logger, f.sd.L2Cfg)
	f.requestSync(ver, f.seqEng.BlockByNumber(f.t, 2).Hash())
	ver.reth.pumpOnce(context.Background(), f.seqEng.reth)
	f.requireSyncedTo(ver, f.seqEng.BlockByNumber(f.t, 2).Hash())
	f.buildLocalBlocks(ver, 4)

	target := f.seqEng.BlockByNumber(f.t, 4).Hash()
	f.requestSync(ver, target)
	ver.reth.pumpOnce(context.Background(), f.seqEng.reth)

	f.requireSyncedTo(ver, target)
}

// TestRethSyncPumpWarnsOnceWithoutACommonBlock checks that a peer the engine shares no block with
// is reported at Warn, once, rather than skipped silently on every pass.
func TestRethSyncPumpWarnsOnceWithoutACommonBlock(gt *testing.T) {
	f := newRethSyncFixture(gt, 3)
	// An engine on a different genesis shares no block with the peer.
	other := *f.sd.L2Cfg
	other.Timestamp++
	foreign := newRethL2Engine(f.t, f.logger, &other)
	f.requestSync(foreign, f.seqEng.BlockByNumber(f.t, 2).Hash())

	foreign.reth.pumpOnce(context.Background(), f.seqEng.reth)
	foreign.reth.pumpOnce(context.Background(), f.seqEng.reth)

	warnings := f.logs.FindLogs(testlog.NewLevelFilter(log.LevelWarn), testlog.NewMessageContainsFilter("EL sync"))
	require.Len(f.t, warnings, 1, "want one warning for the unsyncable peer")
}

// TestRethAddPeerStartsBothPumps checks that peering two engines lets each backfill from the other.
func TestRethAddPeerStartsBothPumps(gt *testing.T) {
	f := newRethSyncFixture(gt, 0)
	a := newRethL2Engine(f.t, f.logger, f.sd.L2Cfg)
	b := newRethL2Engine(f.t, f.logger, f.sd.L2Cfg)

	a.AddPeers(b.Enode())

	pumps := func(e *L2Engine) int {
		e.reth.mu.Lock()
		defer e.reth.mu.Unlock()
		return len(e.reth.pumpCancels)
	}
	require.Equal(f.t, 1, pumps(a))
	require.Equal(f.t, 1, pumps(b))
}
