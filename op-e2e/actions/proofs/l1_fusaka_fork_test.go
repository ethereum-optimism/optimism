package proofs_test

import (
	"testing"

	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func Test_ProgramAction_FusakaForkAfterGenesis(gt *testing.T) {
	runL1FusakaTest := func(gt *testing.T, testCfg *helpers.TestCfg[any]) {
		t := actionsHelpers.NewDefaultTesting(gt)

		// Create test environment with Fusaka activation
		env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(),
			helpers.NewBatcherCfg(
				func(c *actionsHelpers.BatcherCfg) {
					c.DataAvailabilityType = batcherFlags.CalldataType
				},
			),
			func(dp *genesis.DeployConfig) {
				// TODO: When Fusaka is implemented, change this to dp.L1FusakaTimeOffset
				dp.L1PragueTimeOffset = ptr(hexutil.Uint64(24)) // Activate at second L1 block
			},
		)

		miner, batcher, verifier, sequencer := env.Miner, env.Batcher, env.Sequencer, env.Sequencer

		// Start nodes
		sequencer.ActL2PipelineFull(t)
		verifier.ActL2PipelineFull(t)

		// Build L1 blocks to trigger Fusaka activation
		miner.ActEmptyBlock(t) // block 1
		miner.ActEmptyBlock(t) // block 2 - Fusaka activates here

		// TODO: When Fusaka is implemented, add proper fork validation:
		// require.True(t, env.Sd.L1Cfg.Config.IsFusaka(block.Number, block.Time))

		// Build some L2 blocks and batch them
		sequencer.ActL1HeadSignal(t)
		sequencer.ActBuildToL1Head(t)

		batcher.ActSubmitAll(t)
		miner.ActL1IncludeTx(batcher.BatcherAddr)(t)

		// Sync and verify
		verifier.ActL1HeadSignal(t)
		verifier.ActL2PipelineFull(t)

		// Run fault proof program
		safeL2Head := verifier.SyncStatus().SafeL2
		env.RunFaultProofProgramFromGenesis(t, safeL2Head.Number, testCfg.CheckResult, testCfg.InputParams...)
	}

	matrix := helpers.NewMatrix[any]()
	defer matrix.Run(gt)
	matrix.AddDefaultTestCases(nil, helpers.NewForkMatrix(helpers.Holocene, helpers.LatestFork), runL1FusakaTest)
}
