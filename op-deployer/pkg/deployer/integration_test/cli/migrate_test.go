package cli

import (
	"context"
	"encoding/json"
	"math/big"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"log/slog"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/bootstrap"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/manage"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestCLIMigrateRequiredFlags tests that required flags are validated for both OPCM v1 and v2
func TestCLIMigrateRequiredFlags(t *testing.T) {
	// Test common required flags (apply to both v1 and v2)

	t.Run("missing l1-rpc-url", func(t *testing.T) {
		runner := NewCLITestRunner(t)
		runner.ExpectErrorContains(t, []string{
			"manage", "migrate",
			"--private-key", "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			"--opcm-impl-address", common.Address{0x02}.Hex(),
			// Intentionally omit --l1-rpc-url
		}, nil, "missing required flag: l1-rpc-url")
	})

	t.Run("missing private-key", func(t *testing.T) {
		runner := NewCLITestRunner(t)
		runner.ExpectErrorContains(t, []string{
			"manage", "migrate",
			"--l1-rpc-url", "http://localhost:8545",
			"--opcm-impl-address", common.Address{0x02}.Hex(),
			// Intentionally omit --private-key
		}, nil, "missing required flag: private-key")
	})

	t.Run("missing opcm-impl-address", func(t *testing.T) {
		runner := NewCLITestRunner(t)
		runner.ExpectErrorContains(t, []string{
			"manage", "migrate",
			"--l1-rpc-url", "http://localhost:8545",
			"--private-key", "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			// Intentionally omit --opcm-impl-address
		}, nil, "missing required flag: opcm-impl-address")
	})

	t.Run("missing system-config-proxy-address", func(t *testing.T) {
		runner := NewCLITestRunner(t)
		runner.ExpectErrorContains(t, []string{
			"manage", "migrate",
			"--l1-rpc-url", "http://localhost:8545",
			"--private-key", "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			"--opcm-impl-address", common.Address{0x02}.Hex(),
			// Intentionally omit --system-config-proxy-address
		}, nil, "missing required flag: system-config-proxy-address")
	})

	t.Run("missing starting-anchor-root", func(t *testing.T) {
		runner := NewCLITestRunner(t)
		runner.ExpectErrorContains(t, []string{
			"manage", "migrate",
			"--l1-rpc-url", "http://localhost:8545",
			"--private-key", "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			"--opcm-impl-address", common.Address{0x02}.Hex(),
			"--system-config-proxy-address", common.Address{0x03}.Hex(),
			// Intentionally omit --starting-anchor-root
		}, nil, "missing required flag: starting-anchor-root")
	})
}

// TestCLIMigrateV1 tests the migrate-v1 CLI command for OPCM v1
func TestCLIMigrateV1(t *testing.T) {
	lgr := testlog.Logger(t, slog.LevelDebug)

	forkedL1, stopL1, err := devnet.NewForkedSepolia(lgr)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, stopL1())
	})
	l1RPC := forkedL1.RPCUrl()

	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	pkHex, _, _ := shared.DefaultPrivkey(t)

	privateKeyECDSA, err := crypto.HexToECDSA(strings.TrimPrefix(pkHex, "0x"))
	require.NoError(t, err)
	prank := crypto.PubkeyToAddress(privateKeyECDSA.PublicKey)

	// Deploy superchain contracts first (required for OPCM deployment)
	superchainProxyAdminOwner := prank
	superchainOut, err := bootstrap.Superchain(ctx, bootstrap.SuperchainConfig{
		L1RPCUrl:                   l1RPC,
		PrivateKey:                 pkHex,
		ArtifactsLocator:           artifacts.EmbeddedLocator,
		Logger:                     lgr,
		SuperchainProxyAdminOwner:  superchainProxyAdminOwner,
		ProtocolVersionsOwner:      common.Address{'P'},
		Guardian:                   common.Address{'G'},
		Paused:                     false,
		RequiredProtocolVersion:    params.ProtocolVersionV0{Major: 1}.Encode(),
		RecommendedProtocolVersion: params.ProtocolVersionV0{Major: 2}.Encode(),
		CacheDir:                   testCacheDir,
	})
	require.NoError(t, err, "Failed to deploy superchain contracts")

	// Deploy OPCM V1 implementations (no OPCMV2DevFlag)
	cfg := bootstrap.ImplementationsConfig{
		L1RPCUrl:                        l1RPC,
		PrivateKey:                      pkHex,
		ArtifactsLocator:                artifacts.EmbeddedLocator,
		Logger:                          lgr,
		MIPSVersion:                     int(standard.MIPSVersion),
		WithdrawalDelaySeconds:          standard.WithdrawalDelaySeconds,
		MinProposalSizeBytes:            standard.MinProposalSizeBytes,
		ChallengePeriodSeconds:          standard.ChallengePeriodSeconds,
		ProofMaturityDelaySeconds:       standard.ProofMaturityDelaySeconds,
		DisputeGameFinalityDelaySeconds: standard.DisputeGameFinalityDelaySeconds,
		DevFeatureBitmap:                deployer.EnableDevFeature(common.Hash{}, deployer.OptimismPortalInteropDevFlag),
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
	}

	impls, err := bootstrap.Implementations(ctx, cfg)
	require.NoError(t, err, "Failed to deploy implementations")
	require.NotEqual(t, common.Address{}, impls.Opcm, "OPCM V1 address should be set")
	require.Equal(t, common.Address{}, impls.OpcmV2, "OPCM V2 address should be zero when V1 is deployed")

	// Set up a test chain
	l1ChainID := uint64(11155111) // Sepolia chain ID
	l2ChainID := uint256.NewInt(1)

	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)

	// Create a runner with network setup
	runner := NewCLITestRunnerWithNetwork(t, WithL1RPC(l1RPC), WithPrivateKey(pkHex))
	workDir := runner.GetWorkDir()

	// Initialize intent and deploy chain
	intent, _ := cliInitIntent(t, runner, l1ChainID, []common.Hash{l2ChainID.Bytes32()})

	if intent.SuperchainRoles == nil {
		intent.SuperchainRoles = &addresses.SuperchainRoles{}
	}

	l1ChainIDBig := big.NewInt(int64(l1ChainID))
	intent.SuperchainRoles.SuperchainProxyAdminOwner = superchainProxyAdminOwner
	intent.SuperchainRoles.SuperchainGuardian = shared.AddrFor(t, dk, devkeys.SuperchainConfigGuardianKey.Key(l1ChainIDBig))
	intent.SuperchainRoles.ProtocolVersionsOwner = superchainProxyAdminOwner
	intent.SuperchainRoles.Challenger = shared.AddrFor(t, dk, devkeys.ChallengerRole.Key(l1ChainIDBig))

	for _, chain := range intent.Chains {
		chain.Roles.L1ProxyAdminOwner = superchainProxyAdminOwner
		chain.Roles.L2ProxyAdminOwner = shared.AddrFor(t, dk, devkeys.L2ProxyAdminOwnerRole.Key(l1ChainIDBig))
		chain.Roles.SystemConfigOwner = superchainProxyAdminOwner
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

	// Populate the state with predeployed superchain and implementations
	// so the pipeline knows about them
	st, err := pipeline.ReadState(workDir)
	require.NoError(t, err)

	// Set superchain deployment addresses
	if st.SuperchainDeployment == nil {
		st.SuperchainDeployment = &addresses.SuperchainContracts{
			SuperchainConfigProxy:    superchainOut.SuperchainConfigProxy,
			SuperchainConfigImpl:     superchainOut.SuperchainConfigImpl,
			ProtocolVersionsProxy:    superchainOut.ProtocolVersionsProxy,
			ProtocolVersionsImpl:     superchainOut.ProtocolVersionsImpl,
			SuperchainProxyAdminImpl: superchainOut.SuperchainProxyAdmin,
		}
	}

	// Set implementations deployment addresses
	if st.ImplementationsDeployment == nil {
		st.ImplementationsDeployment = &addresses.ImplementationsContracts{
			OpcmImpl:                         impls.Opcm,
			OptimismPortalImpl:               impls.OptimismPortalImpl,
			DelayedWethImpl:                  impls.DelayedWETHImpl,
			EthLockboxImpl:                   impls.ETHLockboxImpl,
			SystemConfigImpl:                 impls.SystemConfigImpl,
			L1CrossDomainMessengerImpl:       impls.L1CrossDomainMessengerImpl,
			L1Erc721BridgeImpl:               impls.L1ERC721BridgeImpl,
			L1StandardBridgeImpl:             impls.L1StandardBridgeImpl,
			OptimismMintableErc20FactoryImpl: impls.OptimismMintableERC20FactoryImpl,
			DisputeGameFactoryImpl:           impls.DisputeGameFactoryImpl,
			AnchorStateRegistryImpl:          impls.AnchorStateRegistryImpl,
			PreimageOracleImpl:               impls.PreimageOracleSingleton,
			MipsImpl:                         impls.MipsSingleton,
		}
	}
	require.NoError(t, pipeline.WriteState(workDir, st))

	require.NoError(t, intent.WriteToFile(filepath.Join(workDir, "intent.toml")))

	// Apply deployment
	// Note: Validation will run automatically but may find expected errors for migration test deployments
	// (e.g., custom dev features, non-standard configurations). We verify deployment succeeded despite validation errors.
	applyCtx, applyCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer applyCancel()
	output, runErr := runner.RunWithNetwork(applyCtx, []string{
		"apply",
		"--deployment-target", "live",
		"--workdir", workDir,
		"--validate", "auto",
	}, nil)

	// Verify deployment succeeded regardless of validation errors
	st, err = pipeline.ReadState(workDir)
	require.NoError(t, err, "State should be readable after apply")
	require.NotNil(t, st.AppliedIntent, "Applied intent should exist")
	require.Len(t, st.Chains, 1, "Should have one chain deployed")

	// If there was an error, it should be validation-related, not a deployment failure
	if runErr != nil {
		require.Contains(t, output, "validation", "Error should be validation-related, not deployment failure")
		require.Contains(t, runErr.Error(), "validation", "Error should mention validation")
	}

	systemConfigProxy := st.Chains[0].SystemConfigProxy

	// Run migrate command
	migrateOutput := runner.ExpectSuccessWithNetwork(t, []string{
		"manage",
		"migrate",
		"--l1-rpc-url", l1RPC,
		"--private-key", pkHex,
		"--l1-proxy-admin-owner-address", superchainProxyAdminOwner.Hex(),
		"--opcm-impl-address", impls.Opcm.Hex(),
		"--system-config-proxy-address", systemConfigProxy.Hex(),
		"--permissionless",
		"--proposer-address", common.Address{'P'}.Hex(),
		"--challenger-address", common.Address{'C'}.Hex(),
		"--starting-anchor-root", "0x0000000000000000000000000000000000000000000000000000000000000abc",
		"--starting-anchor-l2-sequence-number", "1",
		"--dispute-max-game-depth", "73",
		"--dispute-split-depth", "30",
		"--initial-bond", "1000000000000000000",
		"--dispute-clock-extension", "10800",
		"--dispute-max-clock-duration", "302400",
		"--dispute-absolute-prestate-cannon", "0x0000000000000000000000000000000000000000000000000000000000000def",
		"--dispute-absolute-prestate-cannon-kona", "0x0000000000000000000000000000000000000000000000000000000000000fed",
	}, nil)

	// Parse output to verify DisputeGameFactory was deployed
	// Find the JSON output by looking for the opening brace
	var migrationOutput manage.InteropMigrationOutput
	jsonStart := strings.Index(migrateOutput, "{")
	if jsonStart == -1 {
		t.Logf("Full output length: %d", len(migrateOutput))
		t.Logf("Full output: %q", migrateOutput)
		t.Fatalf("No JSON output found in output")
	}
	// Find the end of the JSON object
	jsonEnd := strings.Index(migrateOutput[jsonStart:], "}") + jsonStart + 1
	jsonOutput := migrateOutput[jsonStart:jsonEnd]
	err = json.Unmarshal([]byte(jsonOutput), &migrationOutput)
	require.NoError(t, err, "Failed to parse migration output: %s", jsonOutput)
	require.NotEqual(t, common.Address{}, migrationOutput.DisputeGameFactory, "DisputeGameFactory should be deployed")
}

// TestCLIMigrateV2 tests the migrate-v2 CLI command for OPCM v2
func TestCLIMigrateV2(t *testing.T) {
	lgr := testlog.Logger(t, slog.LevelDebug)

	forkedL1, stopL1, err := devnet.NewForkedSepolia(lgr)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, stopL1())
	})
	l1RPC := forkedL1.RPCUrl()

	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	pkHex, _, _ := shared.DefaultPrivkey(t)

	privateKeyECDSA, err := crypto.HexToECDSA(strings.TrimPrefix(pkHex, "0x"))
	require.NoError(t, err)
	prank := crypto.PubkeyToAddress(privateKeyECDSA.PublicKey)

	// Deploy superchain contracts first (required for OPCM deployment)
	superchainProxyAdminOwner := prank

	// Deploy superchain contracts first
	superchainOut, err := bootstrap.Superchain(ctx, bootstrap.SuperchainConfig{
		L1RPCUrl:                   l1RPC,
		PrivateKey:                 pkHex,
		ArtifactsLocator:           artifacts.EmbeddedLocator,
		Logger:                     lgr,
		SuperchainProxyAdminOwner:  superchainProxyAdminOwner,
		ProtocolVersionsOwner:      common.Address{'P'},
		Guardian:                   common.Address{'G'},
		Paused:                     false,
		RequiredProtocolVersion:    params.ProtocolVersionV0{Major: 1}.Encode(),
		RecommendedProtocolVersion: params.ProtocolVersionV0{Major: 2}.Encode(),
		CacheDir:                   testCacheDir,
	})
	require.NoError(t, err, "Failed to deploy superchain contracts")

	devFeatureBitmap := deployer.EnableDevFeature(deployer.OPCMV2DevFlag, deployer.OptimismPortalInteropDevFlag)

	// Deploy OPCM V2 implementations (with OPCMV2DevFlag)
	cfg := bootstrap.ImplementationsConfig{
		L1RPCUrl:                        l1RPC,
		PrivateKey:                      pkHex,
		ArtifactsLocator:                artifacts.EmbeddedLocator,
		Logger:                          lgr,
		MIPSVersion:                     int(standard.MIPSVersion),
		WithdrawalDelaySeconds:          standard.WithdrawalDelaySeconds,
		MinProposalSizeBytes:            standard.MinProposalSizeBytes,
		ChallengePeriodSeconds:          standard.ChallengePeriodSeconds,
		ProofMaturityDelaySeconds:       standard.ProofMaturityDelaySeconds,
		DisputeGameFinalityDelaySeconds: standard.DisputeGameFinalityDelaySeconds,
		DevFeatureBitmap:                devFeatureBitmap,
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
	}

	impls, err := bootstrap.Implementations(ctx, cfg)
	require.NoError(t, err, "Failed to deploy implementations")
	require.NotEqual(t, common.Address{}, impls.OpcmV2, "OPCM V2 address should be set")
	require.Equal(t, common.Address{}, impls.Opcm, "OPCM V1 address should be zero when V2 is deployed")

	// Set up a test chain
	l1ChainID := uint64(11155111) // Sepolia chain ID
	l2ChainID := uint256.NewInt(1)

	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)

	// Create a runner with network setup
	runner := NewCLITestRunnerWithNetwork(t, WithL1RPC(l1RPC), WithPrivateKey(pkHex))
	workDir := runner.GetWorkDir()

	// Initialize intent and deploy chain
	intent, _ := cliInitIntent(t, runner, l1ChainID, []common.Hash{l2ChainID.Bytes32()})

	if intent.SuperchainRoles == nil {
		intent.SuperchainRoles = &addresses.SuperchainRoles{}
	}

	l1ChainIDBig := big.NewInt(int64(l1ChainID))
	intent.SuperchainRoles.SuperchainProxyAdminOwner = superchainProxyAdminOwner
	intent.SuperchainRoles.SuperchainGuardian = shared.AddrFor(t, dk, devkeys.SuperchainConfigGuardianKey.Key(l1ChainIDBig))
	intent.SuperchainRoles.ProtocolVersionsOwner = superchainProxyAdminOwner
	intent.SuperchainRoles.Challenger = shared.AddrFor(t, dk, devkeys.ChallengerRole.Key(l1ChainIDBig))

	for _, chain := range intent.Chains {
		chain.Roles.L1ProxyAdminOwner = superchainProxyAdminOwner
		chain.Roles.L2ProxyAdminOwner = shared.AddrFor(t, dk, devkeys.L2ProxyAdminOwnerRole.Key(l1ChainIDBig))
		chain.Roles.SystemConfigOwner = superchainProxyAdminOwner
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

	// Populate the state with predeployed superchain and implementations
	// so the pipeline knows about them
	st, err := pipeline.ReadState(workDir)
	require.NoError(t, err)

	// Set superchain deployment addresses
	if st.SuperchainDeployment == nil {
		st.SuperchainDeployment = &addresses.SuperchainContracts{
			SuperchainConfigProxy:    superchainOut.SuperchainConfigProxy,
			SuperchainConfigImpl:     superchainOut.SuperchainConfigImpl,
			ProtocolVersionsProxy:    superchainOut.ProtocolVersionsProxy,
			ProtocolVersionsImpl:     superchainOut.ProtocolVersionsImpl,
			SuperchainProxyAdminImpl: superchainOut.SuperchainProxyAdmin,
		}
	}

	// Set implementations deployment addresses
	if st.ImplementationsDeployment == nil {
		st.ImplementationsDeployment = &addresses.ImplementationsContracts{
			OpcmImpl:                         impls.OpcmV2,
			OpcmContainerImpl:                impls.OpcmContainer,
			OpcmUtilsImpl:                    impls.OpcmUtils,
			OpcmMigratorImpl:                 impls.OpcmMigrator,
			OptimismPortalInteropImpl:        impls.OptimismPortalInteropImpl,
			OptimismPortalImpl:               impls.OptimismPortalImpl,
			DelayedWethImpl:                  impls.DelayedWETHImpl,
			EthLockboxImpl:                   impls.ETHLockboxImpl,
			SystemConfigImpl:                 impls.SystemConfigImpl,
			L1CrossDomainMessengerImpl:       impls.L1CrossDomainMessengerImpl,
			L1Erc721BridgeImpl:               impls.L1ERC721BridgeImpl,
			L1StandardBridgeImpl:             impls.L1StandardBridgeImpl,
			OptimismMintableErc20FactoryImpl: impls.OptimismMintableERC20FactoryImpl,
			DisputeGameFactoryImpl:           impls.DisputeGameFactoryImpl,
			AnchorStateRegistryImpl:          impls.AnchorStateRegistryImpl,
			PreimageOracleImpl:               impls.PreimageOracleSingleton,
			MipsImpl:                         impls.MipsSingleton,
			FaultDisputeGameImpl:             impls.FaultDisputeGameImpl,
			PermissionedDisputeGameImpl:      impls.PermissionedDisputeGameImpl,
			OpcmDeployerImpl:                 impls.OpcmDeployer,
			OpcmGameTypeAdderImpl:            impls.OpcmGameTypeAdder,
			OpcmUpgraderImpl:                 impls.OpcmUpgrader,
			OpcmInteropMigratorImpl:          impls.OpcmInteropMigrator,
			OpcmStandardValidatorImpl:        impls.OpcmStandardValidator,
		}
	}
	require.NoError(t, pipeline.WriteState(workDir, st))

	// Set global deploy overrides with devFeatureBitmap for OPCM V2
	intent.GlobalDeployOverrides = map[string]any{
		"devFeatureBitmap": devFeatureBitmap,
	}
	// Since we are enabling Interop in the bitmap we enable the UseInterop flag
	intent.UseInterop = true

	require.NoError(t, intent.WriteToFile(filepath.Join(workDir, "intent.toml")))

	// Apply deployment
	// Note: Validation will run automatically but may find expected errors for migration test deployments
	// (e.g., custom dev features, non-standard configurations). We verify deployment succeeded despite validation errors.
	applyCtx, applyCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer applyCancel()
	output, runErr := runner.RunWithNetwork(applyCtx, []string{
		"apply",
		"--deployment-target", "live",
		"--workdir", workDir,
		"--validate", "auto",
	}, nil)

	// Verify deployment succeeded regardless of validation errors
	st, err = pipeline.ReadState(workDir)
	require.NoError(t, err, "State should be readable after apply")
	require.NotNil(t, st.AppliedIntent, "Applied intent should exist")
	require.Len(t, st.Chains, 1, "Should have one chain deployed")

	// If there was an error, it should be validation-related, not a deployment failure
	if runErr != nil {
		require.Contains(t, output, "validation", "Error should be validation-related, not deployment failure")
		require.Contains(t, runErr.Error(), "validation", "Error should mention validation")
	}

	systemConfigProxy := st.Chains[0].SystemConfigProxy

	// Run migrate-v2 command
	// Note: dispute-game-type should be 0 (Cannon), not 4 (SuperCannon)
	// Game type 4 is only for starting-respected-game-type
	migrateOutput := runner.ExpectSuccessWithNetwork(t, []string{
		"manage",
		"migrate",
		"--l1-proxy-admin-owner-address", superchainProxyAdminOwner.Hex(),
		"--l1-rpc-url", l1RPC,
		"--private-key", pkHex,
		"--opcm-impl-address", impls.OpcmV2.Hex(),
		"--system-config-proxy-address", systemConfigProxy.Hex(),
		"--dispute-game-enabled",
		"--dispute-game-type", "0", // GameTypeCannon (0), not SuperCannon (4)
		"--dispute-absolute-prestate", "0x0000000000000000000000000000000000000000000000000000000000000abc",
		"--starting-anchor-root", "0x0000000000000000000000000000000000000000000000000000000000000def",
		"--starting-anchor-l2-sequence-number", "1",
		"--starting-respected-game-type", "5", // GameTypeSuperPermissionedCannon (5)
		"--initial-bond", "1000000000000000000",
	}, nil)

	// Parse output to verify DisputeGameFactory was deployed
	// Find the JSON output by looking for the opening brace
	var migrationOutput manage.InteropMigrationOutput
	jsonStart := strings.Index(migrateOutput, "{")
	if jsonStart == -1 {
		t.Logf("Full output length: %d", len(migrateOutput))
		t.Logf("Full output: %q", migrateOutput)
		t.Fatalf("No JSON output found in output")
	}
	// Find the end of the JSON object
	jsonEnd := strings.Index(migrateOutput[jsonStart:], "}") + jsonStart + 1
	jsonOutput := migrateOutput[jsonStart:jsonEnd]
	err = json.Unmarshal([]byte(jsonOutput), &migrationOutput)
	require.NoError(t, err, "Failed to parse migration output: %s", jsonOutput)
	require.NotEqual(t, common.Address{}, migrationOutput.DisputeGameFactory, "DisputeGameFactory should be deployed")
}
