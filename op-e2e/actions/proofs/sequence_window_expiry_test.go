package proofs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/actions"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// Run a test that proves a deposit-only block generated due to sequence window expiry.
func runSequenceWindowExpireTest(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	t := actions.NewDefaultTesting(gt)
	tp := helpers.NewTestParams()
	env := helpers.NewL2FaultProofEnv(t, testCfg, tp, helpers.NewBatcherCfg())

	// Mine an empty block for gas estimation purposes.
	env.Miner.ActEmptyBlock(t)

	// Expire the sequence window by building `SequenceWindow + 1` empty blocks on L1.
	for i := 0; i < int(tp.SequencerWindowSize)+1; i++ {
		env.Alice.L1.ActResetTxOpts(t)
		env.Alice.ActDeposit(t)

		env.Miner.ActL1StartBlock(12)(t)
		env.Miner.ActL1IncludeTx(env.Alice.Address())(t)
		env.Miner.ActL1EndBlock(t)

		env.Miner.ActL1SafeNext(t)
		env.Miner.ActL1FinalizeNext(t)
	}

	// Ensure the safe head is still 0.
	l2SafeHead := env.Engine.L2Chain().CurrentSafeBlock()
	require.EqualValues(t, 0, l2SafeHead.Number.Uint64())

	// Ask the sequencer to derive the deposit-only L2 chain.
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	// Ensure the safe head advanced forcefully.
	l2SafeHead = env.Engine.L2Chain().CurrentSafeBlock()
	require.Greater(t, l2SafeHead.Number.Uint64(), uint64(0))

	// Run the FPP on one of the auto-derived blocks.
	env.RunFaultProofProgram(t, l2SafeHead.Number.Uint64()/2, testCfg.CheckResult, testCfg.InputParams...)
}

func Test_ProgramAction_SequenceWindowExpired(gt *testing.T) {
	matrix := helpers.NewMatrix[any]()
	defer matrix.Run(gt)

	matrix.AddTestCase(
		"HonestClaim",
		nil,
		helpers.LatestForkOnly,
		runSequenceWindowExpireTest,
		helpers.ExpectNoError(),
	)
	matrix.AddTestCase(
		"JunkClaim",
		nil,
		helpers.LatestForkOnly,
		runSequenceWindowExpireTest,
		helpers.ExpectNoError(),
		helpers.WithL2Claim(common.HexToHash("0xdeadbeef")),
	)
}
