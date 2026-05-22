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

// TestCLIBootstrap tests the bootstrap commands via CLI
func TestCLIBootstrap(t *testing.T) {
	// Use the same chain ID that anvil runs on
	l1ChainID := uint64(devnet.DefaultChainID)
	l1ChainIDBig := big.NewInt(int64(l1ChainID))

	// Get dev keys for role addresses
	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)

	// Get addresses for required roles
	superchainProxyAdminOwner := shared.AddrFor(t, dk, devkeys.L1ProxyAdminOwnerRole.Key(l1ChainIDBig))
	guardian := shared.AddrFor(t, dk, devkeys.SuperchainConfigGuardianKey.Key(l1ChainIDBig))
	challenger := shared.AddrFor(t, dk, devkeys.ChallengerRole.Key(l1ChainIDBig))

	t.Run("bootstrap superchain", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)
		workDir := runner.GetWorkDir()

		superchainOutputFile := filepath.Join(workDir, "bootstrap_superchain.json")

		// Run bootstrap superchain command
		output := runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "superchain",
			"--outfile", superchainOutputFile,
			"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--guardian", guardian.Hex(),
		}, nil)

		t.Logf("Bootstrap superchain output:\n%s", output)

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

	t.Run("bootstrap superchain paused", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)
		workDir := runner.GetWorkDir()

		superchainOutputFile := filepath.Join(workDir, "bootstrap_superchain_paused.json")

		// Run bootstrap superchain command with paused flag
		output := runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "superchain",
			"--outfile", superchainOutputFile,
			"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--guardian", guardian.Hex(),
			"--paused",
		}, nil)

		t.Logf("Bootstrap superchain (paused) output:\n%s", output)

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

	t.Run("bootstrap implementations", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)
		workDir := runner.GetWorkDir()

		// First, we need a superchain deployment
		superchainOutputFile := filepath.Join(workDir, "bootstrap_superchain_for_impls.json")
		runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "superchain",
			"--outfile", superchainOutputFile,
			"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--guardian", guardian.Hex(),
		}, nil)

		// Parse superchain output to get addresses
		var superchainOutput opcm.DeploySuperchainOutput
		data, err := os.ReadFile(superchainOutputFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &superchainOutput)
		require.NoError(t, err)
		require.NoError(t, addresses.CheckNoZeroAddresses(superchainOutput))

		implsOutputFile := filepath.Join(workDir, "bootstrap_implementations.json")

		// Run bootstrap implementations command (use same key index as superchain deployment)
		output := runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "implementations",
			"--outfile", implsOutputFile,
			"--mips-version", strconv.Itoa(int(standard.MIPSVersion)),
			"--superchain-config-proxy", superchainOutput.Address("superchainConfigProxy").Hex(),
			"--l1-proxy-admin-owner", superchainProxyAdminOwner.Hex(), // Use proxy admin owner as upgrade controller
			"--superchain-proxy-admin", superchainOutput.Address("superchainProxyAdmin").Hex(),
			"--challenger", challenger.Hex(),
		}, nil)

		t.Logf("Bootstrap implementations output:\n%s", output)

		// Verify output file was created
		require.FileExists(t, implsOutputFile)

		// Parse and validate the output
		var implsOutput opcm.DeployImplementationsOutput
		data, err = os.ReadFile(implsOutputFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &implsOutput)
		require.NoError(t, err)

		// We only check specific addresses that are always set
		require.NotEqual(t, common.Address{}, implsOutput.Address("opcmV2"), "OpcmV2 should be set")
		require.NotEqual(t, common.Address{}, implsOutput.Address("opcmStandardValidator"), "OpcmStandardValidator should be set")
		require.NotEqual(t, common.Address{}, implsOutput.Address("delayedWETHImpl"), "DelayedWETHImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.Address("optimismPortalImpl"), "OptimismPortalImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.Address("ethLockboxImpl"), "ETHLockboxImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.Address("preimageOracleSingleton"), "PreimageOracleSingleton should be set")
		require.NotEqual(t, common.Address{}, implsOutput.Address("mipsSingleton"), "MipsSingleton should be set")
		require.NotEqual(t, common.Address{}, implsOutput.Address("systemConfigImpl"), "SystemConfigImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.Address("l1CrossDomainMessengerImpl"), "L1CrossDomainMessengerImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.Address("l1ERC721BridgeImpl"), "L1ERC721BridgeImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.Address("l1StandardBridgeImpl"), "L1StandardBridgeImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.Address("optimismMintableERC20FactoryImpl"), "OptimismMintableERC20FactoryImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.Address("disputeGameFactoryImpl"), "DisputeGameFactoryImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.Address("anchorStateRegistryImpl"), "AnchorStateRegistryImpl should be set")
		require.NotEqual(t, common.Address{}, implsOutput.Address("superchainConfigImpl"), "SuperchainConfigImpl should be set")
	})
}
