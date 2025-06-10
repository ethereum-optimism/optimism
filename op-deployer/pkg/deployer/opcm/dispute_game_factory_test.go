package opcm

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestSetDisputeGameImpl(t *testing.T) {
	t.Skip("Test requires SEPOLIA_RPC_URL and modifies real contracts - skipping in CI")
	// TODO: snevins - update for local test
	
	t.Parallel()

	_, artifacts := testutil.LocalArtifacts(t)

	l1RPCUrl := os.Getenv("SEPOLIA_RPC_URL")
	require.NotEmpty(t, l1RPCUrl, "SEPOLIA_RPC_URL must be set")

	l1RPC, err := rpc.Dial(l1RPCUrl)
	require.NoError(t, err)

	// OP Sepolia DGF owner
	deployer := common.HexToAddress("0x1Eb2fFc903729a0F03966B917003800b145F56E2")

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	host, err := env.DefaultForkedScriptHost(
		ctx,
		broadcaster.NoopBroadcaster(),
		testlog.Logger(t, log.LevelInfo),
		deployer,
		artifacts,
		l1RPC,
	)
	require.NoError(t, err)

	// Use OP Sepolia's dispute game factory
	factoryAddr := common.HexToAddress("0x05F9613aDB30026FFd634f38e5C4dFd30a197Fa1")

	input := SetDisputeGameImplInput{
		Factory:             factoryAddr,
		Impl:                common.Address{'I'},
		GameType:            999999,
		AnchorStateRegistry: common.HexToAddress("0x9876543210987654321098765432109876543210"),
		AbsolutePrestate:    [32]byte{0xab, 0xcd, 0xab, 0xcd},
		BigStepper:          common.HexToAddress("0x5555666677778888999900001111222233334444"),
	}
	require.NoError(t, SetDisputeGameImpl(host, input))
}
