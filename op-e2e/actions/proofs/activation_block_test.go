package proofs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

type activationBlockTestCfg struct {
	fork          rollup.ForkName
	numUpgradeTxs int
}

// TestUpgradeBlockTxOmission tests that the sequencer omits user transactions in activation blocks
// and that batches that contain user transactions in an activation block are dropped.
func TestActivationBlockTxOmission(gt *testing.T) {
	matrix := helpers.NewMatrix[activationBlockTestCfg]()

	matrix.AddDefaultTestCasesWithName(
		string(rollup.Jovian),
		activationBlockTestCfg{fork: rollup.Jovian, numUpgradeTxs: 5},
		helpers.NewForkMatrix(helpers.Isthmus),
		testActivationBlockTxOmission,
	)
	// New forks should be added here in the future.

	matrix.Run(gt)
}

func testActivationBlockTxOmission(gt *testing.T, testCfg *helpers.TestCfg[activationBlockTestCfg]) {
	tcfg := testCfg.Custom
	t := actionsHelpers.NewDefaultTesting(gt)
	offset := uint64(4)
	testSetup := func(dc *genesis.DeployConfig) {
		dc.L1PragueTimeOffset = ptr(hexutil.Uint64(0))
		// activate fork after a few blocks
		dc.SetForkTimeOffset(tcfg.fork, &offset)
	}
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), testSetup)

	engine := env.Engine
	sequencer := env.Sequencer
	miner := env.Miner
	rollupCfg := env.Sd.RollupCfg
	blockTime := rollupCfg.BlockTime

	miner.ActEmptyBlock(t)

	sequencer.ActL1HeadSignal(t)
	for i := 0; i < int(offset)-1; i++ {
		sequencer.ActL2EmptyBlock(t)
	}
	tx := env.Alice.L2.ActMakeTx(t)
	sequencer.ActL2StartBlock(t)
	// we assert later that the sequencer actually omits this tx in the activation block
	engine.ActL2IncludeTx(env.Alice.Address())
	sequencer.ActL2EndBlock(t)

	actHeader := engine.L2Chain().CurrentHeader()
	require.Equal(t, tcfg.fork,
		rollupCfg.IsActivationBlock(actHeader.Time-blockTime, actHeader.Time),
		"this block should be the activation block")
	actBlock := engine.L2Chain().GetBlockByHash(actHeader.Hash())
	require.Len(t, actBlock.Transactions(), tcfg.numUpgradeTxs+1, "activation block contains unexpected txs")

	batcher := env.Batcher
	for i := 0; i < int(offset)-1; i++ {
		batcher.ActL2BatchBuffer(t)
	}
	batcher.ActL2BatchBuffer(t,
		actionsHelpers.WithBlockModifier(func(block *types.Block) *types.Block {
			// inject user tx into activation batch
			return block.WithBody(types.Body{Transactions: append(block.Transactions(), tx)})
		}))

	batcher.ActL2ChannelClose(t)
	batcher.ActL2BatchSubmit(t)
	env.Miner.ActL1StartBlock(12)(t)
	env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
	env.Miner.ActL1EndBlock(t)

	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	recs := env.Logs.FindLogs(testlog.NewMessageFilter("dropping batch with user transactions in fork activation block"))
	require.Len(t, recs, 1)

	l2SafeHead := engine.L2Chain().CurrentSafeBlock()
	preActHeader := engine.L2Chain().GetHeaderByHash(actHeader.ParentHash)
	require.Equal(t, eth.HeaderBlockID(preActHeader), eth.HeaderBlockID(l2SafeHead), "derivation only reaches pre-upgrade block")

	env.RunFaultProofProgramFromGenesis(t, l2SafeHead.Number.Uint64(), testCfg.CheckResult, testCfg.InputParams...)
}
