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

// runEndOfSourceOutputRootTest tests FPP behavior when the kona derivation pipeline hits EndOfSource
// before deriving any blocks beyond the agreed prestate. The FPP should return the agreed prestate's
// output root (i.e. the safe head output root), not a zero output root.
func runEndOfSourceOutputRootTest(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg())

	// Build 1 L2 block, batch it to L1, and sync
	env.Sequencer.ActL2StartBlock(t)
	env.Sequencer.ActL2EndBlock(t)
	env.BatchMineAndSync(t)

	l2SafeHead := env.Engine.L2Chain().CurrentSafeBlock()
	safeHeadNum := bigs.Uint64Strict(l2SafeHead.Number)
	require.Equal(t, uint64(1), safeHeadNum)

	// Get the output root at the safe head via the rollup client
	rollupClient := env.Sequencer.RollupClient()
	safeHeadOutput, err := rollupClient.OutputAtBlock(t.Ctx(), safeHeadNum)
	require.NoError(t, err)

	safeHeadOutputRoot := common.Hash(safeHeadOutput.OutputRoot)
	safeHeadHash := safeHeadOutput.BlockRef.Hash

	params := []helpers.FixtureInputParam{
		func(f *helpers.FixtureInputs) {
			f.L2OutputRoot = safeHeadOutputRoot
		},
		func(f *helpers.FixtureInputs) {
			f.L2Head = safeHeadHash
		},
		helpers.WithL2BlockNumber(safeHeadNum + 1),
		helpers.WithL2Claim(safeHeadOutputRoot),
	}
	params = append(params, testCfg.InputParams...)

	env.RunFaultProofProgram(t, safeHeadNum, testCfg.CheckResult, params...)
}

// Test_ProgramAction_EndOfSourceOutputRoot verifies that the FPP correctly returns the agreed
// prestate output root when EndOfSource is reached before deriving any new blocks.
func Test_ProgramAction_EndOfSourceOutputRoot(gt *testing.T) {
	matrix := helpers.NewMatrix[any]()
	defer matrix.Run(gt)

	matrix.AddTestCase(
		"HonestClaim",
		nil,
		helpers.LatestForkOnly,
		runEndOfSourceOutputRootTest,
		helpers.ExpectNoError(),
	)
	matrix.AddTestCase(
		"JunkClaim",
		nil,
		helpers.LatestForkOnly,
		runEndOfSourceOutputRootTest,
		helpers.ExpectError(claim.ErrClaimNotValid),
		helpers.WithL2Claim(common.HexToHash("0xdeadbeef")),
	)
	matrix.AddTestCase(
		"ZeroClaim",
		nil,
		helpers.LatestForkOnly,
		runEndOfSourceOutputRootTest,
		helpers.ExpectError(claim.ErrClaimNotValid),
		helpers.WithL2Claim(common.Hash{}),
	)
}
