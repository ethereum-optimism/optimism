package actions

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestDencunL1Fork(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	offset := uint64(24)
	dp.DeployConfig.L1CancunTimeOffset = &offset
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	_, _, miner, sequencer, _, verifier, _, batcher := setupReorgTestActors(t, dp, sd, log)

	l1Head := miner.l1Chain.CurrentBlock()
	require.False(t, sd.L1Cfg.Config.IsCancun(l1Head.Number, l1Head.Time), "Cancun not active yet")
	require.Nil(t, l1Head.ExcessBlobGas, "Cancun blob gas not in header")

	// start op-nodes
	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	// build empty L1 blocks, crossing the fork boundary
	miner.ActL1SetFeeRecipient(common.Address{'A', 0})
	miner.ActEmptyBlock(t)
	miner.ActEmptyBlock(t) // Cancun activates here
	miner.ActEmptyBlock(t)
	// verify Cancun is active
	l1Head = miner.l1Chain.CurrentBlock()
	require.True(t, sd.L1Cfg.Config.IsCancun(l1Head.Number, l1Head.Time), "Cancun active")
	require.NotNil(t, l1Head.ExcessBlobGas, "Cancun blob gas in header")

	// build L2 chain up to and including L2 blocks referencing Cancun L1 blocks
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)
	miner.ActL1StartBlock(12)(t)
	batcher.ActSubmitAll(t)
	miner.ActL1IncludeTx(batcher.batcherAddr)(t)
	miner.ActL1EndBlock(t)

	// sync verifier
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	// verify verifier accepted Cancun L1 inputs
	require.Equal(t, l1Head.Hash(), verifier.SyncStatus().SafeL2.L1Origin.Hash, "verifier synced L1 chain that includes Cancun headers")
	require.Equal(t, sequencer.SyncStatus().UnsafeL2, verifier.SyncStatus().UnsafeL2, "verifier and sequencer agree")
}

func TestDencunL1ForkAtGenesis(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	offset := uint64(0)
	dp.DeployConfig.L1CancunTimeOffset = &offset
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	_, _, miner, sequencer, _, verifier, _, batcher := setupReorgTestActors(t, dp, sd, log)

	l1Head := miner.l1Chain.CurrentBlock()
	require.True(t, sd.L1Cfg.Config.IsCancun(l1Head.Number, l1Head.Time), "Cancun active at genesis")
	require.NotNil(t, l1Head.ExcessBlobGas, "Cancun blob gas in header")

	// start op-nodes
	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	// build empty L1 blocks
	miner.ActL1SetFeeRecipient(common.Address{'A', 0})
	miner.ActEmptyBlock(t)
	miner.ActEmptyBlock(t)

	// verify Cancun is still active
	l1Head = miner.l1Chain.CurrentBlock()
	require.True(t, sd.L1Cfg.Config.IsCancun(l1Head.Number, l1Head.Time), "Cancun active")
	require.NotNil(t, l1Head.ExcessBlobGas, "Cancun blob gas in header")

	// build L2 chain
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)
	miner.ActL1StartBlock(12)(t)
	batcher.ActSubmitAll(t)
	miner.ActL1IncludeTx(batcher.batcherAddr)(t)
	miner.ActL1EndBlock(t)

	// sync verifier
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)

	// verify verifier accepted Cancun L1 inputs
	require.Equal(t, l1Head.Hash(), verifier.SyncStatus().SafeL2.L1Origin.Hash, "verifier synced L1 chain that includes Cancun headers")
	require.Equal(t, sequencer.SyncStatus().UnsafeL2, verifier.SyncStatus().UnsafeL2, "verifier and sequencer agree")
}
