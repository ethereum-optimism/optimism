package pipeline

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/scriptbackend"
)

// These helpers spawn the Rust op-script-engine for the pipeline tests, replacing the deleted Go
// script hosts (env.DefaultForkedScriptHost / env.ForkedScriptHost / env.DefaultScriptHost +
// opcm.NewScripts). They mirror apply.go's initForkedL1Engine / initGenesisL1Engine using only the
// exported scriptbackend / rustengine surface.

// newForkedL1EngineForTest spawns the engine in fork mode against l1RPCUrl. block pins the fork
// height; when nil the latest block is used. It returns the engine and its ArtifactsFS for
// pipeline.Env (L1Engine / L1Artifacts). The engine is closed via t.Cleanup.
func newForkedL1EngineForTest(t *testing.T, ctx context.Context, lgr log.Logger, artifactsFS foundry.StatDirFs, l1RPCUrl string, block *uint64) (*rustengine.Engine, *foundry.ArtifactsFS) {
	t.Helper()
	artifactsDir, err := rustengine.ArtifactsDir(artifactsFS)
	require.NoError(t, err)
	binPath, err := rustengine.EngineBinary(ctx, lgr)
	require.NoError(t, err)
	eng, err := rustengine.Spawn(binPath, scriptbackend.ForkedSpawnOpts(artifactsDir), rustengine.NewLogWriter(lgr))
	require.NoError(t, err)
	t.Cleanup(eng.Close)

	forkBlock := block
	if forkBlock == nil {
		rpcClient, err := rpc.Dial(l1RPCUrl)
		require.NoError(t, err)
		defer rpcClient.Close()
		latest, err := ethclient.NewClient(rpcClient).HeaderByNumber(ctx, nil)
		require.NoError(t, err)
		b := latest.Number.Uint64()
		forkBlock = &b
	}
	_, err = eng.CreateSelectFork(l1RPCUrl, forkBlock)
	require.NoError(t, err)
	return eng, &foundry.ArtifactsFS{FS: artifactsFS}
}

// newGenesisL1EngineForTest spawns a non-forked engine (NoMaxCodeSize, like the genesis L1 deploy)
// and builds the OPCM Scripts bundle on it — the replacement for env.DefaultScriptHost +
// opcm.NewScripts. The engine is closed via t.Cleanup.
func newGenesisL1EngineForTest(t *testing.T, ctx context.Context, lgr log.Logger, deployer common.Address, artifactsFS foundry.StatDirFs) (*rustengine.Engine, *foundry.ArtifactsFS, *opcm.Scripts) {
	t.Helper()
	artifactsDir, err := rustengine.ArtifactsDir(artifactsFS)
	require.NoError(t, err)
	binPath, err := rustengine.EngineBinary(ctx, lgr)
	require.NoError(t, err)
	spawnOpts := scriptbackend.ForkedSpawnOpts(artifactsDir)
	spawnOpts.NoMaxCodeSize = true
	eng, err := rustengine.Spawn(binPath, spawnOpts, rustengine.NewLogWriter(lgr))
	require.NoError(t, err)
	t.Cleanup(eng.Close)
	fa := &foundry.ArtifactsFS{FS: artifactsFS}
	scripts, err := scriptbackend.NewEngineScripts(eng, fa, func() common.Address { return deployer })
	require.NoError(t, err)
	return eng, fa, scripts
}
