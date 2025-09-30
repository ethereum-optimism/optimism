package proofs_test

import (
	"math/big"
	"testing"

	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	legacybindings "github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/stretchr/testify/require"
)

func Test_ProgramAction_OsakaForkAfterGenesis(gt *testing.T) {
	runL1FusakaTest := func(gt *testing.T, testCfg *helpers.TestCfg[any]) {
		t := actionsHelpers.NewDefaultTesting(gt)

		// Create test environment with Fusaka activation
		env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(),
			helpers.NewBatcherCfg(
				func(c *actionsHelpers.BatcherCfg) {
					c.DataAvailabilityType = batcherFlags.BlobsType
				},
			),
			func(dp *genesis.DeployConfig) {
				dp.L1PragueTimeOffset = ptr(hexutil.Uint64(0))
				dp.L1OsakaTimeOffset = ptr(hexutil.Uint64(24))
				// TODO add the BPO forks
				dp.L1GenesisBlockExcessBlobGas = ptr(hexutil.Uint64(1e8)) // Jack up the blob market so we can test the blob fee calculation
			},
		)

		miner, sequencer := env.Miner, env.Sequencer

		// Bind to L1Block contract on L2
		l1BlockContract, err := legacybindings.NewL1Block(predeploys.L1BlockAddr, env.Engine.EthClient())
		require.NoError(t, err)

		atBlockWithHash := func(hash common.Hash) *bind.CallOpts {
			return &bind.CallOpts{
				BlockHash: hash,
			}
		}

		// requireConsistentBlobBaseFee requires the blob base fee to be consistent between
		// the L1 Origin block (computed using the excess blob gas and l1 chain config)
		// and the L1 Block contract on L2 (acessed with a contract method call).
		requireConsistentBlobBaseFee := func(t actionsHelpers.StatefulTesting, l2Block eth.L2BlockRef, expectL1OriginToBeFusaka bool) {
			bbfL2, err := l1BlockContract.BlobBaseFee(atBlockWithHash(l2Block.Hash))
			require.NoError(t, err)

			l1Origin := miner.L1Chain().GetHeaderByHash(l2Block.L1Origin.Hash)
			if expectL1OriginToBeFusaka {
				require.True(t, env.Sd.L1Cfg.Config.IsOsaka(l1Origin.Number, l1Origin.Time), "Osaka not active at l1 origin %d, time %d", l1Origin.Number, l1Origin.Time)
			} else {
				require.False(t, env.Sd.L1Cfg.Config.IsOsaka(l1Origin.Number, l1Origin.Time), "Osaka should not be active at l1 origin %d, time %d", l1Origin.Number, l1Origin.Time)
			}
			bbfL1 := eip4844.CalcBlobFee(env.Sd.L1Cfg.Config, l1Origin)

			require.True(t, bbfL2.Cmp(bbfL1) == 0,
				"blob base fee does not match, bbfL2=%d, bbfL1=%d, l1BlockNum=%d, l2BlockNum=%d", bbfL2, bbfL1, l1Origin.Number, l2Block.Number)

			require.True(t, bbfL2.Cmp(big.NewInt(1)) > 0,
				"blob base fee is unrealistically low and doesn't exercise the blob fee calculation")
		}

		// Build L1 blocks to trigger Fusaka activation
		// TODO in the current version of op-geth, the blob parameters don't change between Prague and Osaka.
		// So this test is no useful until we can activate different blob parameters.
		l1Block := miner.ActBuildToOsaka(t)
		require.Equal(t, uint64(2), l1Block.Number().Uint64())

		// Build an empty L2 block which has a pre-Fusaka L1 origin, and check the blob fee is correct
		sequencer.ActL2EmptyBlock(t)
		l2Block := sequencer.SyncStatus().UnsafeL2
		require.Equal(t, uint64(1), l2Block.Number)
		requireConsistentBlobBaseFee(t, l2Block, false)

		// Advance L2 chain until L1 origin has Fusaka activ
		sequencer.ActL1HeadSignal(t)
		sequencer.ActBuildToL1HeadUnsafe(t)

		// Check that the L1 origin is now a Fusaka block, and that the blob fee is correct
		l2Block = sequencer.L2Unsafe()
		require.Greater(t, l2Block.Number, uint64(1))
		requireConsistentBlobBaseFee(t, l2Block, true)

		// Final sync
		env.BatchMineAndSync(t)

		// Run fault proof program
		safeL2Head := sequencer.L2Safe()
		env.RunFaultProofProgramFromGenesis(t, safeL2Head.Number, testCfg.CheckResult, testCfg.InputParams...)
	}

	matrix := helpers.NewMatrix[any]()
	defer matrix.Run(gt)
	matrix.AddDefaultTestCases(nil, helpers.NewForkMatrix(helpers.LatestFork), runL1FusakaTest)
}
