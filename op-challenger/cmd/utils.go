package main

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

type ContractCreator[T any] func(context.Context, contractMetrics.ContractMetricer, common.Address, *batching.MultiCaller) (T, error)

func Interruptible(action cli.ActionFunc) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		ctx.Context = ctxinterrupt.WithCancelOnInterrupt(ctx.Context)
		return action(ctx)
	}
}
func AddrFromFlag(flagName string) func(ctx *cli.Context) (common.Address, error) {
	return func(ctx *cli.Context) (common.Address, error) {
		gameAddr, err := opservice.ParseAddress(ctx.String(flagName))
		if err != nil {
			return common.Address{}, err
		}
		return gameAddr, nil
	}
}

// NewContractWithTxMgr creates a new contract and a transaction manager.
func NewContractWithTxMgr[T any](ctx *cli.Context, getAddr func(ctx *cli.Context) (common.Address, error), creator ContractCreator[T]) (T, txmgr.TxManager, error) {
	var contract T
	caller, txMgr, err := newClientsFromCLI(ctx)
	if err != nil {
		return contract, nil, err
	}

	created, err := newContractFromCLI(ctx, getAddr, caller, creator)
	if err != nil {
		return contract, nil, err
	}

	return created, txMgr, nil
}

// newContractFromCLI creates a new contract from the CLI context.
func newContractFromCLI[T any](ctx *cli.Context, getAddr func(ctx *cli.Context) (common.Address, error), caller *batching.MultiCaller, creator ContractCreator[T]) (T, error) {
	var contract T
	gameAddr, err := getAddr(ctx)
	if err != nil {
		return contract, err
	}

	created, err := creator(ctx.Context, contractMetrics.NoopContractMetrics, gameAddr, caller)
	if err != nil {
		return contract, fmt.Errorf("failed to create contract bindings: %w", err)
	}

	return created, nil
}

// newClientsFromCLI creates a new caller and transaction manager from the CLI context.
func newClientsFromCLI(ctx *cli.Context) (*batching.MultiCaller, txmgr.TxManager, error) {
	logger, err := setupLogging(ctx)
	if err != nil {
		return nil, nil, err
	}

	rpcUrl := ctx.String(flags.L1EthRpcFlag.Name)
	if rpcUrl == "" {
		return nil, nil, fmt.Errorf("missing %v", flags.L1EthRpcFlag.Name)
	}

	l1Client, err := dial.DialEthClientWithTimeout(ctx.Context, dial.DefaultDialTimeout, logger, rpcUrl)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial L1: %w", err)
	}
	defer l1Client.Close()

	caller := batching.NewMultiCaller(l1Client.Client(), batching.DefaultBatchSize)
	txMgrConfig := txmgr.ReadCLIConfig(ctx)
	txMgr, err := txmgr.NewSimpleTxManager("challenger", logger, &metrics.NoopTxMetrics{}, txMgrConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create the transaction manager: %w", err)
	}

	logger.Info("Configured transaction manager", "sender", txMgr.From())
	return caller, txMgr, nil
}
