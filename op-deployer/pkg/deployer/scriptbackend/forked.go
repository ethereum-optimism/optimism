package scriptbackend

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
)

// ForkedL1 is a forked L1 script backend plus its lifecycle: broadcast draining (the engine captures
// broadcasts that must be handed to the Go broadcaster before Broadcast; the Go host delivers them
// synchronously via its hook) and teardown. Build one with NewForkedL1, defer Close, run OPCM scripts
// through Backend, then call DrainBroadcasts before broadcaster.Broadcast.
type ForkedL1 struct {
	Backend Backend

	bcaster broadcaster.Broadcaster
	engine  *rustengine.Engine // non-nil on the rust path
	rpc     *rpc.Client        // non-nil on the go path (held open for lazy fork reads)
}

// NewForkedL1 builds a forked L1 backend bound to the latest block of l1RPCUrl. engineKind selects the
// implementation: rust (the default) spawns the out-of-process op-script-engine and installs an
// RPC-backed fork; go builds the in-process fork-backed script.Host. deployer is the tx.origin the
// OPCM scripts run as; bcaster receives the broadcasts (directly for go, via DrainBroadcasts for rust).
func NewForkedL1(
	ctx context.Context,
	engineKind env.ScriptEngineKind,
	lgr log.Logger,
	deployer common.Address,
	artifactsFS foundry.StatDirFs,
	l1RPCUrl string,
	bcaster broadcaster.Broadcaster,
) (*ForkedL1, error) {
	resolved, err := engineKind.Resolve()
	if err != nil {
		return nil, err
	}
	if resolved == env.ScriptEngineRust {
		return newForkedL1Engine(ctx, lgr, deployer, artifactsFS, l1RPCUrl, bcaster)
	}
	return newForkedL1Host(ctx, lgr, deployer, artifactsFS, l1RPCUrl, bcaster)
}

func newForkedL1Host(
	ctx context.Context,
	lgr log.Logger,
	deployer common.Address,
	artifactsFS foundry.StatDirFs,
	l1RPCUrl string,
	bcaster broadcaster.Broadcaster,
) (*ForkedL1, error) {
	rpcClient, err := rpc.Dial(l1RPCUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	host, err := env.DefaultForkedScriptHost(ctx, bcaster, lgr, deployer, artifactsFS, rpcClient)
	if err != nil {
		rpcClient.Close()
		return nil, fmt.Errorf("failed to create forked script host: %w", err)
	}
	return &ForkedL1{Backend: FromHost(host), bcaster: bcaster, rpc: rpcClient}, nil
}

// ForkedSpawnOpts returns the rustengine.SpawnOpts shared by every forked-engine spawn (apply
// Live/Calldata/Noop, bootstrap, upgrade, manage, sysgo, op-fetcher), mirroring env.DefaultScriptHost's
// context: the default script chain/block env, the CREATE2 deployer, and isolated broadcasts (whose
// gas accounting must match the Go host for broadcast parity). The non-forked genesis engine layers
// NoMaxCodeSize on top of this base. Defined in one place so the Go host and the engine cannot drift.
func ForkedSpawnOpts(artifactsDir string) rustengine.SpawnOpts {
	return rustengine.SpawnOpts{
		ArtifactsDir:       artifactsDir,
		ChainID:            bigs.Uint64Strict(script.DefaultContext.ChainID),
		Create2Deployer:    true,
		IsolatedBroadcasts: true, // mirrors env.DefaultScriptHost's script.WithIsolatedBroadcasts
		BlockNum:           script.DefaultContext.BlockNum,
		Timestamp:          script.DefaultContext.Timestamp,
		PrevRandao:         script.DefaultContext.PrevRandao,
	}
}

func newForkedL1Engine(
	ctx context.Context,
	lgr log.Logger,
	deployer common.Address,
	artifactsFS foundry.StatDirFs,
	l1RPCUrl string,
	bcaster broadcaster.Broadcaster,
) (*ForkedL1, error) {
	artifactsDir, err := rustengine.ArtifactsDir(artifactsFS)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve L1 artifacts directory: %w", err)
	}
	binPath, err := rustengine.EngineBinary(ctx, lgr)
	if err != nil {
		return nil, fmt.Errorf("failed to provision op-script-engine binary: %w", err)
	}

	eng, err := rustengine.Spawn(binPath, ForkedSpawnOpts(artifactsDir), rustengine.NewLogWriter(lgr))
	if err != nil {
		return nil, fmt.Errorf("failed to spawn op-script-engine for forked L1: %w", err)
	}

	// Pin the fork to the latest block, exactly as env.DefaultForkedScriptHost does. The block is
	// resolved Go-side so both engines fork at the same height; the engine dials l1RPCUrl itself.
	forkBlock, err := latestBlock(ctx, l1RPCUrl)
	if err != nil {
		eng.Close()
		return nil, fmt.Errorf("failed to resolve latest L1 block: %w", err)
	}
	if _, err := eng.CreateSelectFork(l1RPCUrl, &forkBlock); err != nil {
		eng.Close()
		return nil, fmt.Errorf("failed to select fork: %w", err)
	}

	fa := &foundry.ArtifactsFS{FS: artifactsFS}
	return &ForkedL1{Backend: FromEngine(eng, deployer, fa), bcaster: bcaster, engine: eng}, nil
}

func latestBlock(ctx context.Context, l1RPCUrl string) (uint64, error) {
	rpcClient, err := rpc.Dial(l1RPCUrl)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer rpcClient.Close()
	latest, err := ethclient.NewClient(rpcClient).HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest block: %w", err)
	}
	return bigs.Uint64Strict(latest.Number), nil
}

// DrainBroadcasts hands the broadcasts the engine captured during the last script run to the Go
// broadcaster, mirroring the Go host's synchronous WithBroadcastHook delivery. It is a no-op on the
// Go path (the hook already delivered them). Call it after each script run, before Broadcast.
func (f *ForkedL1) DrainBroadcasts() error {
	if f.engine == nil {
		return nil
	}
	bcasts, err := f.engine.TakeBroadcasts()
	if err != nil {
		return fmt.Errorf("failed to take engine broadcasts: %w", err)
	}
	for _, b := range bcasts {
		f.bcaster.Hook(b)
	}
	return nil
}

// Close tears down the backend: the spawned engine process and/or the fork RPC client.
func (f *ForkedL1) Close() {
	if f.engine != nil {
		f.engine.Close()
	}
	if f.rpc != nil {
		f.rpc.Close()
	}
}
