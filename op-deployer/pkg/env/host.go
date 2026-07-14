package env

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script/forking"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// DefaultScriptHost builds the non-forked, in-memory Go script.Host.
//
// Non-forked script execution defaults to the Rust op-script-engine (DefaultScriptEngine); that
// routing happens per operation at the callers that consult the engine selection (the
// DeploymentTargetGenesis L1 deploy stages in apply.go, pipeline.GenerateL2Genesis,
// op-chain-ops/interopgen). This Go host backs the --script-engine=go fallback and the non-forked
// callers not yet routed to the engine (e.g. inspect/semvers). Forked callers must use
// ForkedScriptHost instead.
func DefaultScriptHost(
	bcaster broadcaster.Broadcaster,
	lgr log.Logger,
	deployer common.Address,
	artifacts foundry.StatDirFs,
	additionalOpts ...script.HostOption,
) (*script.Host, error) {
	scriptCtx := script.DefaultContext
	scriptCtx.Sender = deployer
	scriptCtx.Origin = deployer
	h := script.NewHost(
		lgr,
		&foundry.ArtifactsFS{FS: artifacts},
		nil,
		scriptCtx,
		append([]script.HostOption{
			script.WithBroadcastHook(bcaster.Hook),
			script.WithIsolatedBroadcasts(),
			script.WithCreate2Deployer(),
		}, additionalOpts...)...,
	)

	if err := h.EnableCheats(); err != nil {
		return nil, fmt.Errorf("failed to enable cheats: %w", err)
	}

	return h, nil
}

// DefaultForkedScriptHost builds a fork-backed Go script.Host bound to the latest block of forkRPC.
//
// This is the Go implementation of a forked host, reached via the --script-engine=go fallback. The
// Rust op-script-engine supports fork mode, and the forked callers select it by default through the
// scriptbackend.NewForkedL1 factory (apply.go Live/Calldata/Noop, bootstrap, manage.Migrate, and the
// upgrade CLI). The callers still building this host directly — sysgo opcm_upgrade, the
// integration_test OPCM-registry-walk upgrade path, op-fetcher, and some forked test suites — are
// being migrated to that factory.
func DefaultForkedScriptHost(
	ctx context.Context,
	bcaster broadcaster.Broadcaster,
	lgr log.Logger,
	deployer common.Address,
	artifacts foundry.StatDirFs,
	forkRPC *rpc.Client,
	additionalOpts ...script.HostOption,
) (*script.Host, error) {
	client := ethclient.NewClient(forkRPC)

	latest, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest block: %w", err)
	}

	return ForkedScriptHost(
		bcaster,
		lgr,
		deployer,
		artifacts,
		forkRPC,
		latest.Number,
		additionalOpts...,
	)
}

// ForkedScriptHost builds a fork-backed Go script.Host at an explicit block number. It is the Go
// counterpart of the Rust engine's fork mode; see the DefaultForkedScriptHost doc for which callers
// still use it and the --script-engine=go fallback.
func ForkedScriptHost(
	bcaster broadcaster.Broadcaster,
	lgr log.Logger,
	deployer common.Address,
	artifacts foundry.StatDirFs,
	forkRPC *rpc.Client,
	blockNumber *big.Int,
	additionalOpts ...script.HostOption,
) (*script.Host, error) {
	h, err := DefaultScriptHost(
		bcaster,
		lgr,
		deployer,
		artifacts,
		append([]script.HostOption{
			script.WithForkHook(func(cfg *script.ForkConfig) (forking.ForkSource, error) {
				src, err := forking.RPCSourceByNumber(cfg.URLOrAlias, forkRPC, *cfg.BlockNumber)
				if err != nil {
					return nil, fmt.Errorf("failed to create RPC fork source: %w", err)
				}
				return forking.Cache(src), nil
			}),
		}, additionalOpts...)...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create default script host: %w", err)
	}

	if _, err := h.CreateSelectFork(
		script.ForkWithURLOrAlias("main"),
		script.ForkWithBlockNumberU256(blockNumber),
	); err != nil {
		return nil, fmt.Errorf("failed to select fork: %w", err)
	}

	return h, nil
}
