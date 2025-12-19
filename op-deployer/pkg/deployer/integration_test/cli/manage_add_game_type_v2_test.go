package cli

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestManageAddGameTypeV2_CLI(t *testing.T) {
	t.Run("missing required flag --config", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)
		runner.ExpectErrorContains(t, []string{
			"manage", "add-game-type-opcm-v2",
			"--l1-rpc-url", runner.l1RPC,
		}, nil, "missing required flag: config")
	})

	t.Run("missing required flag --l1-rpc-url", func(t *testing.T) {
		runner := NewCLITestRunner(t)
		workDir := runner.GetWorkDir()
		configFile := filepath.Join(workDir, "config.json")

		// Create a minimal valid config file
		config := embedded.UpgradeOPChainInput{
			Prank: common.Address{0x01},
			Opcm:  common.Address{0x02},
			UpgradeInputV2: &embedded.UpgradeInputV2{
				SystemConfig:       common.Address{0x03},
				DisputeGameConfigs: []embedded.DisputeGameConfig{},
				ExtraInstructions:  []embedded.ExtraInstruction{},
			},
		}
		configData, err := json.Marshal(config)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(configFile, configData, 0o644))

		runner.ExpectErrorContains(t, []string{
			"manage", "add-game-type-opcm-v2",
			"--config", configFile,
		}, nil, "missing required flag: l1-rpc-url")
	})

	t.Run("invalid config file path", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)
		runner.ExpectErrorContains(t, []string{
			"manage", "add-game-type-opcm-v2",
			"--config", "/nonexistent/path/config.json",
			"--l1-rpc-url", runner.l1RPC,
		}, nil, "failed to read config file")
	})

	t.Run("invalid JSON config file", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)
		workDir := runner.GetWorkDir()
		configFile := filepath.Join(workDir, "invalid_config.json")

		// Write invalid JSON
		require.NoError(t, os.WriteFile(configFile, []byte("{invalid json}"), 0o644))

		runner.ExpectErrorContains(t, []string{
			"manage", "add-game-type-opcm-v2",
			"--config", configFile,
			"--l1-rpc-url", runner.l1RPC,
		}, nil, "failed to upgrade")
	})

	t.Run("config file missing required fields", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)
		workDir := runner.GetWorkDir()
		configFile := filepath.Join(workDir, "incomplete_config.json")

		// Create config missing prank or opcm
		config := map[string]interface{}{
			"prank": common.Address{0x01}.Hex(),
			// Missing opcm
		}
		configData, err := json.Marshal(config)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(configFile, configData, 0o644))

		runner.ExpectErrorContains(t, []string{
			"manage", "add-game-type-opcm-v2",
			"--config", configFile,
			"--l1-rpc-url", runner.l1RPC,
		}, nil, "failed to upgrade")
	})
}

func TestManageAddGameTypeV2_Integration(t *testing.T) {
	// TODO(#????): Update this to use an actual deployed OPCM V2 contract
	t.Skip("Skipping until we have a deployed OPCM V2 contract")
	return

	runner := NewCLITestRunnerWithNetwork(t)
	workDir := runner.GetWorkDir()

	// Test values - using arbitrary addresses for testing
	l1ProxyAdminOwner := common.HexToAddress("0x1Eb2fFc903729a0F03966B917003800b145F56E2")
	systemConfigProxy := common.HexToAddress("0x034edD2A225f7f429A63E0f1D2084B9E0A93b538")

	// Get OPCM V2 address from standard (using Sepolia chain ID for address lookup)
	opcmV2, err := standard.OPCMImplAddressFor(11155111, standard.ContractsV500Tag)
	require.NoError(t, err)

	bytes32Type, err := abi.NewType("bytes32", "", nil)
	require.NoError(t, err)
	addressType, err := abi.NewType("address", "", nil)
	require.NoError(t, err)

	// FaultDisputeGameConfig just needs absolutePrestate (bytes32)
	testPrestate := common.Hash{'P', 'R', 'E', 'S', 'T', 'A', 'T', 'E'}
	cannonArgs, err := abi.Arguments{{Type: bytes32Type}}.Pack(testPrestate)
	require.NoError(t, err)

	// PermissionedDisputeGameConfig needs absolutePrestate, proposer, challenger
	testProposer := common.Address{'P'}
	testChallenger := common.Address{'C'}
	permissionedArgs, err := abi.Arguments{
		{Type: bytes32Type},
		{Type: addressType},
		{Type: addressType},
	}.Pack(testPrestate, testProposer, testChallenger)
	require.NoError(t, err)

	testConfig := embedded.UpgradeOPChainInput{
		Prank: l1ProxyAdminOwner,
		Opcm:  opcmV2,
		UpgradeInputV2: &embedded.UpgradeInputV2{
			SystemConfig: systemConfigProxy,
			DisputeGameConfigs: []embedded.DisputeGameConfig{
				{
					Enabled:  true,
					InitBond: big.NewInt(1000000000000000000),
					GameType: embedded.GameTypeCannon,
					GameArgs: cannonArgs,
				},
				{
					Enabled:  true,
					InitBond: big.NewInt(1000000000000000000),
					GameType: embedded.GameTypePermissionedCannon,
					GameArgs: permissionedArgs,
				},
				{
					Enabled:  false,
					InitBond: big.NewInt(0),
					GameType: embedded.GameTypeCannonKona,
					GameArgs: []byte{}, // Disabled games don't need args
				},
			},
			ExtraInstructions: []embedded.ExtraInstruction{
				{
					Key:  "PermittedProxyDeployment",
					Data: []byte("DelayedWETH"),
				},
				{
					Key:  "overrides.cfg.useCustomGasToken",
					Data: make([]byte, 32),
				},
			},
		},
	}

	configFile := filepath.Join(workDir, "add_game_type_v2_config.json")
	outputFile := filepath.Join(workDir, "add_game_type_v2_output.json")

	configData, err := json.MarshalIndent(testConfig, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configFile, configData, 0o644))

	// Run the CLI command
	output := runner.ExpectSuccess(t, []string{
		"manage", "add-game-type-opcm-v2",
		"--config", configFile,
		"--l1-rpc-url", runner.l1RPC,
		"--outfile", outputFile,
	}, nil)

	t.Logf("Command output (logs):\n%s", output)

	// Verify output file was created
	require.FileExists(t, outputFile)
	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	var dump []broadcaster.CalldataDump
	require.NoError(t, json.Unmarshal(data, &dump))

	t.Logf("Add game type v2 generated calldata: %s", string(data))

	// Verify the calldata structure
	require.Len(t, dump, 1)
	require.Equal(t, l1ProxyAdminOwner.Hex(), dump[0].To.Hex(), "calldata should be sent to prank address")

	// Verify the calldata has the correct function selector for opcm.upgrade
	// The selector for upgrade(address,bytes) is 0xff2dd5a1
	dataHex := hex.EncodeToString(dump[0].Data)
	prefix := dataHex
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	require.True(t, strings.HasPrefix(dataHex, "ff2dd5a1"),
		"calldata should have opcm.upgrade function selector ff2dd5a1, got: %s", prefix)
}
