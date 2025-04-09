package interop

import (
	"context"
	"log/slog"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestInteropMigration(t *testing.T) {
	t.Skip("Skipped until the sepolia opcm supports the interop migration")

	lgr := testlog.Logger(t, slog.LevelDebug)

	forkedL1, stopL1, err := devnet.NewForkedSepolia(lgr)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, stopL1())
	})
	l1RPC := forkedL1.RPCUrl()

	_, afactsFS := testutil.LocalArtifacts(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	rpcClient, err := rpc.Dial(l1RPC)
	require.NoError(t, err)

	bcast := new(broadcaster.CalldataBroadcaster)
	host, err := env.DefaultForkedScriptHost(
		ctx,
		bcast,
		lgr,
		common.Address{'D'},
		afactsFS,
		rpcClient,
	)
	require.NoError(t, err)

	pao := common.HexToAddress("0x1Eb2fFc903729a0F03966B917003800b145F56E2")
	input := InteropMigrationInput{
		Prank:                          pao,
		Opcm:                           common.HexToAddress("0xaf334f4537e87f5155d135392ff6d52f1866465e"),
		UsePermissionlessGame:          true,
		StartingAnchorL2SequenceNumber: big.NewInt(1),
		Proposer:                       common.Address{'A'},
		Challenger:                     common.Address{'B'},
		MaxGameDepth:                   10,
		SplitDepth:                     10,
		InitBond:                       big.NewInt(1000),
		ClockExtension:                 10,
		MaxClockDuration:               10,
		EncodedChainConfigs: []OPChainConfig{
			{
				SystemConfigProxy: common.HexToAddress("0x034edD2A225f7f429A63E0f1D2084B9E0A93b538"),
				ProxyAdmin:        common.HexToAddress("0x189aBAAaa82DfC015A588A7dbaD6F13b1D3485Bc"),
				AbsolutePrestate:  common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000abc"),
			},
		},
	}
	output, err := Migrate(host, input)
	require.NoError(t, err)
	require.NotEqual(t, common.Address{}, output.DisputeGameFactory)

	dump, err := bcast.Dump()
	require.NoError(t, err)
	require.True(t, dump[0].Value.ToInt().Cmp(common.Big0) == 0)
	require.Equal(t, *dump[0].To, pao)
}
