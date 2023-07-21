package op_challenger

import (
	"context"
	"fmt"

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

	gameLogger := logger.New("game", cfg.GameAddress)
	loader := fault.NewLoader(contract)
	responder, err := fault.NewFaultResponder(gameLogger, txMgr, cfg.GameAddress)
	if err != nil {
		return fmt.Errorf("failed to create the responder: %w", err)
	}
	trace := fault.NewAlphabetProvider(cfg.AlphabetTrace, uint64(cfg.GameDepth))

	agent := fault.NewAgent(loader, cfg.GameDepth, trace, responder, cfg.AgreeWithProposedOutput, gameLogger)

	caller, err := fault.NewFaultCallerFromBindings(cfg.GameAddress, client, gameLogger)
	if err != nil {
		return fmt.Errorf("failed to bind the fault contract: %w", err)
	}

	return fault.MonitorGame(ctx, gameLogger, cfg.AgreeWithProposedOutput, agent, caller)
}
