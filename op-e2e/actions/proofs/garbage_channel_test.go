package proofs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/actions"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-program/client/claim"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

// garbageKinds is a list of garbage kinds to test. We don't use `INVALID_COMPRESSION` and `MALFORM_RLP` because
// they submit malformed frames always, and this test models a valid channel with a single invalid frame in the
// middle.
var garbageKinds = []actions.GarbageKind{
	actions.STRIP_VERSION,
	actions.RANDOM,
	actions.TRUNCATE_END,
	actions.DIRTY_APPEND,
}

// Run a test that submits garbage channel data in the middle of a channel.
//
// channel format ([]Frame):
// [f[0 - correct] f_x[1 - bad frame] f[1 - correct]]
func runGarbageChannelTest(gt *testing.T, garbageKind actions.GarbageKind, checkResult func(gt *testing.T, err error), inputParams ...FixtureInputParam) {
	t := actions.NewDefaultTesting(gt)
	tp := NewTestParams(func(tp *e2eutils.TestParams) {
		// Set the channel timeout to 10 blocks, 12x lower than the sequencing window.
		tp.ChannelTimeout = 10
	})
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

	includeBatchTx := func(env *L2FaultProofEnv) {
		// Instruct the batcher to submit the first channel frame to L1, and include the transaction.
		env.miner.ActL1StartBlock(12)(t)
		env.miner.ActL1IncludeTxByHash(env.batcher.LastSubmitted.Hash())(t)
		env.miner.ActL1EndBlock(t)

		// Finalize the block with the first channel frame on L1.
		env.miner.ActL1SafeNext(t)
		env.miner.ActL1FinalizeNext(t)

		// Instruct the sequencer to derive the L2 chain from the data on L1 that the batcher just posted.
		env.sequencer.ActL1HeadSignal(t)
		env.sequencer.ActL2PipelineFull(t)
	}

	const NumL2Blocks = 10

	// Build NumL2Blocks empty blocks on L2
	for i := 0; i < NumL2Blocks; i++ {
		env.sequencer.ActL2StartBlock(t)
		env.sequencer.ActL2EndBlock(t)
	}

	// Buffer the first half of L2 blocks in the batcher, and submit it.
	for i := 0; i < NumL2Blocks/2; i++ {
		env.batcher.ActL2BatchBuffer(t)
	}
	env.batcher.ActL2BatchSubmit(t)

	// Include the batcher transaction.
	includeBatchTx(env)

	// Ensure that the safe head has not advanced - the channel is incomplete.
	l2SafeHead := env.engine.L2Chain().CurrentSafeBlock()
	require.Equal(t, uint64(0), l2SafeHead.Number.Uint64())

	// Buffer the second half of L2 blocks in the batcher.
	for i := 0; i < NumL2Blocks/2; i++ {
		env.batcher.ActL2BatchBuffer(t)
	}
	env.batcher.ActL2ChannelClose(t)
	expectedSecondFrame := env.batcher.ReadNextOutputFrame(t)

	// Submit a garbage frame, modified from the expected second frame.
	env.batcher.ActL2BatchSubmitGarbageRaw(t, expectedSecondFrame, garbageKind)
	// Include the garbage second frame tx
	includeBatchTx(env)

	// Ensure that the safe head has not advanced - the channel is incomplete.
	l2SafeHead = env.engine.L2Chain().CurrentSafeBlock()
	require.Equal(t, uint64(0), l2SafeHead.Number.Uint64())

	// Submit the correct second frame.
	env.batcher.ActL2BatchSubmitRaw(t, expectedSecondFrame)
	// Include the corract second frame tx.
	includeBatchTx(env)

	// Ensure that the safe head has advanced - the channel is complete.
	l2SafeHead = env.engine.L2Chain().CurrentSafeBlock()
	require.Equal(t, uint64(NumL2Blocks), l2SafeHead.Number.Uint64())

	// Run the FPP on L2 block # NumL2Blocks. The claim is honest, so it should pass.
	err := env.RunFaultProofProgram(t, gt, NumL2Blocks, inputParams...)
	checkResult(gt, err)
}

func Test_ProgramAction_GarbageChannel_HonestClaim_Granite(gt *testing.T) {
	for _, garbageKind := range garbageKinds {
		gt.Run(garbageKind.String(), func(t *testing.T) {
			runGarbageChannelTest(
				t,
				garbageKind,
				func(gt *testing.T, err error) {
					require.NoError(gt, err, "fault proof program should not have failed")
				},
			)
		})
	}
}

func Test_ProgramAction_GarbageChannel_JunkClaim_Granite(gt *testing.T) {
	for _, garbageKind := range garbageKinds {
		gt.Run(garbageKind.String(), func(t *testing.T) {
			runGarbageChannelTest(
				t,
				garbageKind,
				func(gt *testing.T, err error) {
					require.ErrorIs(gt, err, claim.ErrClaimNotValid, "fault proof program should have failed")
				},
				func(f *FixtureInputs) {
					f.L2Claim = common.HexToHash("0xdeadbeef")
				},
			)
		})
	}
}
