package proofs

import (
	"encoding/binary"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-program/client/claim"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/stretchr/testify/require"
)

func setMinBaseFeeViaSystemConfig(t actionsHelpers.Testing, env *helpers.L2FaultProofEnv, minBaseFee uint64) {
	// Create system config contract binding
	systemConfig, err := bindings.NewSystemConfig(env.Sd.RollupCfg.L1SystemConfigAddress, env.Miner.EthClient())
	require.NoError(t, err)

	// Create transactor for the deployer (system config owner)
	deployerTx, err := bind.NewKeyedTransactorWithChainID(env.Dp.Secrets.Deployer, env.Sd.RollupCfg.L1ChainID)
	require.NoError(t, err)
	t.Logf("Setting min base fee on L1: minBaseFee=%d", minBaseFee)

	// Mine the L1 transaction
	env.Miner.ActL1StartBlock(12)(t)
	_, err = systemConfig.SetMinBaseFee(deployerTx, minBaseFee)
	require.NoError(t, err, "SetMinBaseFee transaction failed")
	env.Miner.ActL1IncludeTx(env.Dp.Addresses.Deployer)(t)
	env.Miner.ActL1EndBlock(t)
}

func Test_ProgramAction_JovianActivation(gt *testing.T) {

	runJovianDerivationTest := func(gt *testing.T, testCfg *helpers.TestCfg[any], genesisConfigFn func(*genesis.DeployConfig), jovianAtGenesis bool, minBaseFee uint64) {
		t := actionsHelpers.NewDefaultTesting(gt)

		env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), genesisConfigFn)

		t.Logf("L2 Genesis Time: %d, JovianTime: %d ", env.Sequencer.RollupCfg.Genesis.L2Time, *env.Sequencer.RollupCfg.JovianTime)

		if jovianAtGenesis {
			// Verify Jovian is active at genesis
			require.True(t, env.Sequencer.RollupCfg.IsJovian(env.Sequencer.RollupCfg.Genesis.L2Time), "Jovian should be active at genesis")
		} else {
			// If Jovian is not activated at genesis, build some blocks up to the activation block
			// and verify that the extra data is Holocene
			for env.Engine.L2Chain().CurrentBlock().Time < *env.Sequencer.RollupCfg.JovianTime {
				b := env.Engine.L2Chain().GetBlockByHash(env.Sequencer.L2Unsafe().Hash)
				expectedHoloceneExtraData := eip1559.EncodeHoloceneExtraData(250, 6)
				require.Equal(t, expectedHoloceneExtraData, b.Extra(), "extra data should match Holocene format")
				env.Sequencer.ActL2EmptyBlock(t)
			}
		}

		// Build the activation block
		env.Sequencer.ActL2EmptyBlock(t)
		activationBlock := env.Engine.L2Chain().GetBlockByHash(env.Sequencer.L2Unsafe().Hash)
		require.Equal(t, eip1559.EncodeJovianExtraData(250, 6, 0), activationBlock.Extra(), "activation block should have Jovian extraData")
		require.Greater(t, activationBlock.BaseFee().Uint64(), uint64(0), "activation block should have a base fee above the minimum base fee since it was just enabled")

		// Set the minimum base fee
		setMinBaseFeeViaSystemConfig(t, env, minBaseFee)

		// Allow L1->L2 derivation to propagate the SystemConfig change
		env.Sequencer.ActL1HeadSignal(t)
		env.Sequencer.ActL2PipelineFull(t)

		// Build L2 blocks up to the L1 origin that includes the SystemConfig change
		env.Sequencer.ActBuildToL1Head(t)

		// Build activation+1 block
		env.Sequencer.ActL2EmptyBlock(t)
		nextBlock := env.Engine.L2Chain().GetBlockByHash(env.Sequencer.L2Unsafe().Hash)

		// Extract minimum base fee from extradata
		actualMinBaseFee := binary.BigEndian.Uint64(nextBlock.Extra()[9:17])
		require.Equal(t, minBaseFee, actualMinBaseFee, "minimum base fee should be equal to the set minimum base fee")

		expectedJovianExtraDataWithMinFee := eip1559.EncodeJovianExtraData(250, 6, minBaseFee)
		require.Equal(t, expectedJovianExtraDataWithMinFee, nextBlock.Extra(), "block should have updated Jovian extraData with min base fee")

		// assert that the block's base fee is greater than or equal to the minimum
		require.GreaterOrEqual(t, nextBlock.BaseFee().Uint64(), actualMinBaseFee, "base fee should be >= minimum base fee")

		if !jovianAtGenesis {
			// Verify Jovian fork activation occurred by checking for the activation log
			jovianRecs := env.Logs.FindLogs(
				testlog.NewMessageContainsFilter("Detected hardfork activation block"),
				testlog.NewAttributesFilter("role", "sequencer"),
				testlog.NewAttributesFilter("forkName", "jovian"),
			)
			require.Len(t, jovianRecs, 1, "Jovian fork should be detected and activated exactly once")
		}

		env.BatchMineAndSync(t)

		l2SafeHead := env.Sequencer.L2Safe()
		t.Logf("Safe head block number: %d, timestamp: %d", l2SafeHead.Number, l2SafeHead.Time)
		require.True(t, l2SafeHead.Number >= uint64(0), "safe head should progress")

		env.RunFaultProofProgramFromGenesis(t, l2SafeHead.Number, testCfg.CheckResult, testCfg.InputParams...)
	}

	tests := map[string]struct {
		genesisConfigFn func(*genesis.DeployConfig)
		jovianAtGenesis bool
		minBaseFee      uint64
	}{
		"JovianActivationAfterGenesis": {
			genesisConfigFn: func(dc *genesis.DeployConfig) {
				// Activate Isthmus at genesis
				zero := hexutil.Uint64(0)
				dc.L2GenesisIsthmusTimeOffset = &zero
				// Then set Jovian at 10s
				ten := hexutil.Uint64(10)
				dc.L2GenesisJovianTimeOffset = &ten
			},
			jovianAtGenesis: false,
			minBaseFee:      0,
		},
		"JovianActivationAtGenesisZeroMinBaseFee": {
			genesisConfigFn: func(dc *genesis.DeployConfig) {
				zero := hexutil.Uint64(0)
				dc.L2GenesisJovianTimeOffset = &zero
			},
			jovianAtGenesis: true,
			minBaseFee:      0,
		},
		"JovianActivationAtGenesis1GweiMinBaseFee": {
			genesisConfigFn: func(dc *genesis.DeployConfig) {
				zero := hexutil.Uint64(0)
				dc.L2GenesisJovianTimeOffset = &zero
			},
			jovianAtGenesis: true,
			minBaseFee:      1,
		},
		/*"JovianActivationAtGenesis5GweiMinBaseFee": {
			genesisConfigFn: func(dc *genesis.DeployConfig) {
				zero := hexutil.Uint64(0)
				dc.L2GenesisJovianTimeOffset = &zero
				// Set 5 Gwei minimum base fee
				fiveGwei := hexutil.Uint64(5_000_000_000) // 5e9
				dc.SystemConfigStartBlock = 0
				_ = fiveGwei // Will be used for minimum base fee configuration
			},
			jovianAtGenesis: true,
		},*/
	}

	for name, tt := range tests {
		gt.Run(name, func(t *testing.T) {
			matrix := helpers.NewMatrix[any]()
			defer matrix.Run(t)

			matrix.AddTestCase(
				"HonestClaim-"+name,
				nil,
				helpers.NewForkMatrix(helpers.Isthmus),
				func(gt *testing.T, testCfg *helpers.TestCfg[any]) {
					runJovianDerivationTest(gt, testCfg, tt.genesisConfigFn, tt.jovianAtGenesis, tt.minBaseFee)
				},
				helpers.ExpectNoError(),
			)
			matrix.AddTestCase(
				"JunkClaim-"+name,
				nil,
				helpers.NewForkMatrix(helpers.Isthmus),
				func(gt *testing.T, testCfg *helpers.TestCfg[any]) {
					runJovianDerivationTest(gt, testCfg, tt.genesisConfigFn, tt.jovianAtGenesis, tt.minBaseFee)
				},
				helpers.ExpectError(claim.ErrClaimNotValid),
				helpers.WithL2Claim(common.HexToHash("0xdeadbeef")),
			)
		})
	}
}
