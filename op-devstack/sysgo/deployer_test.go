package sysgo

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params/forks"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
	w3eth "github.com/lmittmann/w3/module/eth"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestLocalContractSourcesLocator(t *testing.T) {
	t.Parallel()

	t.Run("valid dir", func(t *testing.T) {
		t.Parallel()

		artifactsDir := filepath.Join(t.TempDir(), "forge-artifacts")
		require.NoError(t, os.Mkdir(artifactsDir, 0o755))

		loc, err := localContractSourcesLocator(artifactsDir)
		require.NoError(t, err)

		require.True(t, artifacts.MustNewFileLocator(artifactsDir).Equal(loc))
	})

	t.Run("missing dir", func(t *testing.T) {
		t.Parallel()

		_, err := localContractSourcesLocator(filepath.Join(t.TempDir(), "missing"))
		require.Error(t, err)
	})
}

func TestLocalContractSourcesLocatorAcceptsContractsBedrockRoot(t *testing.T) {
	t.Parallel()

	contractsDir := filepath.Join(t.TempDir(), "packages", "contracts-bedrock")
	require.NoError(t, os.MkdirAll(filepath.Join(contractsDir, "forge-artifacts"), 0o755))

	loc, err := localContractSourcesLocator(contractsDir)
	require.NoError(t, err)

	require.True(t, artifacts.MustNewFileLocator(contractsDir).Equal(loc))
}

func TestWithLocalContractSourcesAt(t *testing.T) {
	t.Parallel()

	artifactsDir := filepath.Join(t.TempDir(), "forge-artifacts")
	require.NoError(t, os.Mkdir(artifactsDir, 0o755))

	builder := newValidIntentBuilder()
	dt := devtest.SerialT(t)

	WithLocalContractSourcesAt(artifactsDir)(dt, nil, builder)

	intent, err := builder.Build()
	require.NoError(t, err)

	expected := artifacts.MustNewFileLocator(artifactsDir)
	require.True(t, expected.Equal(intent.L1ContractsLocator))
	require.True(t, expected.Equal(intent.L2ContractsLocator))
}

func TestParseL1Fork(t *testing.T) {
	tests := map[string]forks.Fork{
		"pectra":      forks.Prague,
		"prague":      forks.Prague,
		"fusaka":      forks.Osaka,
		"osaka":       forks.Osaka,
		"glamsterdam": forks.Amsterdam,
		"amsterdam":   forks.Amsterdam,
	}
	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			actual, err := parseL1Fork(input)
			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})
	}

	for _, input := range []string{"", "dencun", "cancun", "frontier", "CANCUN", "Fusaka", "Osaka", " osaka ", "bpo1", "bpo2", "bpo3", "bpo4", "bpo5", "bpo-1", "BPO_5", " amsterdam "} {
		t.Run("reject "+input, func(t *testing.T) {
			_, err := parseL1Fork(input)
			require.EqualError(t, err, `unsupported L1 fork "`+input+`"`)
		})
	}
}

func TestDevstackL1ForkEnv(t *testing.T) {
	tests := map[string]bool{
		"pectra": false,
		"prague": false,
		"fusaka": true,
		"osaka":  true,
	}
	for input, osakaAtGenesis := range tests {
		t.Run(input, func(t *testing.T) {
			t.Setenv(DevstackL1ForkEnvVar, input)
			builder := newValidIntentBuilder()
			builder.WithL1ContractsLocator(artifacts.EmbeddedLocator)
			builder.WithL2ContractsLocator(artifacts.EmbeddedLocator)
			keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
			require.NoError(t, err)

			applyConfigCommons(devtest.SerialT(t), keys, DefaultL1ID, builder)

			intent, err := builder.Build()
			require.NoError(t, err)
			require.NotNil(t, intent.L1DevGenesisParams.PragueTimeOffset)
			require.Zero(t, *intent.L1DevGenesisParams.PragueTimeOffset)
			if osakaAtGenesis {
				require.NotNil(t, intent.L1DevGenesisParams.OsakaTimeOffset)
				require.Zero(t, *intent.L1DevGenesisParams.OsakaTimeOffset)
			} else {
				require.Nil(t, intent.L1DevGenesisParams.OsakaTimeOffset)
			}
			require.Nil(t, intent.L1DevGenesisParams.BPO1TimeOffset)
			require.Nil(t, intent.L1DevGenesisParams.BPO2TimeOffset)
		})
	}
}

func TestDeployerOptionsOverrideDevstackL1Fork(t *testing.T) {
	t.Setenv(DevstackL1ForkEnvVar, "fusaka")
	builder := newValidIntentBuilder()
	builder.WithL1ContractsLocator(artifacts.EmbeddedLocator)
	builder.WithL2ContractsLocator(artifacts.EmbeddedLocator)
	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	dt := devtest.SerialT(t)

	applyConfigCommons(dt, keys, DefaultL1ID, builder)
	applyConfigDeployerOptions(dt, keys, builder, []DeployerOption{
		WithForkAtL1Genesis(forks.Prague),
	})

	intent, err := builder.Build()
	require.NoError(t, err)
	require.NotNil(t, intent.L1DevGenesisParams.PragueTimeOffset)
	require.Zero(t, *intent.L1DevGenesisParams.PragueTimeOffset)
	require.Nil(t, intent.L1DevGenesisParams.OsakaTimeOffset)
	require.Nil(t, intent.L1DevGenesisParams.BPO1TimeOffset)
}

func TestDevstackFutureL1ForkAddsBlobSchedule(t *testing.T) {
	t.Setenv(DevstackL1ForkEnvVar, "glamsterdam")
	builder := newValidIntentBuilder()
	builder.WithL1ContractsLocator(artifacts.EmbeddedLocator)
	builder.WithL2ContractsLocator(artifacts.EmbeddedLocator)
	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)

	applyConfigCommons(devtest.SerialT(t), keys, DefaultL1ID, builder)

	intent, err := builder.Build()
	require.NoError(t, err)
	require.NotNil(t, intent.L1DevGenesisParams.AmsterdamTimeOffset)
	require.Zero(t, *intent.L1DevGenesisParams.AmsterdamTimeOffset)
	schedule := intent.L1DevGenesisParams.BlobSchedule
	require.NotNil(t, schedule)
	require.NotNil(t, schedule.BPO1)
	require.NotNil(t, schedule.BPO2)
	require.NotNil(t, schedule.BPO3)
	require.NotNil(t, schedule.BPO4)
	require.NotNil(t, schedule.BPO5)
	require.NotNil(t, schedule.Amsterdam)
}

func TestProofSetupPreservesGenesisAnchor(t *testing.T) {
	for _, gameType := range []gameTypes.GameType{gameTypes.SuperCannonKonaGameType, gameTypes.ZKDisputeGameType, gameTypes.CannonKonaGameType} {
		t.Run(gameType.String(), func(t *testing.T) {
			dt := devtest.SerialT(t)
			keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
			require.NoError(t, err)
			cfg := PresetConfig{
				AddedGameTypes:  []gameTypes.GameType{gameType},
				DeployerOptions: []DeployerOption{WithJovianAtGenesis},
			}
			if gameType == gameTypes.ZKDisputeGameType {
				cfg.DeployerOptions = append(cfg.DeployerOptions, WithDevFeatureEnabled(devfeatures.ZKDisputeGameFlag))
			}
			world := newDefaultSingleChainWorld(dt, keys, cfg)
			jwtPath, _ := writeJWTSecret(dt)
			l1EL, _ := startInProcessL1(dt, world.L1Network, jwtPath)
			rpcClient, err := rpc.DialContext(t.Context(), l1EL.UserRPC())
			require.NoError(t, err)
			defer rpcClient.Close()
			client := w3.NewClient(rpcClient)
			portal := world.L2Network.rollupCfg.DepositContractAddress
			var registry, factory common.Address
			require.NoError(t, client.Call(
				w3eth.CallFunc(portal, w3.MustNewFunc("anchorStateRegistry()", "address")).Returns(&registry),
			))
			require.NoError(t, client.Call(
				w3eth.CallFunc(registry, w3.MustNewFunc("disputeGameFactory()", "address")).Returns(&factory),
			))
			header := world.L2Network.genesis.ToBlock().Header()
			require.NotNil(t, header.WithdrawalsHash)
			output, err := rollup.ComputeL2OutputRootV0(eth.HeaderBlockInfo(header), *header.WithdrawalsHash)
			require.NoError(t, err)
			expected := common.Hash(eth.SuperRoot(eth.NewSuperV1(header.Time,
				eth.ChainIDAndOutput{ChainID: world.L2Network.ChainID(), Output: output})))
			expectedSequence := header.Time
			if gameType == gameTypes.CannonKonaGameType {
				expected = common.Hash(output)
				expectedSequence = 0
			}
			assertAnchor := func() {
				var root common.Hash
				var sequence *big.Int
				require.NoError(t, client.Call(
					w3eth.CallFunc(registry, w3.MustNewFunc("getAnchorRoot()", "bytes32,uint256")).Returns(&root, &sequence),
				))
				require.Equal(t, expected, root)
				require.Equal(t, expectedSequence, bigs.Uint64Strict(sequence))
			}
			assertAnchor()
			addGameTypesForRuntime(dt, keys, cfg.AddedGameTypes, world.L1Network.ChainID(), l1EL.UserRPC(), world.L2Network)
			var registryAfter, factoryAfter common.Address
			require.NoError(t, client.Call(
				w3eth.CallFunc(portal, w3.MustNewFunc("anchorStateRegistry()", "address")).Returns(&registryAfter),
			))
			require.NoError(t, client.Call(
				w3eth.CallFunc(registryAfter, w3.MustNewFunc("disputeGameFactory()", "address")).Returns(&factoryAfter),
			))
			require.Equal(t, registry, registryAfter)
			require.Equal(t, factory, factoryAfter)
			require.Equal(t, factory, world.L2Network.deployment.DisputeGameFactoryProxyAddr())
			require.NotEqual(t, common.Address{}, getGameImpl(dt, client, factory, uint32(gameType)))
			assertAnchor()
		})
	}
}

func newValidIntentBuilder() intentbuilder.Builder {
	builder := intentbuilder.New()

	_, superchain := builder.WithSuperchain()
	superchain.WithProxyAdminOwner(common.HexToAddress("0x1"))
	superchain.WithGuardian(common.HexToAddress("0x2"))
	superchain.WithChallenger(common.HexToAddress("0x4"))

	builder, _ = builder.WithL1(eth.ChainIDFromUInt64(1))
	_, l2 := builder.WithL2(eth.ChainIDFromUInt64(10))
	l2.WithBaseFeeVaultRecipient(common.HexToAddress("0x5"))
	l2.WithSequencerFeeVaultRecipient(common.HexToAddress("0x6"))
	l2.WithL1FeeVaultRecipient(common.HexToAddress("0x7"))
	l2.WithOperatorFeeVaultRecipient(common.HexToAddress("0x8"))
	l2.WithL1ProxyAdminOwner(common.HexToAddress("0x9"))
	l2.WithL2ProxyAdminOwner(common.HexToAddress("0xa"))
	l2.WithSystemConfigOwner(common.HexToAddress("0xb"))
	l2.WithUnsafeBlockSigner(common.HexToAddress("0xc"))
	l2.WithBatcher(common.HexToAddress("0xd"))
	l2.WithProposer(common.HexToAddress("0xe"))
	l2.WithChallenger(common.HexToAddress("0xf"))

	return builder
}
