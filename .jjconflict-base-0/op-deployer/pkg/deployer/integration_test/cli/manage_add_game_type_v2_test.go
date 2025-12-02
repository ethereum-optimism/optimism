package cli

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/bootstrap"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestManageAddGameTypeV2_CLI(t *testing.T) {
	t.Run("missing required flag --config", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)
		runner.ExpectErrorContains(t, []string{
			"manage", "add-game-type-v2",
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
			"manage", "add-game-type-v2",
			"--config", configFile,
		}, nil, "missing required flag: l1-rpc-url")
	})

	t.Run("invalid config file path", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)
		runner.ExpectErrorContains(t, []string{
			"manage", "add-game-type-v2",
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
			"manage", "add-game-type-v2",
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
			"manage", "add-game-type-v2",
			"--config", configFile,
			"--l1-rpc-url", runner.l1RPC,
		}, nil, "failed to upgrade")
	})
}

// Tests the manage add-game-type-v2 command, from the CLI to the actual contract execution through the Solidity scripts.
func TestManageAddGameTypeV2_Integration(t *testing.T) {
	// TODO(#18718): Update this to use an actual deployed OPCM V2 contract once we have one.
	// For now, we manually deploy the OPCM V2 contract using bootstrap.Implementations.
	lgr := testlog.Logger(t, slog.LevelDebug)

	l1Rpc, stopL1, err := devnet.NewForkedSepolia(lgr)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, stopL1())
	})
	runner := NewCLITestRunnerWithNetwork(t, WithL1RPC(l1Rpc.RPCUrl()))
	workDir := runner.GetWorkDir()

	// Test values - using arbitrary addresses for testing
	l1ProxyAdminOwner := deployer.DefaultL1ProxyAdminOwnerSepolia
	systemConfigProxy := deployer.DefaultSystemConfigProxySepolia

	// Deploy the OPCM V2 contract.
	opcmV2 := deployDependencies(t, runner)

	bytes32Type := deployer.Bytes32Type
	addressType := deployer.AddressType

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
					// TODO(#18502): Remove this extra instruction after U18 ships.
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
		"manage", "add-game-type-v2",
		"--config", configFile,
		"--l1-rpc-url", runner.l1RPC,
		"--outfile", outputFile,
	}, nil)

	t.Logf("Command output (logs):\n%s", output)

	// Verify output file was created
	require.FileExists(t, outputFile)
	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	// Verify the file is not empty
	require.NotEmpty(t, data, "output file should not be empty")

	// Verify the file contains valid JSON
	require.True(t, json.Valid(data), "output file should contain valid JSON")

	// Verify the JSON can be unmarshaled into the expected structure
	var dump []broadcaster.CalldataDump
	require.NoError(t, json.Unmarshal(data, &dump))

	t.Logf("Add game type v2 generated calldata: %s", string(data))

	// Verify the calldata structure
	require.Len(t, dump, 1)
	require.Equal(t, l1ProxyAdminOwner.Hex(), dump[0].To.Hex(), "calldata should be sent to prank address")

	// Verify the calldata has the correct function selector for opcm.upgrade
	// The selector for `upgrade((address,(bool,uint256,uint32,bytes)[],(string,bytes)[]))` is 0x8a847e2e
	calldata := dump[0].Data
	require.GreaterOrEqual(t, len(calldata), 4, "calldata should be at least 4 bytes for function selector")

	expectedSelector := common.FromHex("8a847e2e")
	actualSelector := calldata[:4]
	require.Equal(t, hex.EncodeToString(expectedSelector), hex.EncodeToString(actualSelector),
		"calldata should contain opcmV2.upgrade function selector 0x8a847e2e, got: %s", hex.EncodeToString(actualSelector))

	// Verify the calldata contains the correct upgrade input
	// We construct the expected calldata from testConfig
	expectedEncodedParams, err := testConfig.EncodedUpgradeInputV2()
	require.NoError(t, err, "failed to encode expected upgrade input")

	// Construct expected calldata: function selector + encoded parameters
	expectedCalldata := append(expectedSelector, expectedEncodedParams...)

	// Compare the full calldata (excluding the selector which we already verified)
	require.Equal(t, len(expectedCalldata), len(calldata),
		"calldata length mismatch: expected %d bytes, got %d bytes", len(expectedCalldata), len(calldata))

	// Compare the encoded parameters (skip the 4-byte selector)
	require.Equal(t, hex.EncodeToString(expectedEncodedParams), hex.EncodeToString(calldata[4:]),
		"encoded upgrade input parameters do not match expected values")

	// Verify To is the prank address
	require.Equal(t, l1ProxyAdminOwner.Hex(), dump[0].To.Hex(), "calldata should be sent to prank address")
}

// TODO(#18718): Remove this once we have a deployed OPCM V2 contract.
// deployDependencies deploys the superchain contracts and OPCM V2 implementation
// using the DeployImplementations script, and returns the OPCM V2 address
func deployDependencies(t *testing.T, runner *CLITestRunner) common.Address {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	// First, deploy superchain contracts (required for OPCM deployment)
	superchainProxyAdminOwner := common.Address{'S'}
	superchainOut, err := bootstrap.Superchain(ctx, bootstrap.SuperchainConfig{
		L1RPCUrl:                   runner.l1RPC,
		PrivateKey:                 runner.privateKeyHex,
		ArtifactsLocator:           artifacts.EmbeddedLocator,
		Logger:                     runner.lgr,
		SuperchainProxyAdminOwner:  superchainProxyAdminOwner,
		ProtocolVersionsOwner:      common.Address{'P'},
		Guardian:                   common.Address{'G'},
		Paused:                     false,
		RequiredProtocolVersion:    params.ProtocolVersionV0{Major: 1}.Encode(),
		RecommendedProtocolVersion: params.ProtocolVersionV0{Major: 2}.Encode(),
		CacheDir:                   testCacheDir,
	})
	require.NoError(t, err, "Failed to deploy superchain contracts")

	// Deploy implementations with OPCM V2 enabled
	implOut, err := bootstrap.Implementations(ctx, bootstrap.ImplementationsConfig{
		L1RPCUrl:                        runner.l1RPC,
		PrivateKey:                      runner.privateKeyHex,
		ArtifactsLocator:                artifacts.EmbeddedLocator,
		Logger:                          runner.lgr,
		WithdrawalDelaySeconds:          standard.WithdrawalDelaySeconds,
		MinProposalSizeBytes:            standard.MinProposalSizeBytes,
		ChallengePeriodSeconds:          standard.ChallengePeriodSeconds,
		ProofMaturityDelaySeconds:       standard.ProofMaturityDelaySeconds,
		DisputeGameFinalityDelaySeconds: standard.DisputeGameFinalityDelaySeconds,
		MIPSVersion:                     int(standard.MIPSVersion),
		DevFeatureBitmap:                deployer.OPCMV2DevFlag, // Enable OPCM V2
		SuperchainConfigProxy:           superchainOut.SuperchainConfigProxy,
		ProtocolVersionsProxy:           superchainOut.ProtocolVersionsProxy,
		SuperchainProxyAdmin:            superchainOut.SuperchainProxyAdmin,
		L1ProxyAdminOwner:               superchainProxyAdminOwner,
		Challenger:                      common.Address{'C'},
		CacheDir:                        testCacheDir,
		FaultGameMaxGameDepth:           standard.DisputeMaxGameDepth,
		FaultGameSplitDepth:             standard.DisputeSplitDepth,
		FaultGameClockExtension:         standard.DisputeClockExtension,
		FaultGameMaxClockDuration:       standard.DisputeMaxClockDuration,
	})
	require.NoError(t, err, "Failed to deploy implementations")

	// Verify OPCM V2 was deployed
	require.NotEqual(t, common.Address{}, implOut.OpcmV2, "OPCM V2 address should be set")
	require.Equal(t, common.Address{}, implOut.Opcm, "OPCM V1 address should be zero when V2 is deployed")

	t.Logf("Deployed OPCM V2 at address: %s", implOut.OpcmV2.Hex())
	t.Logf("SuperchainConfigProxy: %s", superchainOut.SuperchainConfigProxy.Hex())

	return implOut.OpcmV2
}
