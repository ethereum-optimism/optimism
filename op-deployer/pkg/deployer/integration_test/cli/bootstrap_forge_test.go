package cli

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestCLIBootstrapForge tests the bootstrap commands via CLI using Forge
func TestCLIBootstrapForge(t *testing.T) {
	// Use the same chain ID that anvil runs on
	l1ChainID := uint64(devnet.DefaultChainID)
	l1ChainIDBig := big.NewInt(int64(l1ChainID))

	// Get dev keys for role addresses
	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)

	// Get addresses for required roles
	superchainProxyAdminOwner := shared.AddrFor(t, dk, devkeys.L1ProxyAdminOwnerRole.Key(l1ChainIDBig))
	protocolVersionsOwner := shared.AddrFor(t, dk, devkeys.SuperchainDeployerKey.Key(l1ChainIDBig))
	guardian := shared.AddrFor(t, dk, devkeys.SuperchainConfigGuardianKey.Key(l1ChainIDBig))
	challenger := shared.AddrFor(t, dk, devkeys.ChallengerRole.Key(l1ChainIDBig))

	t.Run("bootstrap superchain with forge", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)
		workDir := runner.GetWorkDir()

		superchainOutputFile := filepath.Join(workDir, "bootstrap_superchain_forge.json")

		// Run bootstrap superchain command with --use-forge
		output := runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "superchain",
			"--outfile", superchainOutputFile,
			"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--protocol-versions-owner", protocolVersionsOwner.Hex(),
			"--guardian", guardian.Hex(),
			"--use-forge",
		}, nil)

		t.Logf("Bootstrap superchain (forge) output:\n%s", output)

		// Verify output file was created
		require.FileExists(t, superchainOutputFile)

		// Parse and validate the output
		var superchainOutput opcm.DeploySuperchainOutput
		data, err := os.ReadFile(superchainOutputFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &superchainOutput)
		require.NoError(t, err)
		require.NoError(t, addresses.CheckNoZeroAddresses(superchainOutput))
	})

	t.Run("bootstrap implementations with forge", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)
		workDir := runner.GetWorkDir()

		// First, we need a superchain deployment using Forge
		superchainOutputFile := filepath.Join(workDir, "bootstrap_superchain_for_impls_forge.json")
		runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "superchain",
			"--outfile", superchainOutputFile,
			"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--protocol-versions-owner", protocolVersionsOwner.Hex(),
			"--guardian", guardian.Hex(),
			"--use-forge",
		}, nil)

		// Parse superchain output to get addresses
		var superchainOutput opcm.DeploySuperchainOutput
		data, err := os.ReadFile(superchainOutputFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &superchainOutput)
		require.NoError(t, err)
		require.NoError(t, addresses.CheckNoZeroAddresses(superchainOutput))

		implsOutputFile := filepath.Join(workDir, "bootstrap_implementations_forge.json")

		// Run bootstrap implementations command with --use-forge
		output := runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "implementations",
			"--outfile", implsOutputFile,
			"--mips-version", strconv.Itoa(int(standard.MIPSVersion)),
			"--protocol-versions-proxy", superchainOutput.ProtocolVersionsProxy.Hex(),
			"--superchain-config-proxy", superchainOutput.SuperchainConfigProxy.Hex(),
			"--l1-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--superchain-proxy-admin", superchainOutput.SuperchainProxyAdmin.Hex(),
			"--challenger", challenger.Hex(),
			"--use-forge",
		}, nil)

		t.Logf("Bootstrap implementations (forge) output:\n%s", output)

		// Verify output file was created
		require.FileExists(t, implsOutputFile)

		// Parse and validate the output
		var implsOutput opcm.DeployImplementationsOutput
		data, err = os.ReadFile(implsOutputFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &implsOutput)
		require.NoError(t, err)

		// We only check specific addresses that are always set
		require.NotEqual(t, common.Address{}, implsOutput.Opcm, "Opcm should be set")
		require.NotEqual(t, common.Address{}, implsOutput.OpcmStandardValidator, "OpcmStandardValidator should be set")
		require.NotEqual(t, common.Address{}, implsOutput.DelayedWETHImpl, "DelayedWETHImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.OptimismPortalImpl, "OptimismPortalImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.ETHLockboxImpl, "ETHLockboxImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.PreimageOracleSingleton, "PreimageOracleSingleton should be set")
		require.NotEqual(t, common.Address{}, implsOutput.MipsSingleton, "MipsSingleton should be set")
		require.NotEqual(t, common.Address{}, implsOutput.SystemConfigImpl, "SystemConfigImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.L1CrossDomainMessengerImpl, "L1CrossDomainMessengerImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.L1ERC721BridgeImpl, "L1ERC721BridgeImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.L1StandardBridgeImpl, "L1StandardBridgeImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.OptimismMintableERC20FactoryImpl, "OptimismMintableERC20FactoryImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.DisputeGameFactoryImpl, "DisputeGameFactoryImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.AnchorStateRegistryImpl, "AnchorStateRegistryImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.SuperchainConfigImpl, "SuperchainConfigImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.ProtocolVersionsImpl, "ProtocolVersionsImpl should be set")
	})

	t.Run("bootstrap end-to-end with forge", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)
		workDir := runner.GetWorkDir()

		// Step 1: Bootstrap superchain with Forge
		superchainOutputFile := filepath.Join(workDir, "e2e_superchain_forge.json")
		runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "superchain",
			"--outfile", superchainOutputFile,
			"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--protocol-versions-owner", protocolVersionsOwner.Hex(),
			"--guardian", guardian.Hex(),
			"--use-forge",
		}, nil)

		var superchainOutput opcm.DeploySuperchainOutput
		data, err := os.ReadFile(superchainOutputFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &superchainOutput)
		require.NoError(t, err)

		// Step 2: Bootstrap implementations with Forge
		implsOutputFile := filepath.Join(workDir, "e2e_implementations_forge.json")
		runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "implementations",
			"--outfile", implsOutputFile,
			"--mips-version", strconv.Itoa(int(standard.MIPSVersion)),
			"--protocol-versions-proxy", superchainOutput.ProtocolVersionsProxy.Hex(),
			"--superchain-config-proxy", superchainOutput.SuperchainConfigProxy.Hex(),
			"--l1-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--superchain-proxy-admin", superchainOutput.SuperchainProxyAdmin.Hex(),
			"--challenger", challenger.Hex(),
			"--use-forge",
		}, nil)

		var implsOutput opcm.DeployImplementationsOutput
		data, err = os.ReadFile(implsOutputFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &implsOutput)
		require.NoError(t, err)

		// Verify all outputs have valid addresses
		require.NoError(t, addresses.CheckNoZeroAddresses(superchainOutput))
		require.NotEqual(t, common.Address{}, implsOutput.Opcm, "Opcm should be set")

		t.Log("✓ End-to-end bootstrap with Forge completed successfully")
	})
}
