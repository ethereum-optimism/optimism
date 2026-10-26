package opcm

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/gameargs"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	opbindings "github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
)

func TestSetDisputeGameImpl(t *testing.T) {
	t.Parallel()

	l1RPCUrl := os.Getenv("SEPOLIA_RPC_URL")
	require.NotEmpty(t, l1RPCUrl, "SEPOLIA_RPC_URL must be set")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	l1Client, err := ethclient.DialContext(ctx, l1RPCUrl)
	require.NoError(t, err)
	head, err := l1Client.HeaderByNumber(ctx, nil)
	require.NoError(t, err)

	anvil, err := forge.NewAnvilBinary()
	require.NoError(t, err)
	fork, err := forge.StartFork(ctx, anvil, l1RPCUrl, bigs.Uint64Strict(head.Number))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, fork.Stop()) })

	_, afs := testutil.LocalArtifacts(t)
	client, err := NewForgeClient(afs, "", L1ForgeScripts)
	require.NoError(t, err)
	env := &ForgeEnv{
		Client:   client,
		Context:  ctx,
		L1RPCUrl: fork.RPCURL(),
		// OP Sepolia DGF owner
		UnlockedSender: common.HexToAddress("0x1Eb2fFc903729a0F03966B917003800b145F56E2"),
	}

	// Use OP Sepolia's dispute game factory
	factoryAddr := common.HexToAddress("0x05F9613aDB30026FFd634f38e5C4dFd30a197Fa1")

	gameArgs := gameargs.GameArgs{}.PackPermissionless()
	// Backwards compatibility for DisputeGameFactory prior to adding game args
	// Can be removed once sepolia is upgraded.
	factory, err := opbindings.NewDisputeGameFactory(factoryAddr, l1Client)
	require.NoError(t, err)
	version, err := factory.Version(&bind.CallOpts{Context: ctx})
	require.NoError(t, err)
	if version == "1.2.0" {
		gameArgs = nil
	}

	input := SetDisputeGameImplInput{
		Factory:             factoryAddr,
		Impl:                common.Address{'I'},
		GameType:            999,
		AnchorStateRegistry: common.Address{}, // Do not set as respected game type as we aren't authorized
		GameArgs:            gameArgs,
	}
	require.NoError(t, SetDisputeGameImpl(env, input))
}
