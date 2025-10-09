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
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// TestCLIBootstrap tests the bootstrap commands via CLI
func TestCLIBootstrap(t *testing.T) {
	runner := NewCLITestRunnerWithNetwork(t)
	workDir := runner.GetWorkDir()

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

	t.Run("bootstrap superchain", func(t *testing.T) {
		superchainOutputFile := filepath.Join(workDir, "bootstrap_superchain.json")

		// Run bootstrap superchain command
		output := runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "superchain",
			"--outfile", superchainOutputFile,
			"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--protocol-versions-owner", protocolVersionsOwner.Hex(),
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

	t.Run("bootstrap superchain with custom protocol versions", func(t *testing.T) {
		superchainOutputFile := filepath.Join(workDir, "bootstrap_superchain_custom.json")

		// Use a custom protocol version - create v1.0.0 with custom build
		customVersion := params.ProtocolVersionV0{
			Build:      [8]byte{0, 0, 0, 0, 0, 0, 0, 1},
			Major:      1,
			Minor:      0,
			Patch:      0,
			PreRelease: 0,
		}.Encode()

		// Run bootstrap superchain command with custom versions
		output := runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "superchain",
			"--outfile", superchainOutputFile,
			"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--protocol-versions-owner", protocolVersionsOwner.Hex(),
			"--guardian", guardian.Hex(),
			"--required-protocol-version", common.Hash(customVersion).Hex(),
			"--recommended-protocol-version", common.Hash(customVersion).Hex(),
		}, nil)

		t.Logf("Bootstrap superchain (custom protocol versions) output:\n%s", output)

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
		superchainOutputFile := filepath.Join(workDir, "bootstrap_superchain_paused.json")

		// Run bootstrap superchain command with paused flag
		output := runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "superchain",
			"--outfile", superchainOutputFile,
			"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--protocol-versions-owner", protocolVersionsOwner.Hex(),
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
		// First, we need a superchain deployment
		superchainOutputFile := filepath.Join(workDir, "bootstrap_superchain_for_impls.json")
		runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "superchain",
			"--outfile", superchainOutputFile,
			"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--protocol-versions-owner", protocolVersionsOwner.Hex(),
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
			"--protocol-versions-proxy", superchainOutput.ProtocolVersionsProxy.Hex(),
			"--superchain-config-proxy", superchainOutput.SuperchainConfigProxy.Hex(),
			"--upgrade-controller", superchainProxyAdminOwner.Hex(), // Use proxy admin owner as upgrade controller
			"--superchain-proxy-admin", superchainOutput.SuperchainProxyAdmin.Hex(),
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

	t.Run("bootstrap proxy", func(t *testing.T) {
		proxyOutputFile := filepath.Join(workDir, "bootstrap_proxy.json")

		// Run bootstrap proxy command
		output := runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "proxy",
			"--proxy-owner", superchainProxyAdminOwner.Hex(),
			"--outfile", proxyOutputFile,
		}, nil)

		t.Logf("Bootstrap proxy output:\n%s", output)

		// Verify output file was created
		require.FileExists(t, proxyOutputFile)

		// Parse and validate the output
		var proxyOutput opcm.DeployProxyOutput
		data, err := os.ReadFile(proxyOutputFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &proxyOutput)
		require.NoError(t, err)
		require.NoError(t, addresses.CheckNoZeroAddresses(proxyOutput))
	})

	t.Run("bootstrap with stdout output", func(t *testing.T) {
		// Test that stdout output works (no --outfile flag)
		output := runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "proxy",
			"--proxy-owner", superchainProxyAdminOwner.Hex(),
			"--outfile", "-", // stdout
		}, nil)

		t.Logf("Bootstrap proxy (stdout) output:\n%s", output)
	})
}
