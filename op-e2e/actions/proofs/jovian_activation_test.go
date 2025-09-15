package proofs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-program/client/claim"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/stretchr/testify/require"
)

func Test_ProgramAction_JovianActivation(gt *testing.T) {

	runJovianDerivationTest := func(gt *testing.T, testCfg *helpers.TestCfg[any], genesisConfigFn func(*genesis.DeployConfig), jovianAtGenesis bool) {
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
		expectedJovianExtraData := eip1559.EncodeJovianExtraData(250, 6, 0)
		require.Equal(t, expectedJovianExtraData, activationBlock.Extra(), "activation block should have Jovian extraData")

		// Build a few more blocks
		for range 10 {
			b := env.Engine.L2Chain().GetBlockByHash(env.Sequencer.L2Unsafe().Hash)
			require.Equal(t, expectedJovianExtraData, b.Extra(), "subsequent blocks should have Jovian extraData")
			// assert that the block's base fee is greater than the minimum
			require.Greater(t, b.BaseFee().Uint64(), uint64(0), "base fee should be > minimum")
			env.Sequencer.ActL2EmptyBlock(t)
		}

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
		},
		"JovianActivationAtGenesis": {
			genesisConfigFn: func(dc *genesis.DeployConfig) {
				zero := hexutil.Uint64(0)
				dc.L2GenesisJovianTimeOffset = &zero
			},
			jovianAtGenesis: true,
		},
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
					runJovianDerivationTest(gt, testCfg, tt.genesisConfigFn, tt.jovianAtGenesis)
				},
				helpers.ExpectNoError(),
			)
			matrix.AddTestCase(
				"JunkClaim-"+name,
				nil,
				helpers.NewForkMatrix(helpers.Isthmus),
				func(gt *testing.T, testCfg *helpers.TestCfg[any]) {
					runJovianDerivationTest(gt, testCfg, tt.genesisConfigFn, tt.jovianAtGenesis)
				},
				helpers.ExpectError(claim.ErrClaimNotValid),
				helpers.WithL2Claim(common.HexToHash("0xdeadbeef")),
			)
		})
	}
}
