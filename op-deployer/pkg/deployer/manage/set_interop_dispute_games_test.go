package manage

import (
	"context"
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

// TestSetInteropDisputeGames is a forked-Sepolia end-to-end test of the full
// Go -> forge script -> OPCMv2.setInteropDisputeGames path. It deploys a chain with BOTH the
// interop and ZK dev features enabled (so the OPCM ships a ZKDisputeGame impl), migrates it into
// an interop set with super-cannon fault proofs, then swaps the shared dispute games to the ZK
// dispute game. The script's own checkOutput asserts on-chain that the respected game type became
// ZK, so a non-erroring run proves the swap took effect.
func TestSetInteropDisputeGames(t *testing.T) {
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

	intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, 30_000_000)

	// Enable BOTH interop (for migrate) and ZK (so DeployImplementations ships the ZKDisputeGame
	// impl into the OPCM container, which setInteropDisputeGames requires).
	devBitmap := devfeatures.EnableDevFeature(common.Hash{}, devfeatures.OptimismPortalInteropFlag)
	devBitmap = devfeatures.EnableDevFeature(devBitmap, devfeatures.ZKDisputeGameFlag)
	intent.GlobalDeployOverrides = map[string]any{
		"devFeatureBitmap": devBitmap,
	}
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

	require.Len(t, st.Chains, 1, "Expected one chain to be deployed")
	systemConfigProxy := st.Chains[0].SystemConfigProxy
	l1ProxyAdminOwner := intent.Chains[0].Roles.L1ProxyAdminOwner

	require.NotEqual(t, common.Address{}, st.ImplementationsDeployment.OpcmV2Impl, "OPCM V2 address should be set")
	opcmAddr := st.ImplementationsDeployment.OpcmV2Impl

	rpcClient, err := rpc.Dial(l1RPC)
	require.NoError(t, err)

	shared.DeployDummyCaller(t, rpcClient, afactsFS, l1ProxyAdminOwner, opcmAddr)

	bcast := new(broadcaster.CalldataBroadcaster)
	host, err := env.DefaultForkedScriptHost(ctx, bcast, lgr, l1ProxyAdminOwner, afactsFS, rpcClient)
	require.NoError(t, err)

	const (
		gameTypeSuperCannonKona = uint32(9)
		gameTypeZK              = uint32(10)
	)

	startingAnchorRoot := Proposal{
		Root:             common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000def"),
		L2SequenceNumber: big.NewInt(1),
	}

	// Step 1: migrate the chain into an interop set with super-cannon-kona fault proofs.
	bytes32Type, err := abi.NewType("bytes32", "", nil)
	require.NoError(t, err)
	faultArgs, err := abi.Arguments{{Type: bytes32Type}}.Pack(
		common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000abc"),
	)
	require.NoError(t, err)

	_, err = Migrate(host, InteropMigrationInput{
		Prank: l1ProxyAdminOwner,
		Opcm:  opcmAddr,
		MigrateInputV2: &MigrateInputV2{
			ChainSystemConfigs: []common.Address{systemConfigProxy},
			DisputeGameConfigs: []DisputeGameConfig{
				{
					Enabled:  true,
					InitBond: big.NewInt(1000000000000000000),
					GameType: gameTypeSuperCannonKona,
					GameArgs: faultArgs,
				},
			},
			StartingAnchorRoot:        startingAnchorRoot,
			StartingRespectedGameType: gameTypeSuperCannonKona,
		},
	})
	require.NoError(t, err, "interop migration failed")

	// Step 2: swap the shared dispute games to ZK. encodeZKGameArgs produces the 5-field config the
	// contract's _makeGameArgs decodes; the source super game is cleared and ZK is enabled.
	zkArgs, err := encodeZKGameArgs(zkDisputeGameConfig{
		AbsolutePrestate:     common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000abc"),
		Verifier:             common.HexToAddress("0x000000000000000000000000000000000000bEEF"),
		MaxChallengeDuration: 7 * 24 * 60 * 60,
		MaxProveDuration:     3 * 24 * 60 * 60,
		ChallengerBond:       big.NewInt(1000000000000000000),
	})
	require.NoError(t, err)

	output, err := SetInteropDisputeGames(host, InteropMigrationInput{
		Prank: l1ProxyAdminOwner,
		Opcm:  opcmAddr,
		MigrateInputV2: &MigrateInputV2{
			ChainSystemConfigs: []common.Address{systemConfigProxy},
			DisputeGameConfigs: []DisputeGameConfig{
				// Clear the retired source super game.
				{Enabled: false, InitBond: big.NewInt(0), GameType: gameTypeSuperCannonKona, GameArgs: []byte{}},
				// Enable the ZK dispute game.
				{Enabled: true, InitBond: big.NewInt(1000000000000000000), GameType: gameTypeZK, GameArgs: zkArgs},
			},
			// anchorGame is unset after a fresh migrate, so the re-seed is unconstrained; keep the
			// same anchor root for clarity.
			StartingAnchorRoot:        startingAnchorRoot,
			StartingRespectedGameType: gameTypeZK,
		},
	})
	require.NoError(t, err, "interop dispute game swap failed")
	require.NotEqual(t, common.Address{}, output.DisputeGameFactory, "shared DGF should be resolved")

	dump, err := bcast.Dump()
	require.NoError(t, err)
	require.Len(t, dump, 2, "should have two transactions (migrate + swap)")
	require.Equal(t, l1ProxyAdminOwner, *dump[1].To, "swap tx should be sent to the prank address")
}

func TestEncodeZKGameArgs(t *testing.T) {
	t.Run("succeeds", func(t *testing.T) {
		args, err := encodeZKGameArgs(zkDisputeGameConfig{
			AbsolutePrestate:     common.HexToHash("0x1234"),
			Verifier:             common.HexToAddress("0xBEEF"),
			MaxChallengeDuration: 7 * 24 * 60 * 60,
			MaxProveDuration:     3 * 24 * 60 * 60,
			ChallengerBond:       big.NewInt(1e18),
		})
		require.NoError(t, err)
		// 5 head words for (bytes32, address, uint64, uint64, uint256) = 160 bytes, selector stripped.
		require.Len(t, args, 160)
	})

	t.Run("rejects zero verifier", func(t *testing.T) {
		_, err := encodeZKGameArgs(zkDisputeGameConfig{
			AbsolutePrestate:     common.HexToHash("0x1234"),
			MaxChallengeDuration: 1,
			MaxProveDuration:     1,
			ChallengerBond:       big.NewInt(1),
		})
		require.ErrorContains(t, err, "verifier")
	})

	t.Run("rejects zero prestate", func(t *testing.T) {
		_, err := encodeZKGameArgs(zkDisputeGameConfig{
			Verifier:             common.HexToAddress("0xBEEF"),
			MaxChallengeDuration: 1,
			MaxProveDuration:     1,
			ChallengerBond:       big.NewInt(1),
		})
		require.ErrorContains(t, err, "absolutePrestate")
	})

	t.Run("rejects zero durations and bond", func(t *testing.T) {
		base := zkDisputeGameConfig{
			AbsolutePrestate:     common.HexToHash("0x1234"),
			Verifier:             common.HexToAddress("0xBEEF"),
			MaxChallengeDuration: 1,
			MaxProveDuration:     1,
			ChallengerBond:       big.NewInt(1),
		}

		c := base
		c.MaxChallengeDuration = 0
		_, err := encodeZKGameArgs(c)
		require.ErrorContains(t, err, "maxChallengeDuration")

		c = base
		c.MaxProveDuration = 0
		_, err = encodeZKGameArgs(c)
		require.ErrorContains(t, err, "maxProveDuration")

		c = base
		c.ChallengerBond = big.NewInt(0)
		_, err = encodeZKGameArgs(c)
		require.ErrorContains(t, err, "challengerBond")
	})
}
