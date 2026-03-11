package manage

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log/slog"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestInteropMigration(t *testing.T) {
	lgr := testlog.Logger(t, slog.LevelDebug)

	forkedL1, stopL1, err := devnet.NewForkedSepolia(lgr)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, stopL1())
	})
	l1RPC := forkedL1.RPCUrl()

	loc, afactsFS := testutil.LocalArtifacts(t)
	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	_, pk, dk := shared.DefaultPrivkey(t)

	l1ChainID := big.NewInt(11155111) // Sepolia
	l2ChainID := uint256.NewInt(12345)

	tests := []struct {
		name       string
		devFeature common.Hash
	}{
		{"opcm-v1", common.Hash{}},
		{"opcm-v2", deployer.OPCMV2DevFlag},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Deploy a complete chain using ApplyPipeline - this ensures all addresses are properly connected
			intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, 30_000_000)

			// Set dev features for this test
			devBitmap := deployer.EnableDevFeature(tt.devFeature, deployer.OptimismPortalInteropDevFlag)
			intent.GlobalDeployOverrides = map[string]any{
				"devFeatureBitmap": devBitmap,
			}

			// Since we are enabling Interop in the bitmap we enable the UseInterop flag
			intent.UseInterop = true

			err := deployer.ApplyPipeline(ctx, deployer.ApplyPipelineOpts{
				DeploymentTarget:   deployer.DeploymentTargetLive,
				L1RPCUrl:           l1RPC,
				DeployerPrivateKey: pk,
				Intent:             intent,
				State:              st,
				Logger:             lgr,
				StateWriter:        pipeline.NoopStateWriter(),
				CacheDir:           testCacheDir,
			})
			require.NoError(t, err, "Failed to deploy chain")

			// Get addresses from the deployed state
			require.Len(t, st.Chains, 1, "Expected one chain to be deployed")
			chainState := st.Chains[0]
			systemConfigProxy := chainState.SystemConfigProxy

			// Get the L1ProxyAdminOwner from the intent
			l1ProxyAdminOwner := intent.Chains[0].Roles.L1ProxyAdminOwner

			t.Logf("SystemConfigProxy: %s", systemConfigProxy.Hex())
			t.Logf("L1ProxyAdminOwner: %s", l1ProxyAdminOwner.Hex())

			rpcClient, err := rpc.Dial(l1RPC)
			require.NoError(t, err)

			var opcmAddr common.Address
			if deployer.IsDevFeatureEnabled(tt.devFeature, deployer.OPCMV2DevFlag) {
				require.NotEqual(t, common.Address{}, st.ImplementationsDeployment.OpcmV2Impl, "OPCM V2 address should be set")
				opcmAddr = st.ImplementationsDeployment.OpcmV2Impl
				t.Logf("OPCM V2: %s", opcmAddr.Hex())
			} else {
				require.NotEqual(t, common.Address{}, st.ImplementationsDeployment.OpcmImpl, "OPCM V1 address should be set")
				opcmAddr = st.ImplementationsDeployment.OpcmImpl
				t.Logf("OPCM V1: %s", opcmAddr.Hex())
			}

			// Deploy DummyCaller at l1ProxyAdminOwner for the OPCM
			shared.DeployDummyCaller(t, rpcClient, afactsFS, l1ProxyAdminOwner, opcmAddr)

			bcast := new(broadcaster.CalldataBroadcaster)
			host, err := env.DefaultForkedScriptHost(
				ctx,
				bcast,
				lgr,
				l1ProxyAdminOwner,
				afactsFS,
				rpcClient,
			)
			require.NoError(t, err)

			var input InteropMigrationInput

			if deployer.IsDevFeatureEnabled(tt.devFeature, deployer.OPCMV2DevFlag) {
				// OPCM V2 path
				// Note: No need to call upgradeChainV2 since ApplyPipeline already deploys a fully initialized chain

				// Prepare game args for V2 - ABI encode the prestate
				bytes32Type, err := abi.NewType("bytes32", "", nil)
				require.NoError(t, err)
				testPrestate := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000abc")
				gameArgs, err := abi.Arguments{{Type: bytes32Type}}.Pack(testPrestate)
				require.NoError(t, err)

				// Define game type constants matching Solidity GameTypes library
				const (
					GameTypeCannon      = uint32(0)
					GameTypeSuperCannon = uint32(4)
				)

				input = InteropMigrationInput{
					Prank: l1ProxyAdminOwner,
					Opcm:  opcmAddr,
					MigrateInputV2: &MigrateInputV2{
						ChainSystemConfigs: []common.Address{
							systemConfigProxy,
						},
						DisputeGameConfigs: []DisputeGameConfig{
							{
								Enabled:  true,
								InitBond: big.NewInt(1000000000000000000), // 1 ETH
								GameType: GameTypeCannon,
								GameArgs: gameArgs,
							},
						},
						StartingAnchorRoot: Proposal{
							Root:             common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000def"),
							L2SequenceNumber: big.NewInt(1),
						},
						StartingRespectedGameType: GameTypeSuperCannon,
					},
				}
			} else {
				// OPCM V1 path
				// Note: No need to call upgradeChainV1 since ApplyPipeline already deploys a fully initialized chain

				// Get proposer and challenger from devkeys
				proposer, err := dk.Address(devkeys.ProposerRole.Key(l1ChainID))
				require.NoError(t, err)
				challenger, err := dk.Address(devkeys.ChallengerRole.Key(l1ChainID))
				require.NoError(t, err)

				input = InteropMigrationInput{
					Prank: l1ProxyAdminOwner,
					Opcm:  opcmAddr,
					MigrateInputV1: &MigrateInputV1{
						UsePermissionlessGame: true,
						StartingAnchorRoot: Proposal{
							Root:             common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000def"),
							L2SequenceNumber: big.NewInt(1),
						},
						GameParameters: GameParameters{
							Proposer:         proposer,
							Challenger:       challenger,
							MaxGameDepth:     73,
							SplitDepth:       30,
							InitBond:         big.NewInt(1000000000000000000), // 1 ETH
							ClockExtension:   10800,
							MaxClockDuration: 302400,
						},
						OpChainConfigs: []OPChainConfig{
							{
								SystemConfigProxy:  systemConfigProxy,
								CannonPrestate:     common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000abc"),
								CannonKonaPrestate: common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000fed"),
							},
						},
					},
				}
			}

			// Execute Migration
			output, err := Migrate(host, input)
			require.NoError(t, err)
			require.NotEqual(t, common.Address{}, output.DisputeGameFactory)

			dump, err := bcast.Dump()
			require.NoError(t, err)
			require.Len(t, dump, 1, "Should have one transaction (migration)")
			require.True(t, dump[0].Value.ToInt().Cmp(common.Big0) == 0, "Transaction value should be zero")
			require.Equal(t, l1ProxyAdminOwner, *dump[0].To, "Transaction should be sent to prank address")
		})
	}
}

func TestMigrateCLIV1Flags(t *testing.T) {
	app := cli.NewApp()
	flagSet := flag.NewFlagSet("test-migrate-v1", flag.ContinueOnError)

	// Set V1-specific flags
	flagSet.String(OPCMImplFlag.Name, "0xaf334f4537e87f5155d135392ff6d52f1866465e", "doc")
	flagSet.String(SystemConfigProxyFlag.Name, "0x034edD2A225f7f429A63E0f1D2084B9E0A93b538", "doc")
	flagSet.Bool(PermissionlessFlag.Name, true, "doc")
	flagSet.String(ProposerFlag.Name, "0x1111111111111111111111111111111111111111", "doc")
	flagSet.String(ChallengerFlag.Name, "0x2222222222222222222222222222222222222222", "doc")
	flagSet.String(StartingAnchorRootFlag.Name, "0x0000000000000000000000000000000000000000000000000000000000000abc", "doc")
	flagSet.Uint64(StartingAnchorL2SequenceNumberFlag.Name, 1, "doc")
	flagSet.Uint64(DisputeMaxGameDepthFlag.Name, 73, "doc")
	flagSet.Uint64(DisputeSplitDepthFlag.Name, 30, "doc")
	flagSet.String(InitialBondFlag.Name, "1000000000000000000", "doc")
	flagSet.Uint64(DisputeClockExtensionFlag.Name, 10800, "doc")
	flagSet.Uint64(DisputeMaxClockDurationFlag.Name, 302400, "doc")
	flagSet.String(DisputeAbsolutePrestateCannonFlag.Name, "0x0000000000000000000000000000000000000000000000000000000000000def", "doc")
	flagSet.String(DisputeAbsolutePrestateCannonKonaFlag.Name, "0x0000000000000000000000000000000000000000000000000000000000000fed", "doc")

	ctx := cli.NewContext(app, flagSet, nil)

	// Parse V1 flags
	opcmAddr := common.HexToAddress(ctx.String(OPCMImplFlag.Name))
	systemConfigProxy := common.HexToAddress(ctx.String(SystemConfigProxyFlag.Name))
	permissionless := ctx.Bool(PermissionlessFlag.Name)
	proposer := common.HexToAddress(ctx.String(ProposerFlag.Name))
	challenger := common.HexToAddress(ctx.String(ChallengerFlag.Name))
	startingAnchorRoot := common.HexToHash(ctx.String(StartingAnchorRootFlag.Name))
	startingAnchorL2SeqNum := ctx.Uint64(StartingAnchorL2SequenceNumberFlag.Name)
	maxGameDepth := ctx.Uint64(DisputeMaxGameDepthFlag.Name)
	splitDepth := ctx.Uint64(DisputeSplitDepthFlag.Name)
	initBondStr := ctx.String(InitialBondFlag.Name)
	initBond, ok := new(big.Int).SetString(initBondStr, 10)
	require.True(t, ok)
	clockExtension := ctx.Uint64(DisputeClockExtensionFlag.Name)
	maxClockDuration := ctx.Uint64(DisputeMaxClockDurationFlag.Name)
	cannonPrestate := common.HexToHash(ctx.String(DisputeAbsolutePrestateCannonFlag.Name))
	cannonKonaPrestate := common.HexToHash(ctx.String(DisputeAbsolutePrestateCannonKonaFlag.Name))

	// Verify values
	require.Equal(t, common.HexToAddress("0xaf334f4537e87f5155d135392ff6d52f1866465e"), opcmAddr)
	require.Equal(t, common.HexToAddress("0x034edD2A225f7f429A63E0f1D2084B9E0A93b538"), systemConfigProxy)
	require.True(t, permissionless)
	require.Equal(t, common.HexToAddress("0x1111111111111111111111111111111111111111"), proposer)
	require.Equal(t, common.HexToAddress("0x2222222222222222222222222222222222222222"), challenger)
	require.Equal(t, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000abc"), startingAnchorRoot)
	require.Equal(t, uint64(1), startingAnchorL2SeqNum)
	require.Equal(t, uint64(73), maxGameDepth)
	require.Equal(t, uint64(30), splitDepth)
	require.Equal(t, big.NewInt(1000000000000000000), initBond)
	require.Equal(t, uint64(10800), clockExtension)
	require.Equal(t, uint64(302400), maxClockDuration)
	require.Equal(t, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000def"), cannonPrestate)
	require.Equal(t, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000fed"), cannonKonaPrestate)
}

func TestMigrateCLIV2Flags(t *testing.T) {
	app := cli.NewApp()
	flagSet := flag.NewFlagSet("test-migrate-v2", flag.ContinueOnError)

	// Set V2-specific flags
	flagSet.String(OPCMImplFlag.Name, "0xaf334f4537e87f5155d135392ff6d52f1866465e", "doc")
	flagSet.String(SystemConfigProxyFlag.Name, "0x034edD2A225f7f429A63E0f1D2084B9E0A93b538", "doc")
	flagSet.Bool(MigrateDisputeGameEnabledFlag.Name, true, "doc")
	flagSet.String(InitialBondFlag.Name, "1000000000000000000", "doc")
	flagSet.Uint64(DisputeGameTypeFlag.Name, 0, "doc")
	flagSet.String(DisputeAbsolutePrestateFlag.Name, "0x0000000000000000000000000000000000000000000000000000000000000abc", "doc")
	flagSet.String(StartingAnchorRootFlag.Name, "0x0000000000000000000000000000000000000000000000000000000000000def", "doc")
	flagSet.Uint64(StartingAnchorL2SequenceNumberFlag.Name, 1, "doc")
	flagSet.Uint64(MigrateStartingRespectedGameTypeFlag.Name, 0, "doc")

	ctx := cli.NewContext(app, flagSet, nil)

	// Parse V2 flags
	opcmAddr := common.HexToAddress(ctx.String(OPCMImplFlag.Name))
	systemConfigProxy := common.HexToAddress(ctx.String(SystemConfigProxyFlag.Name))
	disputeGameEnabled := ctx.Bool(MigrateDisputeGameEnabledFlag.Name)
	initBondStr := ctx.String(InitialBondFlag.Name)
	initBond, ok := new(big.Int).SetString(initBondStr, 10)
	require.True(t, ok)
	gameType := uint32(ctx.Uint64(DisputeGameTypeFlag.Name))
	gameArgs := common.FromHex(ctx.String(DisputeAbsolutePrestateFlag.Name))
	startingAnchorRoot := common.HexToHash(ctx.String(StartingAnchorRootFlag.Name))
	startingAnchorL2SeqNum := ctx.Uint64(StartingAnchorL2SequenceNumberFlag.Name)
	startingRespectedGameType := uint32(ctx.Uint64(MigrateStartingRespectedGameTypeFlag.Name))

	// Verify values
	require.Equal(t, common.HexToAddress("0xaf334f4537e87f5155d135392ff6d52f1866465e"), opcmAddr)
	require.Equal(t, common.HexToAddress("0x034edD2A225f7f429A63E0f1D2084B9E0A93b538"), systemConfigProxy)
	require.True(t, disputeGameEnabled)
	require.Equal(t, big.NewInt(1000000000000000000), initBond)
	require.Equal(t, uint32(0), gameType)
	require.Equal(t, common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000abc"), gameArgs)
	require.Equal(t, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000def"), startingAnchorRoot)
	require.Equal(t, uint64(1), startingAnchorL2SeqNum)
	require.Equal(t, uint32(0), startingRespectedGameType)
}

func TestMigrateCLIV2Uint32Overflow(t *testing.T) {
	testCases := []struct {
		name                      string
		disputeGameType           uint64
		startingRespectedGameType uint64
		expectError               bool
		expectedErrContains       string
	}{
		{
			name:                      "valid uint32 values",
			disputeGameType:           0,
			startingRespectedGameType: 4,
			expectError:               false,
		},
		{
			name:                      "max valid uint32 values",
			disputeGameType:           0xFFFFFFFF,
			startingRespectedGameType: 0xFFFFFFFF,
			expectError:               false,
		},
		{
			name:                      "disputeGameType overflow",
			disputeGameType:           0x100000000, // 2^32
			startingRespectedGameType: 4,
			expectError:               true,
			expectedErrContains:       "disputeGameType",
		},
		{
			name:                      "startingRespectedGameType overflow",
			disputeGameType:           0,
			startingRespectedGameType: 0x100000000, // 2^32
			expectError:               true,
			expectedErrContains:       "startingRespectedGameType",
		},
		{
			name:                      "disputeGameType large overflow",
			disputeGameType:           0xFFFFFFFFFFFFFFFF, // max uint64
			startingRespectedGameType: 4,
			expectError:               true,
			expectedErrContains:       "disputeGameType",
		},
		{
			name:                      "startingRespectedGameType large overflow",
			disputeGameType:           0,
			startingRespectedGameType: 0xFFFFFFFFFFFFFFFF, // max uint64
			expectError:               true,
			expectedErrContains:       "startingRespectedGameType",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := cli.NewApp()
			flagSet := flag.NewFlagSet(fmt.Sprintf("test-%s", tc.name), flag.ContinueOnError)

			// Set all required flags
			flagSet.String(deployer.L1RPCURLFlag.Name, "http://localhost:8545", "doc")
			flagSet.String(deployer.PrivateKeyFlag.Name, "0000000000000000000000000000000000000000000000000000000000000001", "doc")
			flagSet.String(OPCMImplFlag.Name, "0xaf334f4537e87f5155d135392ff6d52f1866465e", "doc")
			flagSet.String(SystemConfigProxyFlag.Name, "0x034edD2A225f7f429A63E0f1D2084B9E0A93b538", "doc")
			flagSet.String(L1ProxyAdminOwnerFlag.Name, "0x1Eb2fFc903729a0F03966B917003800b145F56E2", "doc")
			flagSet.Bool(MigrateDisputeGameEnabledFlag.Name, true, "doc")
			flagSet.String(InitialBondFlag.Name, "1000000000000000000", "doc")
			flagSet.Uint64(DisputeGameTypeFlag.Name, tc.disputeGameType, "doc")
			flagSet.String(DisputeAbsolutePrestateFlag.Name, "0x0000000000000000000000000000000000000000000000000000000000000abc", "doc")
			flagSet.String(StartingAnchorRootFlag.Name, "0x0000000000000000000000000000000000000000000000000000000000000def", "doc")
			flagSet.Uint64(StartingAnchorL2SequenceNumberFlag.Name, 1, "doc")
			flagSet.Uint64(MigrateStartingRespectedGameTypeFlag.Name, tc.startingRespectedGameType, "doc")
			flagSet.String(deployer.ArtifactsLocatorFlag.Name, "tag://op-contracts/v1.6.0", "doc")
			flagSet.String(deployer.CacheDirFlag.Name, t.TempDir(), "doc")

			ctx := cli.NewContext(app, flagSet, nil)

			// Parse the flags to validate uint32 bounds
			disputeGameTypeU64 := ctx.Uint64(DisputeGameTypeFlag.Name)
			startingRespectedGameTypeU64 := ctx.Uint64(MigrateStartingRespectedGameTypeFlag.Name)

			// Simulate the validation logic from MigrateCLIV2
			var validationErr error
			if disputeGameTypeU64 > 0xFFFFFFFF {
				validationErr = fmt.Errorf("disputeGameType %d exceeds uint32 max value", disputeGameTypeU64)
			}
			if startingRespectedGameTypeU64 > 0xFFFFFFFF {
				validationErr = fmt.Errorf("startingRespectedGameType %d exceeds uint32 max value", startingRespectedGameTypeU64)
			}

			if tc.expectError {
				require.Error(t, validationErr)
				require.Contains(t, validationErr.Error(), tc.expectedErrContains)
			} else {
				require.NoError(t, validationErr)
				// Verify casting to uint32 is safe
				disputeGameType := uint32(disputeGameTypeU64)
				startingRespectedGameType := uint32(startingRespectedGameTypeU64)
				require.Equal(t, tc.disputeGameType, uint64(disputeGameType))
				require.Equal(t, tc.startingRespectedGameType, uint64(startingRespectedGameType))
			}
		})
	}
}

func TestEncodedMigrateInputV1(t *testing.T) {
	input := &InteropMigrationInput{
		Prank: common.Address{0xaa},
		Opcm:  common.Address{0xbb},
		MigrateInputV1: &MigrateInputV1{
			UsePermissionlessGame: true,
			StartingAnchorRoot: Proposal{
				Root:             common.Hash{0xde},
				L2SequenceNumber: big.NewInt(100),
			},
			GameParameters: GameParameters{
				Proposer:         common.Address{0x11},
				Challenger:       common.Address{0x22},
				MaxGameDepth:     73,
				SplitDepth:       30,
				InitBond:         big.NewInt(1000),
				ClockExtension:   10800,
				MaxClockDuration: 302400,
			},
			OpChainConfigs: []OPChainConfig{
				{
					SystemConfigProxy:  common.Address{0x01},
					CannonPrestate:     common.Hash{0xab},
					CannonKonaPrestate: common.Hash{0xcd},
				},
			},
		},
	}

	data, err := input.EncodedMigrateInputV1()
	require.NoError(t, err)
	require.NotEmpty(t, data)

	expected := "0000000000000000000000000000000000000000000000000000000000000020" + // offset to tuple
		"0000000000000000000000000000000000000000000000000000000000000001" + // usePermissionlessGame (true)
		"de00000000000000000000000000000000000000000000000000000000000000" + // startingAnchorRoot.root
		"0000000000000000000000000000000000000000000000000000000000000064" + // startingAnchorRoot.l2SequenceNumber (100)
		"0000000000000000000000001100000000000000000000000000000000000000" + // gameParameters.proposer
		"0000000000000000000000002200000000000000000000000000000000000000" + // gameParameters.challenger
		"0000000000000000000000000000000000000000000000000000000000000049" + // gameParameters.maxGameDepth (73)
		"000000000000000000000000000000000000000000000000000000000000001e" + // gameParameters.splitDepth (30)
		"00000000000000000000000000000000000000000000000000000000000003e8" + // gameParameters.initBond (1000)
		"0000000000000000000000000000000000000000000000000000000000002a30" + // gameParameters.clockExtension (10800)
		"0000000000000000000000000000000000000000000000000000000000049d40" + // gameParameters.maxClockDuration (302400)
		"0000000000000000000000000000000000000000000000000000000000000160" + // offset to opChainConfigs (11 words * 32 = 352 = 0x160)
		"0000000000000000000000000000000000000000000000000000000000000001" + // opChainConfigs.length (1)
		"0000000000000000000000000100000000000000000000000000000000000000" + // opChainConfigs[0].systemConfigProxy
		"ab00000000000000000000000000000000000000000000000000000000000000" + // opChainConfigs[0].cannonPrestate
		"cd00000000000000000000000000000000000000000000000000000000000000" // opChainConfigs[0].cannonKonaPrestate

	require.Equal(t, expected, hex.EncodeToString(data))
}

func TestEncodedMigrateInputV2(t *testing.T) {
	// Prepare game args - ABI encode a prestate hash
	bytes32Type, err := abi.NewType("bytes32", "", nil)
	require.NoError(t, err)
	testPrestate := common.HexToHash("0xaa00000000000000000000000000000000000000000000000000000000000000")
	gameArgs, err := abi.Arguments{{Type: bytes32Type}}.Pack(testPrestate)
	require.NoError(t, err)

	input := &InteropMigrationInput{
		Prank: common.Address{0xaa},
		Opcm:  common.Address{0xbb},
		MigrateInputV2: &MigrateInputV2{
			ChainSystemConfigs: []common.Address{
				{0x01},
			},
			DisputeGameConfigs: []DisputeGameConfig{
				{
					Enabled:  true,
					InitBond: big.NewInt(1000),
					GameType: 4,
					GameArgs: gameArgs,
				},
			},
			StartingAnchorRoot: Proposal{
				Root:             common.Hash{0xde},
				L2SequenceNumber: big.NewInt(100),
			},
			StartingRespectedGameType: 4,
		},
	}

	data, err := input.EncodedMigrateInputV2()
	require.NoError(t, err)
	require.NotEmpty(t, data)

	expected := "0000000000000000000000000000000000000000000000000000000000000020" + // offset to tuple
		"00000000000000000000000000000000000000000000000000000000000000a0" + // offset to chainSystemConfigs (5 words * 32 = 160 = 0xa0)
		"00000000000000000000000000000000000000000000000000000000000000e0" + // offset to disputeGameConfigs (0xa0 + 0x40)
		"de00000000000000000000000000000000000000000000000000000000000000" + // startingAnchorRoot.root
		"0000000000000000000000000000000000000000000000000000000000000064" + // startingAnchorRoot.l2SequenceNumber (100)
		"0000000000000000000000000000000000000000000000000000000000000004" + // startingRespectedGameType (4)
		"0000000000000000000000000000000000000000000000000000000000000001" + // chainSystemConfigs.length (1)
		"0000000000000000000000000100000000000000000000000000000000000000" + // chainSystemConfigs[0]
		"0000000000000000000000000000000000000000000000000000000000000001" + // disputeGameConfigs.length (1)
		"0000000000000000000000000000000000000000000000000000000000000020" + // offset to disputeGameConfigs[0]
		"0000000000000000000000000000000000000000000000000000000000000001" + // disputeGameConfigs[0].enabled
		"00000000000000000000000000000000000000000000000000000000000003e8" + // disputeGameConfigs[0].initBond (1000)
		"0000000000000000000000000000000000000000000000000000000000000004" + // disputeGameConfigs[0].gameType (4)
		"0000000000000000000000000000000000000000000000000000000000000080" + // offset to gameArgs
		"0000000000000000000000000000000000000000000000000000000000000020" + // gameArgs.length (32 bytes)
		"aa00000000000000000000000000000000000000000000000000000000000000" // gameArgs data (prestate)

	require.Equal(t, expected, hex.EncodeToString(data))
}
