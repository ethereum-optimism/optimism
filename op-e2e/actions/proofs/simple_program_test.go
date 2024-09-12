package proofs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/actions"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func Test_ProgramAction_SimpleEmptyChain_HonestClaim_Granite(gt *testing.T) {
	t := actions.NewDefaultTesting(gt)
	tp := NewTestParams()
	dp := NewDeployParams(t, func(dp *e2eutils.DeployParams) {
		genesisBlock := hexutil.Uint64(0)

		// Enable Cancun on L1 & Granite on L2 at genesis
		dp.DeployConfig.L1CancunTimeOffset = &genesisBlock
		dp.DeployConfig.L2GenesisRegolithTimeOffset = &genesisBlock
		dp.DeployConfig.L2GenesisCanyonTimeOffset = &genesisBlock
		dp.DeployConfig.L2GenesisDeltaTimeOffset = &genesisBlock
		dp.DeployConfig.L2GenesisEcotoneTimeOffset = &genesisBlock
		dp.DeployConfig.L2GenesisFjordTimeOffset = &genesisBlock
		dp.DeployConfig.L2GenesisGraniteTimeOffset = &genesisBlock
	})
	bCfg := NewBatcherCfg()
	env := NewL2FaultProofEnv(t, tp, dp, bCfg)

	// Build an empty block on L2
	env.sequencer.ActL2StartBlock(t)
	env.sequencer.ActL2EndBlock(t)

	// Instruct the batcher to submit the block to L1, and include the transaction.
	env.batcher.ActSubmitAll(t)
	env.miner.ActL1StartBlock(12)(t)
	env.miner.ActL1IncludeTxByHash(env.batcher.LastSubmitted.Hash())(t)
	env.miner.ActL1EndBlock(t)

	// Finalize the block with the batch on L1.
	env.miner.ActL1SafeNext(t)
	env.miner.ActL1FinalizeNext(t)

	// Instruct the sequencer to derive the L2 chain from the data on L1 that the batcher just posted.
	env.sequencer.ActL1HeadSignal(t)
	env.sequencer.ActL2PipelineFull(t)

	l1Head := env.miner.L1Chain().CurrentBlock()
	l2SafeHead := env.engine.L2Chain().CurrentSafeBlock()

	// Ensure there is only 1 block on L1.
	require.Equal(t, uint64(1), l1Head.Number.Uint64())
	// Ensure the block is marked as safe before we attempt to fault prove it.
	require.Equal(t, uint64(1), l2SafeHead.Number.Uint64())

	err := env.RunFaultProofProgram(t, gt, l2SafeHead.Number.Uint64())
	require.NoError(t, err, "fault proof program failed")
}

func Test_ProgramAction_SimpleEmptyChain_JunkClaim_Granite(gt *testing.T) {
	t := actions.NewDefaultTesting(gt)
	tp := NewTestParams()
	dp := NewDeployParams(t, func(dp *e2eutils.DeployParams) {
		genesisBlock := hexutil.Uint64(0)

		// Enable Cancun on L1 & Granite on L2 at genesis
		dp.DeployConfig.L1CancunTimeOffset = &genesisBlock
		dp.DeployConfig.L2GenesisRegolithTimeOffset = &genesisBlock
		dp.DeployConfig.L2GenesisCanyonTimeOffset = &genesisBlock
		dp.DeployConfig.L2GenesisDeltaTimeOffset = &genesisBlock
		dp.DeployConfig.L2GenesisEcotoneTimeOffset = &genesisBlock
		dp.DeployConfig.L2GenesisFjordTimeOffset = &genesisBlock
		dp.DeployConfig.L2GenesisGraniteTimeOffset = &genesisBlock
	})
	bCfg := NewBatcherCfg()
	env := NewL2FaultProofEnv(t, tp, dp, bCfg)

	// Build an empty block on L2
	env.sequencer.ActL2StartBlock(t)
	env.sequencer.ActL2EndBlock(t)

	// Instruct the batcher to submit the block to L1, and include the transaction.
	env.batcher.ActSubmitAll(t)
	env.miner.ActL1StartBlock(12)(t)
	env.miner.ActL1IncludeTxByHash(env.batcher.LastSubmitted.Hash())(t)
	env.miner.ActL1EndBlock(t)

	// Finalize the block with the batch on L1.
	env.miner.ActL1SafeNext(t)
	env.miner.ActL1FinalizeNext(t)

	// Instruct the sequencer to derive the L2 chain from the data on L1 that the batcher just posted.
	env.sequencer.ActL1HeadSignal(t)
	env.sequencer.ActL2PipelineFull(t)

	l1Head := env.miner.L1Chain().CurrentBlock()
	l2SafeHead := env.engine.L2Chain().CurrentSafeBlock()

	// Ensure there is only 1 block on L1.
	require.Equal(t, uint64(1), l1Head.Number.Uint64())
	// Ensure the block is marked as safe before we attempt to fault prove it.
	require.Equal(t, uint64(1), l2SafeHead.Number.Uint64())

	err := env.RunFaultProofProgram(t, gt, l2SafeHead.Number.Uint64(), func(f *FixtureInputs) {
		f.L2Claim = common.HexToHash("0xdeadbeef")
	})
	require.Error(t, err, "fault proof program should have failed")
}
