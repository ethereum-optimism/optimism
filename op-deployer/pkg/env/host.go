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
// routing happens per operation at the callers that consult the engine selection
// (pipeline.GenerateL2Genesis, op-chain-ops/interopgen). This Go host backs the
// --script-engine=go fallback and the non-forked stages not yet routed to the engine
// (L1 deploy / OPCM apply, inspect). Forked callers must use ForkedScriptHost instead.
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
// Forked hosts (CreateSelectFork against a live L1 — apply Live/Calldata, bootstrap, upgrade,
// manage, sysgo opcm_upgrade) always run on the Go script.Host regardless of the configured script
// engine: the Rust op-script-engine has no fork mode yet. This is a deliberate per-host-kind
// selection, not a silent fallback — Rust fork mode is a follow-up milestone.
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

// ForkedScriptHost builds a fork-backed Go script.Host at an explicit block number. Like
// DefaultForkedScriptHost, forked hosts always run on the Go engine (Rust fork mode is a follow-up
// milestone); see the DefaultForkedScriptHost doc for the per-host-kind engine rationale.
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
