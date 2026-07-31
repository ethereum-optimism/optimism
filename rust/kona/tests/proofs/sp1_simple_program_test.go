package proofs_test

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func runSP1RangeSimpleProgramTest(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	if helpers.SP1RangeExecutorPath() == "" {
		t.Skip("KONA_SP1_RANGE_EXECUTOR_PATH not set; build the range-executor with built ELFs " +
			"(see rust/kona/sp1) to run the SP1 execute action tests")
	}

	testSetup := func(dc *genesis.DeployConfig) {
		dc.L1PragueTimeOffset = ptr(hexutil.Uint64(0))
		// Set non-trivial excess blob gas so that the L1 miner's blob logic is
		// properly tested.
		dc.L1GenesisBlockExcessBlobGas = ptr(hexutil.Uint64(1e8))
	}
	bcfg := helpers.NewBatcherCfg(func(c *actionsHelpers.BatcherCfg) {
		c.DataAvailabilityType = flags.BlobsType
	})
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), bcfg, testSetup)

	// Build an empty block on L2.
	env.Sequencer.ActL2StartBlock(t)
	env.Sequencer.ActL2EndBlock(t)

	// Submit the block to L1 and include the batcher transaction.
	env.Batcher.ActSubmitAll(t)
	env.Miner.ActL1StartBlock(12)(t)
	env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
	env.Miner.ActL1EndBlock(t)

	// Finalize the block with the batch on L1.
	env.Miner.ActL1SafeNext(t)
	env.Miner.ActL1FinalizeNext(t)

	// Derive the L2 chain from the data on L1.
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	l1Head := env.Miner.L1Chain().CurrentBlock()
	l2SafeHead := env.Engine.L2Chain().CurrentSafeBlock()
	require.Equal(t, uint64(1), bigs.Uint64Strict(l1Head.Number))
	require.Equal(t, uint64(1), bigs.Uint64Strict(l2SafeHead.Number))

	// Run the kona-sp1 range guest in SP1 execute mode over the 0 -> 1 transition.
	env.RunSP1RangeProgram(t, bigs.Uint64Strict(l2SafeHead.Number), testCfg.CheckResult, testCfg.InputParams...)
}

// TestSP1RangeSimpleEmptyChain runs the kona-sp1 range guest in SP1 execute mode against a single
// real state transition, covering both the honest-claim and invalid-claim paths.
func TestSP1RangeSimpleEmptyChain(gt *testing.T) {
	matrix := helpers.NewMatrix[any]()
	matrix.AddTestCase(
		"HonestClaim",
		nil,
		helpers.LatestForkOnly,
		runSP1RangeSimpleProgramTest,
		helpers.ExpectNoError(),
	).AddTestCase(
		"JunkClaim",
		nil,
		helpers.LatestForkOnly,
		runSP1RangeSimpleProgramTest,
		helpers.ExpectError(helpers.ErrClaimNotValid),
		helpers.WithCorruptClaim(),
	)
	matrix.Run(gt)
}
