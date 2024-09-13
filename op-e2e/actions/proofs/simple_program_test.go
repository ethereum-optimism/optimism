package proofs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/actions"
	"github.com/ethereum-optimism/optimism/op-program/client/claim"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func runSimpleProgramTest(gt *testing.T, testCfg *TestCfg[any]) {
	t := actions.NewDefaultTesting(gt)
	env := NewL2FaultProofEnv(t, testCfg, NewTestParams(), NewBatcherCfg())

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

	env.RunFaultProofProgram(t, l2SafeHead.Number.Uint64(), testCfg.CheckResult, testCfg.InputParams...)
}

func Test_ProgramAction_SimpleEmptyChain(gt *testing.T) {
	matrix := NewMatrix[any]()
	defer matrix.Run(gt)

	matrix.AddTestCase(
		"HonestClaim",
		nil,
		LatestForkOnly,
		runSimpleProgramTest,
		ExpectNoError(),
	)
	matrix.AddTestCase(
		"JunkClaim",
		nil,
		LatestForkOnly,
		runSimpleProgramTest,
		ExpectError(claim.ErrClaimNotValid),
		WithL2Claim(common.HexToHash("0xdeadbeef")),
	)
}
