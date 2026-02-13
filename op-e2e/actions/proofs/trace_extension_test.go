package proofs

import (
	"testing"

	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-program/client/claim"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func runSafeHeadTraceExtensionTest(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg())

	// Build an empty block on L2
	env.Sequencer.ActL2StartBlock(t)
	env.Sequencer.ActL2EndBlock(t)

	env.BatchMineAndSync(t)

	l1Head := env.Miner.L1Chain().CurrentBlock()
	l2SafeHead := env.Engine.L2Chain().CurrentSafeBlock()

	// Ensure there is only 1 block on L1.
	require.Equal(t, uint64(1), bigs.Uint64Strict(l1Head.Number))
	// Ensure the block is marked as safe before we attempt to fault prove it.
	require.Equal(t, uint64(1), bigs.Uint64Strict(l2SafeHead.Number))

	// Set claimed L2 block number to be past the actual safe head (still using the safe head output as the claim)
	params := []helpers.FixtureInputParam{helpers.WithL2BlockNumber(bigs.Uint64Strict(l2SafeHead.Number) + 1)}
	params = append(params, testCfg.InputParams...)
	env.RunFaultProofProgram(t, bigs.Uint64Strict(l2SafeHead.Number), testCfg.CheckResult, params...)
}

func runTraceExtensionLeafTest(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg())

	// Build two empty blocks on L2.
	// We want a transition where the agreed output root is at block N, but the claim targets block N+1,
	// and maliciously repeats the output root from block N. This is invalid, and used to be accepted
	// by kona due to faulty trace-extension short-circuiting based only on root equality.
	env.Sequencer.ActL2StartBlock(t)
	env.Sequencer.ActL2EndBlock(t)
	env.Sequencer.ActL2StartBlock(t)
	env.Sequencer.ActL2EndBlock(t)

	env.BatchMineAndSync(t)

	l1Head := env.Miner.L1Chain().CurrentBlock()
	l2SafeHead := env.Engine.L2Chain().CurrentSafeBlock()

	// Ensure there is only 1 block on L1.
	require.Equal(t, uint64(1), bigs.Uint64Strict(l1Head.Number))
	// Ensure the block is marked as safe before we attempt to fault prove it.
	require.Equal(t, uint64(2), bigs.Uint64Strict(l2SafeHead.Number))

	// Prove the transition 1 -> 2. This means the agreed output root is at block 1, and the claim
	// targets block 2.
	env.RunFaultProofProgram(t, 2, testCfg.CheckResult, testCfg.InputParams...)
}

func runTraceExtensionRepeatedRootAtNextBlockTest(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg())

	// Build two empty blocks on L2.
	env.Sequencer.ActL2StartBlock(t)
	env.Sequencer.ActL2EndBlock(t)
	env.Sequencer.ActL2StartBlock(t)
	env.Sequencer.ActL2EndBlock(t)

	env.BatchMineAndSync(t)

	l1Head := env.Miner.L1Chain().CurrentBlock()
	l2SafeHead := env.Engine.L2Chain().CurrentSafeBlock()

	// Ensure there is only 1 block on L1.
	require.Equal(t, uint64(1), bigs.Uint64Strict(l1Head.Number))
	// Ensure the block is marked as safe before we attempt to fault prove it.
	require.Equal(t, uint64(2), bigs.Uint64Strict(l2SafeHead.Number))

	rollupClient := env.Sequencer.L2Verifier.RollupClient()
	out1, err := rollupClient.OutputAtBlock(t.Ctx(), 1)
	require.NoError(t, err)

	params := []helpers.FixtureInputParam{
		helpers.WithL2Claim(common.Hash(out1.OutputRoot)),
	}
	params = append(params, testCfg.InputParams...)

	// Prove the transition 1 -> 2, but maliciously repeat the output root from block 1 at block 2.
	env.RunFaultProofProgram(t, 2, testCfg.CheckResult, params...)
}

// Test_ProgramAction_SafeHeadTraceExtension checks that op-program correctly handles the trace extension case where
// the claimed l2 block number is after the safe head. The honest actor should repeat the output root from the safe head
// and op-program should consider it valid even though the claimed l2 block number is not reached.
// Output roots other than from the safe head should be invalid if the claimed l2 block number is not reached.
func Test_ProgramAction_SafeHeadTraceExtension(gt *testing.T) {
	matrix := helpers.NewMatrix[any]()
	defer matrix.Run(gt)

	matrix.AddTestCase(
		"HonestClaim",
		nil,
		helpers.LatestForkOnly,
		runSafeHeadTraceExtensionTest,
		helpers.ExpectNoError(),
	)
	matrix.AddTestCase(
		"JunkClaim",
		nil,
		helpers.LatestForkOnly,
		runSafeHeadTraceExtensionTest,
		helpers.ExpectError(claim.ErrClaimNotValid),
		helpers.WithL2Claim(common.HexToHash("0xdeadbeef")),
	)
}

// Test_ProgramAction_TraceExtensionLeaf checks that both op-program and kona reject a claim that
// maliciously repeats the agreed output root at a later (reachable) block number.
func Test_ProgramAction_TraceExtensionLeaf(gt *testing.T) {
	matrix := helpers.NewMatrix[any]()
	defer matrix.Run(gt)

	matrix.AddTestCase(
		"HonestTransition",
		nil,
		helpers.LatestForkOnly,
		runTraceExtensionLeafTest,
		helpers.ExpectNoError(),
	)
	matrix.AddTestCase(
		"RepeatedRootAtNextBlock",
		nil,
		helpers.LatestForkOnly,
		runTraceExtensionRepeatedRootAtNextBlockTest,
		helpers.ExpectError(claim.ErrClaimNotValid),
	)
}
