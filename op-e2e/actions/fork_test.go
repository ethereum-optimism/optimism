package actions

import (
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
)

func TestShapellaL1Fork(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)

	sd := e2eutils.Setup(t, dp, defaultAlloc)
	activation := sd.L1Cfg.Timestamp + 24
	sd.L1Cfg.Config.ShanghaiTime = &activation
	log := testlog.Logger(t, log.LvlDebug)

	_, _, miner, sequencer, _, verifier, _, batcher := setupReorgTestActors(t, dp, sd, log)

	require.False(t, sd.L1Cfg.Config.IsShanghai(miner.l1Chain.CurrentBlock().Time), "not active yet")

	// start op-nodes
	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	// build empty L1 blocks, crossing the fork boundary
	miner.ActEmptyBlock(t)
	miner.ActEmptyBlock(t)
	miner.ActEmptyBlock(t)

	// verify Shanghai is active
	l1Head := miner.l1Chain.CurrentBlock()
	require.True(t, sd.L1Cfg.Config.IsShanghai(l1Head.Time))

	// build L2 chain up to and including L2 blocks referencing shanghai L1 blocks
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)
	miner.ActL1StartBlock(12)(t)
	batcher.ActSubmitAll(t)
	miner.ActL1IncludeTx(batcher.batcherAddr)(t)
	miner.ActL1EndBlock(t)

	// sync verifier
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	// verify verifier accepted shanghai L1 inputs
	require.Equal(t, l1Head.Hash(), verifier.SyncStatus().SafeL2.L1Origin.Hash, "verifier synced L1 chain that includes shanghai headers")
	require.Equal(t, sequencer.SyncStatus().UnsafeL2, verifier.SyncStatus().UnsafeL2, "verifier and sequencer agree")
}
