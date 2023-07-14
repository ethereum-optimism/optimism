package op_challenger

import (
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
func Main(logger log.Logger, cfg *config.Config) error {
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
	gameDepth := 4
	trace := fault.NewAlphabetProvider(cfg.AlphabetTrace, uint64(gameDepth))

	agent := fault.NewAgent(loader, gameDepth, trace, responder, cfg.AgreeWithProposedOutput, logger)

	logger.Info("Fault game started")

	for {
		logger.Info("Performing action")
		_ = agent.Act()
		time.Sleep(300 * time.Millisecond)
	}
}
