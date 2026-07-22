package manage

import (
	"context"
	"encoding/hex"
	"log/slog"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
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

	// Deploy a complete chain using ApplyPipeline
	intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, 30_000_000)

	devBitmap := devfeatures.EnableDevFeature(common.Hash{}, devfeatures.OptimismPortalInteropFlag)
	intent.GlobalDeployOverrides = map[string]any{
		"devFeatureBitmap": devBitmap,
	}

	// Since we are enabling Interop in the bitmap we enable the UseInterop flag
	intent.UseInterop = true

	err = deployer.ApplyPipeline(ctx, deployer.ApplyPipelineOpts{
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

	require.NotEqual(t, common.Address{}, st.ImplementationsDeployment.OpcmV2Impl, "OPCM V2 address should be set")
	opcmAddr := st.ImplementationsDeployment.OpcmV2Impl
	t.Logf("OPCM V2: %s", opcmAddr.Hex())

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

	// Prepare game args for V2 - ABI encode the prestate
	bytes32Type, err := abi.NewType("bytes32", "", nil)
	require.NoError(t, err)
	testPrestate := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000abc")
	gameArgs, err := abi.Arguments{{Type: bytes32Type}}.Pack(testPrestate)
	require.NoError(t, err)

	// Define game type constants matching Solidity GameTypes library.
	const (
		GameTypeCannon          = uint32(0)
		GameTypeSuperCannonKona = uint32(9)
	)

	// The registered game type and the starting respected game type are intentionally
	// different: the migrator does not validate disputeGameConfigs[i].gameType (see
	// OPContractsManagerMigrator.migrate), so this exercises the permissive-registration
	// invariant alongside the strict respected-type check.
	input := InteropMigrationInput{
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
			StartingRespectedGameType: GameTypeSuperCannonKona,
		},
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
}

func TestCommandsDoNotIncludeMigrate(t *testing.T) {
	for _, command := range Commands {
		require.NotEqual(t, "migrate", command.Name)
	}
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
					GameType: 9,
					GameArgs: gameArgs,
				},
			},
			StartingAnchorRoot: Proposal{
				Root:             common.Hash{0xde},
				L2SequenceNumber: big.NewInt(100),
			},
			StartingRespectedGameType: 9,
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
		"0000000000000000000000000000000000000000000000000000000000000009" + // startingRespectedGameType (9, SUPER_CANNON_KONA)
		"0000000000000000000000000000000000000000000000000000000000000001" + // chainSystemConfigs.length (1)
		"0000000000000000000000000100000000000000000000000000000000000000" + // chainSystemConfigs[0]
		"0000000000000000000000000000000000000000000000000000000000000001" + // disputeGameConfigs.length (1)
		"0000000000000000000000000000000000000000000000000000000000000020" + // offset to disputeGameConfigs[0]
		"0000000000000000000000000000000000000000000000000000000000000001" + // disputeGameConfigs[0].enabled
		"00000000000000000000000000000000000000000000000000000000000003e8" + // disputeGameConfigs[0].initBond (1000)
		"0000000000000000000000000000000000000000000000000000000000000009" + // disputeGameConfigs[0].gameType (9, SUPER_CANNON_KONA)
		"0000000000000000000000000000000000000000000000000000000000000080" + // offset to gameArgs
		"0000000000000000000000000000000000000000000000000000000000000020" + // gameArgs.length (32 bytes)
		"aa00000000000000000000000000000000000000000000000000000000000000" // gameArgs data (prestate)

	require.Equal(t, expected, hex.EncodeToString(data))
}
