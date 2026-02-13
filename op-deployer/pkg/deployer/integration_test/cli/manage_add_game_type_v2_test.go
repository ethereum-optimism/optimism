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
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
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
	lgr := testlog.Logger(t, slog.LevelDebug)

	l1Rpc, stopL1, err := devnet.NewForkedSepolia(lgr)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, stopL1())
	})
	runner := NewCLITestRunnerWithNetwork(t, WithL1RPC(l1Rpc.RPCUrl()))
	workDir := runner.GetWorkDir()

	// We deploy superchain, OPCM V2, and a fresh OP chain.
	deployed := deployDependencies(t, runner)

	l1ProxyAdminOwner := deployed.proxyAdminOwner
	systemConfigProxy := deployed.systemConfigProxy
	opcmV2 := deployed.opcmV2

	// FaultDisputeGameConfig just needs absolutePrestate (bytes32)
	testPrestate := common.Hash{'P', 'R', 'E', 'S', 'T', 'A', 'T', 'E'}

	// PermissionedDisputeGameConfig needs absolutePrestate, proposer, challenger
	testProposer := common.Address{'P'}
	testChallenger := common.Address{'C'}

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
					FaultDisputeGameConfig: &embedded.FaultDisputeGameConfig{
						AbsolutePrestate: testPrestate,
					},
				},
				{
					Enabled:  true,
					InitBond: big.NewInt(1000000000000000000),
					GameType: embedded.GameTypePermissionedCannon,
					PermissionedDisputeGameConfig: &embedded.PermissionedDisputeGameConfig{
						AbsolutePrestate: testPrestate,
						Proposer:         testProposer,
						Challenger:       testChallenger,
					},
				},
				{
					Enabled:  false,
					InitBond: big.NewInt(0),
					GameType: embedded.GameTypeCannonKona,
				},
				{
					Enabled:  false,
					InitBond: big.NewInt(0),
					GameType: embedded.GameTypeSuperCannon,
				},
				{
					Enabled:  false,
					InitBond: big.NewInt(0),
					GameType: embedded.GameTypeSuperPermCannon,
				},
				{
					Enabled:  false,
					InitBond: big.NewInt(0),
					GameType: embedded.GameTypeSuperCannonKona,
				},
			},
			ExtraInstructions: []embedded.ExtraInstruction{
				{
					Key:  "PermittedProxyDeployment",
					Data: []byte("DelayedWETH"),
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

// deployedChain holds the addresses returned from deploying a fresh OP chain
type deployedChain struct {
	opcmV2            common.Address
	systemConfigProxy common.Address
	proxyAdminOwner   common.Address
}

// deployDependencies deploys superchain, OPCM V2, and a fresh OP chain using ApplyPipeline.
// Returns addresses needed for testing the add-game-type-v2 command.
func deployDependencies(t *testing.T, runner *CLITestRunner) deployedChain {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	// Get the private key and devkeys
	pk, err := crypto.HexToECDSA(runner.privateKeyHex)
	require.NoError(t, err)

	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)

	l1ChainID := big.NewInt(11155111)

	// We use the shared helper to create an intent and state
	loc, _ := testutil.LocalArtifacts(t)
	l2ChainID := uint256.NewInt(12345) // Test L2 chain ID

	intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, 30_000_000)

	// Ensure we are using OPCM V2
	intent.GlobalDeployOverrides = map[string]any{
		"devFeatureBitmap": deployer.OPCMV2DevFlag,
	}

	// Deploy using ApplyPipeline with live target
	err = deployer.ApplyPipeline(ctx, deployer.ApplyPipelineOpts{
		DeploymentTarget:   deployer.DeploymentTargetLive,
		L1RPCUrl:           runner.l1RPC,
		DeployerPrivateKey: pk,
		Intent:             intent,
		State:              st,
		Logger:             runner.lgr,
		StateWriter:        pipeline.NoopStateWriter(),
		CacheDir:           testCacheDir,
	})
	require.NoError(t, err, "Failed to deploy OP chain")

	// Verify OPCM V2 was deployed
	require.NotEqual(t, common.Address{}, st.ImplementationsDeployment.OpcmV2Impl, "OPCM V2 address should be set")

	// Get the chain state
	require.Len(t, st.Chains, 1, "Expected one chain to be deployed")
	chainState := st.Chains[0]

	t.Logf("Deployed OPCM V2 at address: %s", st.ImplementationsDeployment.OpcmV2Impl.Hex())
	t.Logf("Deployed SystemConfigProxy at address: %s", chainState.SystemConfigProxy.Hex())
	t.Logf("ProxyAdminOwner: %s", intent.Chains[0].Roles.L1ProxyAdminOwner.Hex())

	return deployedChain{
		opcmV2:            st.ImplementationsDeployment.OpcmV2Impl,
		systemConfigProxy: chainState.SystemConfigProxy,
		proxyAdminOwner:   intent.Chains[0].Roles.L1ProxyAdminOwner,
	}
}
