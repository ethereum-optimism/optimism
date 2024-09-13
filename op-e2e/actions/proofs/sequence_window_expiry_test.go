package proofs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/actions"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// Run a test that proves a deposit-only block generated due to sequence window expiry.
func runSequenceWindowExpireTest(gt *testing.T, testCfg *TestCfg[any]) {
	t := actions.NewDefaultTesting(gt)
	tp := NewTestParams()
	env := NewL2FaultProofEnv(t, testCfg, tp, NewBatcherCfg())

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
	env.RunFaultProofProgram(t, l2SafeHead.Number.Uint64()/2, testCfg.CheckResult, testCfg.InputParams...)
}

func Test_ProgramAction_SequenceWindowExpired(gt *testing.T) {
	matrix := NewMatrix[any]()
	defer matrix.Run(gt)

	matrix.AddTestCase(
		"HonestClaim",
		nil,
		LatestForkOnly,
		runSequenceWindowExpireTest,
		ExpectNoError(),
	)
	matrix.AddTestCase(
		"JunkClaim",
		nil,
		LatestForkOnly,
		runSequenceWindowExpireTest,
		ExpectNoError(),
		WithL2Claim(common.HexToHash("0xdeadbeef")),
	)
}
