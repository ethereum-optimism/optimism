package opcm

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/gameargs"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestSetDisputeGameImpl(t *testing.T) {
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

	gameArgs := gameargs.GameArgs{}.PackPermissionless()
	// Backwards compatibility for DisputeGameFactory prior to adding game args
	// Can be removed once sepolia is upgraded.
	if contractVersion(t, host, factoryAddr) == "1.2.0" {
		gameArgs = nil
	}

	input := SetDisputeGameImplInput{
		Factory:             factoryAddr,
		UseV2:               len(gameArgs) > 0,
		Impl:                common.Address{'I'},
		GameType:            999,
		AnchorStateRegistry: common.Address{}, // Do not set as respected game type as we aren't authorized
		GameArgs:            gameArgs,
	}
	require.NoError(t, SetDisputeGameImpl(host, input))
}

func contractVersion(t *testing.T, host *script.Host, factoryAddr common.Address) string {
	versionSelector := crypto.Keccak256([]byte("version()"))[:4]
	data, _, err := host.Call(common.Address{}, factoryAddr, versionSelector, 1_000_000, uint256.NewInt(0))
	require.NoError(t, err)
	decoded, err := (abi.Arguments{{Type: abi.Type{T: abi.StringTy}}}).Unpack(data)
	require.NoError(t, err)
	version := decoded[0].(string)
	return version
}
