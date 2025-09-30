package overrides

import (
	"context"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	test "github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/cli"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestGlobalOverrides tests that global deployment overrides are properly applied
func TestGlobalOverrides(t *testing.T) {
	require := require.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lgr, l1RPC, _, _, pk, dk, l1ChainID := test.SetupAnvilTest(t)
	l2ChainID := uint256.NewInt(1)
	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	intent, st := test.NewIntent(t, l1ChainID, dk, l2ChainID, artifacts.DefaultL1ContractsLocator, artifacts.DefaultL2ContractsLocator)

	expectedBaseFeeVaultRecipient := common.HexToAddress("0x0000000000000000000000000000000000000001")
	expectedL1FeeVaultRecipient := common.HexToAddress("0x0000000000000000000000000000000000000002")
	expectedSequencerFeeVaultRecipient := common.HexToAddress("0x0000000000000000000000000000000000000003")
	expectedBaseFeeVaultMinimumWithdrawalAmount := strings.ToLower("0x1BC16D674EC80000")
	expectedBaseFeeVaultWithdrawalNetwork := genesis.FromUint8(0)
	expectedEnableGovernance := false
	expectedGasPriceOracleBaseFeeScalar := uint32(1300)
	expectedEIP1559Denominator := uint64(500)
	expectedUseFaultProofs := false

	intent.GlobalDeployOverrides = map[string]interface{}{
		"l2BlockTime":                         float64(3),
		"baseFeeVaultRecipient":               expectedBaseFeeVaultRecipient,
		"l1FeeVaultRecipient":                 expectedL1FeeVaultRecipient,
		"sequencerFeeVaultRecipient":          expectedSequencerFeeVaultRecipient,
		"baseFeeVaultMinimumWithdrawalAmount": expectedBaseFeeVaultMinimumWithdrawalAmount,
		"baseFeeVaultWithdrawalNetwork":       expectedBaseFeeVaultWithdrawalNetwork,
		"enableGovernance":                    expectedEnableGovernance,
		"gasPriceOracleBaseFeeScalar":         expectedGasPriceOracleBaseFeeScalar,
		"eip1559Denominator":                  expectedEIP1559Denominator,
		"useFaultProofs":                      expectedUseFaultProofs,
	}

	err := deployer.ApplyPipeline(
		ctx,
		deployer.ApplyPipelineOpts{
			DeploymentTarget:   deployer.DeploymentTargetGenesis,
			L1RPCUrl:           l1RPC,
			DeployerPrivateKey: pk,
			Intent:             intent,
			State:              st,
			Logger:             lgr,
			StateWriter:        pipeline.NoopStateWriter(),
			CacheDir:           testCacheDir,
		},
	)
	require.NoError(err)

	// Verify that overrides were applied correctly
	cfg, err := state.CombineDeployConfig(intent, intent.Chains[0], st, st.Chains[0])
	require.NoError(err)
	require.Equal(uint64(3), cfg.L2InitializationConfig.L2CoreDeployConfig.L2BlockTime, "L2 block time should be 3 seconds")
	require.Equal(expectedBaseFeeVaultRecipient, cfg.L2InitializationConfig.L2VaultsDeployConfig.BaseFeeVaultRecipient, "Base Fee Vault Recipient should be the expected address")
	require.Equal(expectedL1FeeVaultRecipient, cfg.L2InitializationConfig.L2VaultsDeployConfig.L1FeeVaultRecipient, "L1 Fee Vault Recipient should be the expected address")
	require.Equal(expectedSequencerFeeVaultRecipient, cfg.L2InitializationConfig.L2VaultsDeployConfig.SequencerFeeVaultRecipient, "Sequencer Fee Vault Recipient should be the expected address")
	require.Equal(expectedBaseFeeVaultMinimumWithdrawalAmount, strings.ToLower(cfg.L2InitializationConfig.L2VaultsDeployConfig.BaseFeeVaultMinimumWithdrawalAmount.String()), "Base Fee Vault Minimum Withdrawal Amount should be the expected value")
	require.Equal(expectedBaseFeeVaultWithdrawalNetwork, cfg.L2InitializationConfig.L2VaultsDeployConfig.BaseFeeVaultWithdrawalNetwork, "Base Fee Vault Withdrawal Network should be the expected value")
	require.Equal(expectedEnableGovernance, cfg.L2InitializationConfig.GovernanceDeployConfig.EnableGovernance, "Governance should be disabled")
	require.Equal(expectedGasPriceOracleBaseFeeScalar, cfg.L2InitializationConfig.GasPriceOracleDeployConfig.GasPriceOracleBaseFeeScalar, "Gas Price Oracle Base Fee Scalar should be the expected value")
	require.Equal(expectedEIP1559Denominator, cfg.L2InitializationConfig.EIP1559DeployConfig.EIP1559Denominator, "EIP-1559 Denominator should be the expected value")
	require.Equal(expectedUseFaultProofs, cfg.UseFaultProofs, "Fault proofs should not be enabled")
}
