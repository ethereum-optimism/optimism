package cli

import (
	"math/big"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func getL1RPCFromRunner(runner *CLITestRunner) string {
	v := reflect.ValueOf(runner).Elem()
	field := v.FieldByName("l1RPC")
	if field.IsValid() && field.Kind() == reflect.String {
		return field.String()
	}
	return ""
}

func TestCLIValidate(t *testing.T) {
	runner := NewCLITestRunnerWithNetwork(t)
	workDir := runner.GetWorkDir()
	l1ChainID := uint64(devnet.DefaultChainID)
	l2ChainID := uint256.NewInt(1)

	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)

	t.Run("setup deployment", func(t *testing.T) {
		intent, _ := cliInitIntent(t, runner, l1ChainID, []common.Hash{l2ChainID.Bytes32()})

		if intent.SuperchainRoles == nil {
			intent.SuperchainRoles = &addresses.SuperchainRoles{}
		}

		l1ChainIDBig := big.NewInt(int64(l1ChainID))
		intent.SuperchainRoles.SuperchainProxyAdminOwner = shared.AddrFor(t, dk, devkeys.L1ProxyAdminOwnerRole.Key(l1ChainIDBig))
		intent.SuperchainRoles.SuperchainGuardian = shared.AddrFor(t, dk, devkeys.SuperchainConfigGuardianKey.Key(l1ChainIDBig))
		intent.SuperchainRoles.ProtocolVersionsOwner = shared.AddrFor(t, dk, devkeys.SuperchainDeployerKey.Key(l1ChainIDBig))
		intent.SuperchainRoles.Challenger = shared.AddrFor(t, dk, devkeys.ChallengerRole.Key(l1ChainIDBig))

		for _, chain := range intent.Chains {
			chain.Roles.L1ProxyAdminOwner = shared.AddrFor(t, dk, devkeys.L2ProxyAdminOwnerRole.Key(l1ChainIDBig))
			chain.Roles.L2ProxyAdminOwner = shared.AddrFor(t, dk, devkeys.L2ProxyAdminOwnerRole.Key(l1ChainIDBig))
			chain.Roles.SystemConfigOwner = shared.AddrFor(t, dk, devkeys.SystemConfigOwner.Key(l1ChainIDBig))
			chain.Roles.UnsafeBlockSigner = shared.AddrFor(t, dk, devkeys.SequencerP2PRole.Key(l1ChainIDBig))
			chain.Roles.Batcher = shared.AddrFor(t, dk, devkeys.BatcherRole.Key(l1ChainIDBig))
			chain.Roles.Proposer = shared.AddrFor(t, dk, devkeys.ProposerRole.Key(l1ChainIDBig))
			chain.Roles.Challenger = shared.AddrFor(t, dk, devkeys.ChallengerRole.Key(l1ChainIDBig))

			chain.BaseFeeVaultRecipient = shared.AddrFor(t, dk, devkeys.BaseFeeVaultRecipientRole.Key(l1ChainIDBig))
			chain.L1FeeVaultRecipient = shared.AddrFor(t, dk, devkeys.L1FeeVaultRecipientRole.Key(l1ChainIDBig))
			chain.SequencerFeeVaultRecipient = shared.AddrFor(t, dk, devkeys.SequencerFeeVaultRecipientRole.Key(l1ChainIDBig))
			chain.OperatorFeeVaultRecipient = shared.AddrFor(t, dk, devkeys.OperatorFeeVaultRecipientRole.Key(l1ChainIDBig))

			chain.Eip1559DenominatorCanyon = standard.Eip1559DenominatorCanyon
			chain.Eip1559Denominator = standard.Eip1559Denominator
			chain.Eip1559Elasticity = standard.Eip1559Elasticity
		}
		require.NoError(t, intent.WriteToFile(filepath.Join(workDir, "intent.toml")))

		// Apply deployment
		// Note: Validation is skipped for unsupported chain IDs (like test chains).
		// We verify deployment succeeded, then test validation separately below.
		runner.ExpectSuccessWithNetwork(t, []string{
			"apply",
			"--deployment-target", "live",
			"--workdir", workDir,
		}, nil)

		st, err := pipeline.ReadState(workDir)
		require.NoError(t, err, "State should be readable after apply")
		require.NotNil(t, st.AppliedIntent, "Applied intent should exist")
		require.Len(t, st.Chains, 1, "Should have one chain deployed")

	})

	t.Run("validate all chains", func(t *testing.T) {
		output := runner.ExpectSuccess(t, []string{
			"validate", "auto",
			"--l1-rpc-url", getL1RPCFromRunner(runner),
			"--workdir", workDir,
		}, nil)

		require.Contains(t, output, "Validating chain", "Should show validation progress")
		require.Contains(t, output, "Contract validated", "Should validate contracts")
	})

	t.Run("validate specific chain", func(t *testing.T) {
		st, err := pipeline.ReadState(workDir)
		require.NoError(t, err)
		require.Len(t, st.Chains, 1)

		chainID := st.Chains[0].ID.Hex()

		output := runner.ExpectSuccess(t, []string{
			"validate", "auto",
			"--l1-rpc-url", getL1RPCFromRunner(runner),
			"--workdir", workDir,
			chainID,
		}, nil)

		require.Contains(t, output, "Validating chain", "Should show validation progress")
		require.Contains(t, output, chainID, "Should validate the specified chain")
	})

	t.Run("validate with fail flag", func(t *testing.T) {
		output := runner.ExpectSuccess(t, []string{
			"validate", "auto",
			"--l1-rpc-url", getL1RPCFromRunner(runner),
			"--workdir", workDir,
			"--fail",
		}, nil)

		require.Contains(t, output, "Validating chain", "Should show validation progress")
	})

	t.Run("validate fails when no applied intent", func(t *testing.T) {
		emptyWorkDir := runner.GetWorkDir() + "_empty"
		runner.ExpectSuccess(t, []string{
			"init",
			"--l1-chain-id", "11155111",
			"--l2-chain-ids", "1",
			"--workdir", emptyWorkDir,
		}, nil)

		runner.ExpectErrorContains(t, []string{
			"validate", "auto",
			"--l1-rpc-url", getL1RPCFromRunner(runner),
			"--workdir", emptyWorkDir,
		}, nil, "cannot validate: no applied intent found")
	})

	t.Run("validate fails for non-existent chain", func(t *testing.T) {
		nonExistentChainID := "0x" + strings.Repeat("ff", 32)

		runner.ExpectErrorContains(t, []string{
			"validate", "auto",
			"--l1-rpc-url", getL1RPCFromRunner(runner),
			"--workdir", workDir,
			nonExistentChainID,
		}, nil, "chain with ID")
	})

	t.Run("validate requires l1-rpc-url", func(t *testing.T) {
		runner.ExpectErrorContains(t, []string{
			"validate", "auto",
			"--workdir", workDir,
		}, nil, "l1-rpc-url")
	})
}
