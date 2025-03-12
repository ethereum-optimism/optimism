package proofs_test

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func runSimpleProgramTest(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	t := actionsHelpers.NewDefaultTesting(gt)
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

	// Build an empty block on L2
	env.Sequencer.ActL2StartBlock(t)
	env.Sequencer.ActL2EndBlock(t)

	// Instruct the batcher to submit the block to L1, and include the transaction.
	env.Batcher.ActSubmitAll(t)
	env.Miner.ActL1StartBlock(12)(t)
	env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
	env.Miner.ActL1EndBlock(t)

	// Finalize the block with the batch on L1.
	env.Miner.ActL1SafeNext(t)
	env.Miner.ActL1FinalizeNext(t)

	// Instruct the sequencer to derive the L2 chain from the data on L1 that the batcher just posted.
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	l1Head := env.Miner.L1Chain().CurrentBlock()
	l2SafeHead := env.Engine.L2Chain().CurrentSafeBlock()

	// Ensure there is only 1 block on L1.
	require.Equal(t, uint64(1), l1Head.Number.Uint64())
	// Ensure the block is marked as safe before we attempt to fault prove it.
	require.Equal(t, uint64(1), l2SafeHead.Number.Uint64())

	env.RunFaultProofProgram(t, l2SafeHead.Number.Uint64(), testCfg.CheckResult, testCfg.InputParams...)
}

func Test_ProgramAction_SimpleEmptyChain(gt *testing.T) {
	matrix := helpers.NewMatrix[any]()
	defer matrix.Run(gt)

	matrix.AddDefaultTestCases(
		nil,
		helpers.LatestForkOnly,
		runSimpleProgramTest,
	)
}
