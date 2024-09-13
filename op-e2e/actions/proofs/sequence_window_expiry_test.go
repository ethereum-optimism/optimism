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

// Run a test that proves a deposit-only block generated due to sequence window expiry.
func runSequenceWindowExpireTest(gt *testing.T, checkResult func(gt *testing.T, err error), inputParams ...FixtureInputParam) {
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

	// Mine an empty block for gas estimation purposes.
	env.miner.ActEmptyBlock(t)

	// Expire the sequence window by building `SequenceWindow + 1` empty blocks on L1.
	for i := 0; i < int(tp.SequencerWindowSize)+1; i++ {
		env.alice.L1.ActResetTxOpts(t)
		env.alice.ActDeposit(t)

		env.miner.ActL1StartBlock(12)(t)
		env.miner.ActL1IncludeTx(env.alice.Address())(t)
		env.miner.ActL1EndBlock(t)

		env.miner.ActL1SafeNext(t)
		env.miner.ActL1FinalizeNext(t)
	}

	// Ensure the safe head is still 0.
	l2SafeHead := env.engine.L2Chain().CurrentSafeBlock()
	require.EqualValues(t, 0, l2SafeHead.Number.Uint64())

	// Ask the sequencer to derive the deposit-only L2 chain.
	env.sequencer.ActL1HeadSignal(t)
	env.sequencer.ActL2PipelineFull(t)

	// Ensure the safe head advanced forcefully.
	l2SafeHead = env.engine.L2Chain().CurrentSafeBlock()
	require.Greater(t, l2SafeHead.Number.Uint64(), uint64(0))

	// Run the FPP on one of the auto-derived blocks.
	err := env.RunFaultProofProgram(t, l2SafeHead.Number.Uint64()/2, inputParams...)
	checkResult(gt, err)
}

func Test_ProgramAction_SequenceWindowExpired_HonestClaim_Granite(gt *testing.T) {
	runSequenceWindowExpireTest(gt, func(gt *testing.T, err error) {
		require.NoError(gt, err, "fault proof program should have succeeded")
	})
}

func Test_ProgramAction_SequenceWindowExpired_JunkClaim_Granite(gt *testing.T) {
	runSequenceWindowExpireTest(
		gt,
		func(gt *testing.T, err error) {
			require.ErrorIs(gt, err, claim.ErrClaimNotValid, "fault proof program should have failed")
		},
		func(f *FixtureInputs) {
			f.L2Claim = common.HexToHash("0xdeadbeef")
		},
	)
}
