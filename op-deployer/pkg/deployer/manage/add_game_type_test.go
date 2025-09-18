package manage

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum/go-ethereum/superchain"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/superchain-registry/validation"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

// getAddressesOnchain reads addresses from on-chain contracts (using chainConfig to get entrypoints)
func getAddressesOnchain(ctx context.Context, rpcURL string, chainConfig *superchain.ChainConfig) (opChainProxyAdmin, delayedWETHProxy common.Address, err error) {
	var proxyAdminFn = w3.MustNewFunc("proxyAdmin()", "address")
	var gameImplsFn = w3.MustNewFunc("gameImpls(uint32)", "address")
	var wethFn = w3.MustNewFunc("weth()", "address")

	client, err := w3.Dial(rpcURL)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("failed to connect to RPC: %w", err)
	}
	defer client.Close()

	systemConfigProxy := *chainConfig.Addresses.SystemConfigProxy
	disputeGameFactoryProxy := *chainConfig.Addresses.DisputeGameFactoryProxy

	// Read OPChainProxyAdmin from systemConfigProxy.proxyAdmin()
	err = client.CallCtx(ctx, eth.CallFunc(systemConfigProxy, proxyAdminFn).Returns(&opChainProxyAdmin))
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("failed to read proxyAdmin from SystemConfig: %w", err)
	}

	// Read permissionless dispute game address from disputeGameFactoryProxy.gameImpls(0)
	// GameTypes.CANNON = 0 (permissionless)
	var permissionlessDisputeGame common.Address
	err = client.CallCtx(ctx, eth.CallFunc(disputeGameFactoryProxy, gameImplsFn, uint32(0)).Returns(&permissionlessDisputeGame))
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("failed to read gameImpls(0) from DisputeGameFactory: %w", err)
	}

	// Read DelayedWETHProxy from permissionlessDisputeGame.weth()
	err = client.CallCtx(ctx, eth.CallFunc(permissionlessDisputeGame, wethFn).Returns(&delayedWETHProxy))
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("failed to read weth from permissionless dispute game: %w", err)
	}

	return opChainProxyAdmin, delayedWETHProxy, nil
}

func TestAddGameType(t *testing.T) {
	rpcURL := os.Getenv("SEPOLIA_RPC_URL")
	require.NotEmpty(t, rpcURL, "must specify RPC url via SEPOLIA_RPC_URL env var")

	afacts, _ := testutil.LocalArtifacts(t)
	v200SepoliaAddrs := validation.StandardVersionsSepolia[standard.ContractsV200Tag]
	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	chain, err := superchain.GetChain(11155420)
	require.NoError(t, err)
	chainConfig, err := chain.Config()
	require.NoError(t, err)

	readCtx, readCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer readCancel()

	opChainProxyAdmin, delayedWETHProxy, err := getAddressesOnchain(readCtx, rpcURL, chainConfig)
	require.NoError(t, err, "failed to read addresses from chain")

	cfg := AddGameTypeConfig{
		L1RPCUrl:         rpcURL,
		Logger:           testlog.Logger(t, slog.LevelInfo),
		ArtifactsLocator: afacts,
		SaltMixer:        "foo",
		// The values below were pulled from the Superchain Registry for OP Sepolia.
		SystemConfigProxy:       *chainConfig.Addresses.SystemConfigProxy,
		OPChainProxyAdmin:       opChainProxyAdmin,
		DelayedWETHProxy:        delayedWETHProxy,
		DisputeGameType:         999,
		DisputeAbsolutePrestate: common.HexToHash("0x1234"),
		DisputeMaxGameDepth:     big.NewInt(73),
		DisputeSplitDepth:       big.NewInt(30),
		DisputeClockExtension:   10800,
		DisputeMaxClockDuration: 302400,
		InitialBond:             big.NewInt(1),
		VM:                      common.Address(*v200SepoliaAddrs.Mips.Address),
		Permissionless:          false,
		L1ProxyAdminOwner:       *chainConfig.Roles.ProxyAdminOwner,
		OPCMImpl:                common.Address(*v200SepoliaAddrs.OPContractsManager.Address),
		CacheDir:                testCacheDir,
	}

	addCtx, addCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer addCancel()

	output, broadcasts, err := AddGameType(addCtx, cfg)
	require.NoError(t, err)

	require.Equal(t, 1, len(broadcasts))
	// Selector for addGameType
	require.EqualValues(t, []byte{0x16, 0x61, 0xa2, 0xe9}, broadcasts[0].Data[0:4])

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
		{OPChainProxyAdminFlag, common.Address{0x04}.String()},
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
		require.Equal(t, common.HexToAddress("0x7bd8879acf1e74547455c7ddc07f5c3f4a3c133d"), cfg.OPChainProxyAdmin)
		require.Equal(t, common.HexToAddress("0x02f909cf91c2134e70a67950b7f27db7c8ee55d6"), cfg.SystemConfigProxy)
		require.Equal(t, common.HexToAddress("0x0000000000000000000000000000000000000001"), cfg.VM)
	})

	t.Run("successful population from CLI", func(t *testing.T) {
		app := cli.NewApp()
		flagSet := flag.NewFlagSet("test-success", flag.ContinueOnError)

		flagSet.String(L1ProxyAdminOwnerFlag.Name, "0x1eb2ffc903729a0f03966b917003800b145f56e2", "doc")
		flagSet.String(OPCMImplFlag.Name, "0xfbceed4de885645fbded164910e10f52febfab35", "doc")
		flagSet.String(OPChainProxyAdminFlag.Name, "0x7bd8879acf1e74547455c7ddc07f5c3f4a3c133d", "doc")
		flagSet.String(SystemConfigProxyFlag.Name, "0x02f909cf91c2134e70a67950b7f27db7c8ee55d6", "doc")
		flagSet.String(VMFlag.Name, "0x0000000000000000000000000000000000000001", "doc")

		ctx := cli.NewContext(app, flagSet, nil)
		cfg := &AddGameTypeConfig{}
		err := populateConfigFromFlags(cfg, ctx)
		require.NoError(t, err)

		require.Equal(t, common.HexToAddress("0x1eb2ffc903729a0f03966b917003800b145f56e2"), cfg.L1ProxyAdminOwner)
		require.Equal(t, common.HexToAddress("0xfbceed4de885645fbded164910e10f52febfab35"), cfg.OPCMImpl)
		require.Equal(t, common.HexToAddress("0x7bd8879acf1e74547455c7ddc07f5c3f4a3c133d"), cfg.OPChainProxyAdmin)
		require.Equal(t, common.HexToAddress("0x02f909cf91c2134e70a67950b7f27db7c8ee55d6"), cfg.SystemConfigProxy)
		require.Equal(t, common.HexToAddress("0x0000000000000000000000000000000000000001"), cfg.VM)
	})
}
