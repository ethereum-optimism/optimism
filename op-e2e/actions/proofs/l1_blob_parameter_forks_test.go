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
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// Test_ProgramAction_BlobParameterForks tests the blob base fee calculation for different forks.
func Test_ProgramAction_BlobParameterForks(gt *testing.T) {
	runBlobParameterForksTest := func(gt *testing.T, testCfg *helpers.TestCfg[any]) {
		t := actionsHelpers.NewDefaultTesting(gt)

		// Create test environment with Fusaka activation
		env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(),
			helpers.NewBatcherCfg(
				func(c *actionsHelpers.BatcherCfg) {
					c.DataAvailabilityType = batcherFlags.BlobsType
				},
			),
			func(dp *genesis.DeployConfig) {
				dp.L1CancunTimeOffset = ptr(hexutil.Uint64(0))
				dp.L1PragueTimeOffset = ptr(hexutil.Uint64(12))
				dp.L1OsakaTimeOffset = ptr(hexutil.Uint64(24))
				dp.L1BPO1TimeOffset = ptr(hexutil.Uint64(36))
				dp.L1BPO2TimeOffset = ptr(hexutil.Uint64(48))
				dp.L1BPO3TimeOffset = ptr(hexutil.Uint64(60))
				dp.L1BPO4TimeOffset = ptr(hexutil.Uint64(72))
				dp.L1BlobScheduleConfig = &params.BlobScheduleConfig{
					Cancun: params.DefaultCancunBlobConfig,
					Osaka:  params.DefaultOsakaBlobConfig,
					Prague: params.DefaultPragueBlobConfig,
					BPO1:   params.DefaultBPO1BlobConfig,
					BPO2:   params.DefaultBPO2BlobConfig,
					BPO3:   params.DefaultBPO3BlobConfig,
					BPO4:   params.DefaultBPO4BlobConfig,
				}
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

		// requireConsistentBlobBaseFeeForFork requires the blob base fee to be consistent between
		// the L1 Origin block (computed using the excess blob gas and l1 chain config)
		// and the L1 Block contract on L2 (accessed with a contract method call), for a given fork predicate.
		requireConsistentBlobBaseFeeForFork := func(t actionsHelpers.StatefulTesting, l2Block eth.L2BlockRef, expectActive bool, label string, isActive func(num *big.Int, time uint64) bool) {
			bbfL2, err := l1BlockContract.BlobBaseFee(atBlockWithHash(l2Block.Hash))
			require.NoError(t, err)

			l1Origin := miner.L1Chain().GetHeaderByHash(l2Block.L1Origin.Hash)
			if expectActive {
				require.True(t, isActive(l1Origin.Number, l1Origin.Time), "%s not active at l1 origin %d, time %d", label, l1Origin.Number, l1Origin.Time)
			} else {
				require.False(t, isActive(l1Origin.Number, l1Origin.Time), "%s should not be active at l1 origin %d, time %d", label, l1Origin.Number, l1Origin.Time)
			}
			bbfL1 := eip4844.CalcBlobFee(env.Sd.L1Cfg.Config, l1Origin)

			require.True(t, bbfL2.Cmp(bbfL1) == 0,
				"%s: blob base fee does not match, bbfL2=%d, bbfL1=%d, l1BlockNum=%d, l2BlockNum=%d", label, bbfL2, bbfL1, l1Origin.Number, l2Block.Number)

			require.True(t, bbfL2.Cmp(big.NewInt(1)) > 0,
				"%s: blob base fee is unrealistically low and doesn't exercise the blob fee calculation", label)
		}

		// buildL1ToTime advances L1 with empty blocks until the given fork time.
		buildL1ToTime := func(t actionsHelpers.StatefulTesting, forkTime *uint64) {
			require.NotNil(t, forkTime, "fork time must be configured")
			h := miner.L1Chain().CurrentHeader()
			for h.Time < *forkTime {
				h = miner.ActEmptyBlock(t).Header()
			}
		}

		// Iterate through all forks and assert pre/post activation blob fees match expectations
		cfg := env.Sd.L1Cfg.Config
		forks := []struct {
			label    string
			forkTime *uint64
			isActive func(num *big.Int, time uint64) bool
		}{
			{"Prague", cfg.PragueTime, func(num *big.Int, time uint64) bool { return cfg.IsPrague(num, time) }},
			{"Osaka", cfg.OsakaTime, func(num *big.Int, time uint64) bool { return cfg.IsOsaka(num, time) }},
			{"BPO1", cfg.BPO1Time, func(num *big.Int, time uint64) bool { return cfg.IsBPO1(num, time) }},
			{"BPO2", cfg.BPO2Time, func(num *big.Int, time uint64) bool { return cfg.IsBPO2(num, time) }},
			{"BPO3", cfg.BPO3Time, func(num *big.Int, time uint64) bool { return cfg.IsBPO3(num, time) }},
			{"BPO4", cfg.BPO4Time, func(num *big.Int, time uint64) bool { return cfg.IsBPO4(num, time) }},
		}
		for _, f := range forks {
			// Advance L1 to fork activation
			buildL1ToTime(t, f.forkTime)

			// Build an empty L2 block which still has a pre-fork L1 origin, and check blob fee
			sequencer.ActL2EmptyBlock(t)
			l2Block := sequencer.SyncStatus().UnsafeL2
			requireConsistentBlobBaseFeeForFork(t, l2Block, false, f.label, f.isActive)

			// Advance L2 chain until L1 origin is at/after the fork activation
			sequencer.ActL1HeadSignal(t)
			sequencer.ActBuildToL1HeadUnsafe(t)

			l2Block = sequencer.L2Unsafe()
			require.Greater(t, l2Block.Number, uint64(1))
			requireConsistentBlobBaseFeeForFork(t, l2Block, true, f.label, f.isActive)
		}

		// Final sync
		env.BatchMineAndSync(t)

		// Run fault proof program
		safeL2Head := sequencer.L2Safe()
		env.RunFaultProofProgramFromGenesis(t, safeL2Head.Number, testCfg.CheckResult, testCfg.InputParams...)
	}

	matrix := helpers.NewMatrix[any]()
	matrix.AddDefaultTestCases(nil, helpers.NewForkMatrix(helpers.LatestFork), runBlobParameterForksTest)
	matrix.Run(gt)
}
