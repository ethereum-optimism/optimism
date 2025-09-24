package proofs_test

import (
	"testing"

	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func Test_ProgramAction_OsakaForkAfterGenesis(gt *testing.T) {
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
				dp.L1OsakaTimeOffset = ptr(hexutil.Uint64(24))
			},
		)

		miner, _, verifier, sequencer, _ := env.Miner, env.Batcher, env.Sequencer, env.Sequencer, env.Engine

		// Helper function to check L1 blob base fee consistency between L1 and L2
		// checkL1BlockBlobBaseFee := func(t actionsHelpers.StatefulTesting, l2Block *types.Block, l1BlockID eth.BlockID) {
		// 	// TODO get the blob fee via L1 RPC, and also via L2 RPC if possible.
		// 	// Also query the L1Block contract and check these value are all consistent.
		// }

		// Start nodes
		sequencer.ActL2PipelineFull(t)
		verifier.ActL2PipelineFull(t)

		// Build L1 blocks to trigger Fusaka activation
		miner.ActEmptyBlock(t) // block 1
		miner.ActEmptyBlock(t) // block 2 - Fusaka activates here

		block := miner.L1Chain().CurrentBlock()
		require.True(t, env.Sd.L1Cfg.Config.IsOsaka(block.Number, block.Time), "Osaka not active")

		// Build an empty L2 block which has a pre-Fusaka L1 origin, and check the blob fee is correct
		sequencer.ActL2EmptyBlock(t)

		// Build L2 unsafe chain and batch it to L1
		sequencer.ActL1HeadSignal(t)
		sequencer.ActBuildToL1Head(t)
		env.BatchMineAndSync(t)

		// Advance L2 chain until L1 origin has Fusaka active
		sequencer.ActBuildToL1Head(t)

		// Check that the L1 origin is now a Fusaka block, and that the blob fee is correct
		l2Block := verifier.SyncStatus().UnsafeL2

		l1BlockHeader := miner.L1Chain().GetHeaderByHash(l2Block.L1Origin.Hash)
		require.True(t, env.Sd.L1Cfg.Config.IsOsaka(l1BlockHeader.Number, l1BlockHeader.Time), "Osaka not active at l1 origin")

		// checkL1BlockBlobBaseFee(t, fullL2Block, l2Block.L1Origin) // TODO

		// Final sync
		sequencer.ActL1HeadSignal(t)
		sequencer.ActBuildToL1Head(t)
		env.BatchMineAndSync(t)

		// Run fault proof program
		safeL2Head := verifier.SyncStatus().SafeL2
		env.RunFaultProofProgramFromGenesis(t, safeL2Head.Number, testCfg.CheckResult, testCfg.InputParams...)
	}

	matrix := helpers.NewMatrix[any]()
	defer matrix.Run(gt)
	matrix.AddDefaultTestCases(nil, helpers.NewForkMatrix(helpers.Holocene, helpers.LatestFork), runL1FusakaTest)
}
