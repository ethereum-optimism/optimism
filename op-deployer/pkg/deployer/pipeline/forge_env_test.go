package pipeline

import (
	"context"
	"log/slog"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// newTestForge returns a forge env that runs the local scripts against l1RPCURL.
func newTestForge(t *testing.T, ctx context.Context, l1RPCURL string) *opcm.ForgeEnv {
	t.Helper()
	_, afs := testutil.LocalArtifacts(t)
	client, err := opcm.NewForgeClient(afs, "", opcm.L1ForgeScripts)
	require.NoError(t, err)
	return &opcm.ForgeEnv{Client: client, Context: ctx, L1RPCUrl: l1RPCURL}
}

// newTestGenesisEnv returns an env that runs the local scripts offline on an empty genesis state.
func newTestGenesisEnv(t *testing.T, deployer common.Address) *Env {
	t.Helper()
	_, afs := testutil.LocalArtifacts(t)
	client, err := opcm.NewForgeClient(afs, "", opcm.L1ForgeScripts)
	require.NoError(t, err)
	ctx := context.Background()
	return &Env{
		Logger:   testlog.Logger(t, slog.LevelDebug),
		Deployer: deployer,
		Forge: &opcm.ForgeEnv{
			Client:  client,
			Context: ctx,
			Genesis: opcm.NewGenesisState(deployer, true),
		},
		L1Artifacts:               &foundry.ArtifactsFS{FS: afs},
		IsGenesis:                 true,
		AllowUnoptimizedContracts: true,
		Context:                   ctx,
	}
}

// forkL1 starts an anvil fork of l1RPCURL at blockNumber and returns its RPC URL.
func forkL1(t *testing.T, ctx context.Context, l1RPCURL string, blockNumber uint64) string {
	t.Helper()
	anvil, err := forge.NewAnvilBinary()
	require.NoError(t, err)
	fork, err := forge.StartFork(ctx, anvil, l1RPCURL, blockNumber)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, fork.Stop()) })
	return fork.RPCURL()
}
