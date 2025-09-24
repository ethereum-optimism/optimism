package proofs_test

import (
	"testing"

	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	legacybindings "github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/stretchr/testify/require"
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
				dp.L1FusakaTimeOffset = ptr(hexutil.Uint64(24))
			},
		)

		miner, batcher, verifier, sequencer, engine := env.Miner, env.Batcher, env.Sequencer, env.Sequencer, env.Engine

		l1Block, err := legacybindings.NewL1Block(predeploys.L1BlockAddr, engine.EthClient())
		require.NoError(t, err)

		// Helper function to check L1 blob base fee consistency between L1 and L2
		checkL1BlockBlobBaseFee := func(t actionsHelpers.StatefulTesting, l2Block eth.L2BlockRef) {
			l1BlockID := l2Block.L1Origin
			l1BlockHeader := miner.L1Chain().GetHeaderByHash(l1BlockID.Hash)
			expectedBbf := eth.CalcBlobFeeDefault(l1BlockHeader)
			upstreamExpectedBbf := eip4844.CalcBlobFee(env.Sd.L1Cfg.Config, l1BlockHeader)
			require.Equal(t, expectedBbf.Uint64(), upstreamExpectedBbf.Uint64(), "expected blob base fee should match upstream calculation")
			bbf, err := l1Block.BlobBaseFee(&bind.CallOpts{BlockHash: l2Block.Hash})
			require.NoError(t, err, "failed to get blob base fee")
			require.Equal(t, expectedBbf.Uint64(), bbf.Uint64(), "l1Block blob base fee does not match expectation, l1BlockNum %d, l2BlockNum %d", l1BlockID.Number, l2Block.Number)
		}

		// Start nodes
		sequencer.ActL2PipelineFull(t)
		verifier.ActL2PipelineFull(t)

		// Build L1 blocks to trigger Fusaka activation
		miner.ActEmptyBlock(t) // block 1
		miner.ActEmptyBlock(t) // block 2 - Fusaka activates here

		block := miner.L1Chain().CurrentBlock()
		require.True(t, env.Sd.L1Cfg.Config.IsOsaka(block.Number, block.Time))

		// Build an empty L2 block which has a pre-Fusaka L1 origin, and check the blob fee is correct
		sequencer.ActL2EmptyBlock(t)
		// TODO: When Fusaka is implemented, add fork status validation for pre-Fusaka L1 origin
		checkL1BlockBlobBaseFee(t, verifier.SyncStatus().UnsafeL2)

		// Build L2 unsafe chain and batch it to L1
		sequencer.ActL1HeadSignal(t)
		sequencer.ActBuildToL1Head(t)
		batcher.ActSubmitAll(t)
		miner.ActL1IncludeTx(batcher.BatcherAddr)(t)

		// Sync verifier
		verifier.ActL1HeadSignal(t)
		verifier.ActL2PipelineFull(t)

		// Advance L2 chain until L1 origin has Fusaka active
		sequencer.ActBuildToL1Head(t)

		// Check that the L1 origin is now a Fusaka block, and that the blob fee is correct
		// TODO: When Fusaka is implemented, add fork status validation for Fusaka-active L1 origin
		checkL1BlockBlobBaseFee(t, verifier.SyncStatus().UnsafeL2)

		// Final sync
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
