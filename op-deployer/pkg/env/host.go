package env

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script/forking"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

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
	return ScriptHost(bcaster, lgr, artifacts, scriptCtx, additionalOpts...)
}

func ScriptHost(
	bcaster broadcaster.Broadcaster,
	lgr log.Logger,
	artifacts foundry.StatDirFs,
	scriptCtx script.Context,
	additionalOpts ...script.HostOption,
) (*script.Host, error) {
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
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get L1 chain ID: %w", err)
	}
	// Selecting an L1 fork loads its state but does not update the host's EVM
	// context. Populate that context from the same chain and head so scripts
	// observe consistent block metadata.
	scriptCtx := script.DefaultContext
	scriptCtx.ChainID = chainID
	scriptCtx.Sender = deployer
	scriptCtx.Origin = deployer
	scriptCtx.FeeRecipient = latest.Coinbase
	scriptCtx.BlockNum = bigs.Uint64Strict(latest.Number)
	scriptCtx.Timestamp = latest.Time
	scriptCtx.PrevRandao = latest.MixDigest

	return forkedScriptHost(
		bcaster,
		lgr,
		artifacts,
		forkRPC,
		latest.Number,
		scriptCtx,
		additionalOpts...,
	)
}

func ForkedScriptHost(
	bcaster broadcaster.Broadcaster,
	lgr log.Logger,
	deployer common.Address,
	artifacts foundry.StatDirFs,
	forkRPC *rpc.Client,
	blockNumber *big.Int,
	additionalOpts ...script.HostOption,
) (*script.Host, error) {
	scriptCtx := script.DefaultContext
	scriptCtx.Sender = deployer
	scriptCtx.Origin = deployer

	return forkedScriptHost(
		bcaster,
		lgr,
		artifacts,
		forkRPC,
		blockNumber,
		scriptCtx,
		additionalOpts...,
	)
}

func forkedScriptHost(
	bcaster broadcaster.Broadcaster,
	lgr log.Logger,
	artifacts foundry.StatDirFs,
	forkRPC *rpc.Client,
	blockNumber *big.Int,
	scriptCtx script.Context,
	additionalOpts ...script.HostOption,
) (*script.Host, error) {
	h, err := ScriptHost(
		bcaster,
		lgr,
		artifacts,
		scriptCtx,
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
		return nil, fmt.Errorf("failed to create forked script host: %w", err)
	}

	if _, err := h.CreateSelectFork(
		script.ForkWithURLOrAlias("main"),
		script.ForkWithBlockNumberU256(blockNumber),
	); err != nil {
		return nil, fmt.Errorf("failed to select fork: %w", err)
	}

	return h, nil
}
