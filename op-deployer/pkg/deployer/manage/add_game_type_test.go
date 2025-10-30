package manage

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/bootstrap"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/superchain"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestAddGameType(t *testing.T) {
	// Since the opcm version is not yet on sepolia, we create a fork of sepolia then deploy the opcm via deploy implementations.
	lgr := testlog.Logger(t, slog.LevelDebug)
	forkedL1, stopL1, err := devnet.NewForkedSepolia(lgr)
	pkHex, _, _ := shared.DefaultPrivkey(t)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, stopL1())
	})

	afacts, _ := testutil.LocalArtifacts(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	sChain, err := standard.SuperchainFor(11155111)
	require.NoError(t, err)

	superchainProxyAdmin, err := standard.SuperchainProxyAdminAddrFor(11155111)
	require.NoError(t, err)

	superchainProxyAdminOwner, err := standard.L1ProxyAdminOwner(11155111)
	require.NoError(t, err)

	impls, err := bootstrap.Implementations(ctx, bootstrap.ImplementationsConfig{
		L1RPCUrl:                        forkedL1.RPCUrl(),
		PrivateKey:                      pkHex,
		ArtifactsLocator:                afacts,
		MIPSVersion:                     int(standard.MIPSVersion),
		WithdrawalDelaySeconds:          standard.WithdrawalDelaySeconds,
		MinProposalSizeBytes:            standard.MinProposalSizeBytes,
		ChallengePeriodSeconds:          standard.ChallengePeriodSeconds,
		ProofMaturityDelaySeconds:       standard.ProofMaturityDelaySeconds,
		DisputeGameFinalityDelaySeconds: standard.DisputeGameFinalityDelaySeconds,
		DevFeatureBitmap:                common.Hash{},
		SuperchainConfigProxy:           sChain.SuperchainConfigAddr,
		ProtocolVersionsProxy:           sChain.ProtocolVersionsAddr,
		L1ProxyAdminOwner:               superchainProxyAdminOwner,
		SuperchainProxyAdmin:            superchainProxyAdmin,
		CacheDir:                        testCacheDir,
		Logger:                          lgr,
		Challenger:                      common.Address{'C'},
	})
	require.NoError(t, err)

	chain, err := superchain.GetChain(11155420)
	require.NoError(t, err)
	chainConfig, err := chain.Config()
	require.NoError(t, err)

	cfg := AddGameTypeConfig{
		L1RPCUrl:         forkedL1.RPCUrl(),
		Logger:           testlog.Logger(t, slog.LevelInfo),
		ArtifactsLocator: afacts,
		SaltMixer:        "foo",
		// The values below were pulled from the Superchain Registry for OP Sepolia.
		SystemConfigProxy:       *chainConfig.Addresses.SystemConfigProxy,
		DelayedWETHProxy:        common.Address{}, // Let the OPCM create a new one.
		DisputeGameType:         0,
		DisputeAbsolutePrestate: common.HexToHash("0x1234"),
		DisputeMaxGameDepth:     big.NewInt(73),
		DisputeSplitDepth:       big.NewInt(30),
		DisputeClockExtension:   10800,
		DisputeMaxClockDuration: 302400,
		InitialBond:             big.NewInt(1),
		VM:                      impls.MipsSingleton,
		Permissionless:          true,
		L1ProxyAdminOwner:       superchainProxyAdminOwner,
		OPCMImpl:                impls.Opcm,
		CacheDir:                testCacheDir,
	}

	addCtx, addCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer addCancel()

	output, broadcasts, err := AddGameType(addCtx, cfg)
	require.NoError(t, err)

	require.Equal(t, 1, len(broadcasts))
	// Selector for addGameType
	// Gotten from `cast sig "addGameType((string,address,address,uint32,bytes32,uint256,uint256,uint64,uint64,uint256,address,bool)[])"`
	require.EqualValues(t, []byte{0x60, 0x4a, 0xa6, 0x28}, broadcasts[0].Data[0:4])

	require.NotEqual(t, common.Address{}, output.DelayedWETHProxy)
	require.NotEqual(t, common.Address{}, output.FaultDisputeGameProxy)
}

func TestAddGameType_CLI(t *testing.T) {
	incompatibleFlags := []struct {
		flag  *cli.StringFlag
		value string
	}{
		{L1ProxyAdminOwnerFlag, common.Address{0x01}.String()},
		{OPCMImplFlag, common.Address{0x02}.String()},
		{SystemConfigProxyFlag, common.Address{0x03}.String()},
		{VMFlag, common.Address{0x05}.String()},
	}

	for _, tt := range incompatibleFlags {
		t.Run(fmt.Sprintf("incompatible flag %s", tt.flag.Name), func(t *testing.T) {
			flagSet := flag.NewFlagSet(fmt.Sprintf("test-%s", tt.flag.Name), flag.ContinueOnError)
			flagSet.String(WorkdirFlag.Name, "/tmp/testworkdir", "")
			flagSet.String(L2ChainIDFlag.Name, "12345", "")

			flagSet.String(tt.flag.Name, tt.value, "doc")

			ctx := cli.NewContext(cli.NewApp(), flagSet, nil)

			err := populateConfigFromWorkdir(new(AddGameTypeConfig), ctx)
			require.Error(t, err)
			expectedError := fmt.Sprintf("cannot specify --%s when --workdir is set", tt.flag.Name)
			require.ErrorContains(t, err, expectedError)
		})
	}

	t.Run("missing chain id", func(t *testing.T) {
		app := cli.NewApp()
		flagSet := flag.NewFlagSet("test-missing-chainid", flag.ContinueOnError)

		// Set WorkdirFlag
		flagSet.String(WorkdirFlag.Name, "/tmp/testworkdir", "doc")

		ctx := cli.NewContext(app, flagSet, nil)

		err := populateConfigFromWorkdir(new(AddGameTypeConfig), ctx)
		require.Error(t, err)
		require.ErrorContains(t, err, "flag --l2-chain-id must be specified when --workdir is set")
	})

	t.Run("successful population from workdir", func(t *testing.T) {
		app := cli.NewApp()
		flagSet := flag.NewFlagSet("test-success", flag.ContinueOnError)
		flagSet.String(WorkdirFlag.Name, "./testdata", "doc")
		flagSet.String(L2ChainIDFlag.Name, "1234", "doc")

		ctx := cli.NewContext(app, flagSet, nil)
		cfg := &AddGameTypeConfig{}
		err := populateConfigFromWorkdir(cfg, ctx)
		require.NoError(t, err)

		require.Equal(t, common.HexToAddress("0x1eb2ffc903729a0f03966b917003800b145f56e2"), cfg.L1ProxyAdminOwner)
		require.Equal(t, common.HexToAddress("0xfbceed4de885645fbded164910e10f52febfab35"), cfg.OPCMImpl)
		require.Equal(t, common.HexToAddress("0x02f909cf91c2134e70a67950b7f27db7c8ee55d6"), cfg.SystemConfigProxy)
		require.Equal(t, common.HexToAddress("0x0000000000000000000000000000000000000001"), cfg.VM)
	})

	t.Run("successful population from CLI", func(t *testing.T) {
		app := cli.NewApp()
		flagSet := flag.NewFlagSet("test-success", flag.ContinueOnError)

		flagSet.String(L1ProxyAdminOwnerFlag.Name, "0x1eb2ffc903729a0f03966b917003800b145f56e2", "doc")
		flagSet.String(OPCMImplFlag.Name, "0xfbceed4de885645fbded164910e10f52febfab35", "doc")
		flagSet.String(SystemConfigProxyFlag.Name, "0x02f909cf91c2134e70a67950b7f27db7c8ee55d6", "doc")
		flagSet.String(VMFlag.Name, "0x0000000000000000000000000000000000000001", "doc")

		ctx := cli.NewContext(app, flagSet, nil)
		cfg := &AddGameTypeConfig{}
		err := populateConfigFromFlags(cfg, ctx)
		require.NoError(t, err)

		require.Equal(t, common.HexToAddress("0x1eb2ffc903729a0f03966b917003800b145f56e2"), cfg.L1ProxyAdminOwner)
		require.Equal(t, common.HexToAddress("0xfbceed4de885645fbded164910e10f52febfab35"), cfg.OPCMImpl)
		require.Equal(t, common.HexToAddress("0x02f909cf91c2134e70a67950b7f27db7c8ee55d6"), cfg.SystemConfigProxy)
		require.Equal(t, common.HexToAddress("0x0000000000000000000000000000000000000001"), cfg.VM)
	})
}
