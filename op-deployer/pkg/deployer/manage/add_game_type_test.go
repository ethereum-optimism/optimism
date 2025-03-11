package manage

import (
	"context"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/superchain-registry/validation"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"log/slog"
	"math/big"
	"os"
	"testing"
	"time"
)

func TestAddGameType(t *testing.T) {
	rpcURL := os.Getenv("SEPOLIA_RPC_URL")
	require.NotEmpty(t, rpcURL, "must specify RPC url via SEPOLIA_RPC_URL env var")

	afacts, _ := testutil.LocalArtifacts(t)
	v200SepoliaAddrs := validation.StandardVersionsSepolia["op-contracts/v2.0.0-rc.1"]
	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	cfg := AddGameTypeConfig{
		L1RPCUrl:         rpcURL,
		Logger:           testlog.Logger(t, slog.LevelInfo),
		ArtifactsLocator: afacts,
		Input: opcm.AddGameTypeInput{
			SaltMixer: "foo",
			// The values below were pulled from the Superchain Registry.
			SystemConfig:            common.HexToAddress("0x034edD2A225f7f429A63E0f1D2084B9E0A93b538"),
			ProxyAdmin:              common.HexToAddress("0x189aBAAaa82DfC015A588A7dbaD6F13b1D3485Bc"),
			DelayedWETH:             common.HexToAddress("0x9C7750C1c7b39E6b0eFeec06A1F2cf06190f6018"),
			DisputeGameType:         999,
			DisputeAbsolutePrestate: common.HexToHash("0x1234"),
			DisputeMaxGameDepth:     big.NewInt(73),
			DisputeSplitDepth:       big.NewInt(30),
			DisputeClockExtension:   10800,
			DisputeMaxClockDuration: 302400,
			InitialBond:             big.NewInt(0),
			VM:                      common.Address(*v200SepoliaAddrs.Mips.Address),
			Permissioned:            false,
			Prank:                   common.HexToAddress("0x1Eb2fFc903729a0F03966B917003800b145F56E2"),
			OPCM:                    common.Address(*v200SepoliaAddrs.OPContractsManager.Address),
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

	require.NotEqual(t, common.Address{}, output.DelayedWETH)
	require.NotEqual(t, common.Address{}, output.FaultDisputeGame)
}
