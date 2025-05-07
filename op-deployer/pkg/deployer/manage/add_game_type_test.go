package manage

import (
	"context"
	"log/slog"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum/go-ethereum/superchain"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/superchain-registry/validation"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestAddGameType(t *testing.T) {
	rpcURL := os.Getenv("SEPOLIA_RPC_URL")
	require.NotEmpty(t, rpcURL, "must specify RPC url via SEPOLIA_RPC_URL env var")

	afacts, _ := testutil.LocalArtifacts(t)
	v200SepoliaAddrs := validation.StandardVersionsSepolia[standard.ContractsV200Tag]
	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	supChain, err := superchain.GetChain(11155420)
	require.NoError(t, err)
	supChainConfig, err := supChain.Config()
	require.NoError(t, err)

	cfg := AddGameTypeConfig{
		L1RPCUrl:         rpcURL,
		Logger:           testlog.Logger(t, slog.LevelInfo),
		ArtifactsLocator: afacts,
		Input: opcm.AddGameTypeInput{
			SaltMixer: "foo",
			// The values below were pulled from the Superchain Registry for OP Sepolia.
			SystemConfigProxy:       *supChainConfig.Addresses.SystemConfigProxy,
			OPChainProxyAdmin:       *supChainConfig.Addresses.ProxyAdmin,
			DelayedWETHProxy:        *supChainConfig.Addresses.DelayedWETHProxy,
			DisputeGameType:         999,
			DisputeAbsolutePrestate: common.HexToHash("0x1234"),
			DisputeMaxGameDepth:     big.NewInt(73),
			DisputeSplitDepth:       big.NewInt(30),
			DisputeClockExtension:   10800,
			DisputeMaxClockDuration: 302400,
			InitialBond:             big.NewInt(0),
			VM:                      common.Address(*v200SepoliaAddrs.Mips.Address),
			Permissioned:            false,
			Prank:                   *supChainConfig.Roles.ProxyAdminOwner,
			OPCMImpl:                common.Address(*v200SepoliaAddrs.OPContractsManager.Address),
		},
		CacheDir: testCacheDir,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	output, broadcasts, err := AddGameType(ctx, cfg)
	require.NoError(t, err)

	require.Equal(t, 1, len(broadcasts))
	// Selector for addGameType
	require.EqualValues(t, []byte{0x16, 0x61, 0xa2, 0xe9}, broadcasts[0].Data[0:4])

	require.NotEqual(t, common.Address{}, output.DelayedWETHProxy)
	require.NotEqual(t, common.Address{}, output.FaultDisputeGameProxy)
}
