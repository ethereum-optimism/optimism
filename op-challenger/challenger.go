package op_challenger

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/fault"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// Main is the programmatic entry-point for running op-challenger
func Main(ctx context.Context, logger log.Logger, cfg *config.Config) error {
	client, err := ethclient.Dial(cfg.L1EthRpc)
	if err != nil {
		return fmt.Errorf("failed to dial L1: %w", err)
	}

	txMgr, err := txmgr.NewSimpleTxManager("challenger", logger, &metrics.NoopTxMetrics{}, cfg.TxMgrConfig)
	if err != nil {
		return fmt.Errorf("failed to create the transaction manager: %w", err)
	}

	contract, err := bindings.NewFaultDisputeGameCaller(cfg.GameAddress, client)
	if err != nil {
		return fmt.Errorf("failed to bind the fault dispute game contract: %w", err)
	}

	loader := fault.NewLoader(logger, contract)
	responder, err := fault.NewFaultResponder(logger, txMgr, cfg.GameAddress)
	if err != nil {
		return fmt.Errorf("failed to create the responder: %w", err)
	}
	trace := fault.NewAlphabetProvider(cfg.AlphabetTrace, uint64(cfg.GameDepth))

	agent := fault.NewAgent(loader, cfg.GameDepth, trace, responder, cfg.AgreeWithProposedOutput, logger)

	caller, err := fault.NewFaultCallerFromBindings(cfg.GameAddress, client, logger)
	if err != nil {
		return fmt.Errorf("failed to bind the fault contract: %w", err)
	}

	logger.Info("Fault game started")

	for {
		logger.Info("Performing action")
		_ = agent.Act(ctx)
		status, _ := caller.GetGameStatus(ctx)
		if status != 0 {
			caller.LogGameStatus(ctx)
			return nil
		} else {
			caller.LogGameInfo(ctx)
		}
		select {
		case <-time.After(300 * time.Millisecond):
		// Continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
