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

type Challenger struct {
	agent  fault.Agent
	logger log.Logger
}

// NewChallenger creates a new challenger agent.
func NewChallenger(logger log.Logger, cfg *config.Config) (*Challenger, error) {
	client, err := ethclient.Dial(cfg.L1EthRpc)
	if err != nil {
		return nil, fmt.Errorf("failed to dial L1: %w", err)
	}

	txMgr, err := txmgr.NewSimpleTxManager("challenger", logger, &metrics.NoopTxMetrics{}, cfg.TxMgrConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create the transaction manager: %w", err)
	}

	contract, err := bindings.NewFaultDisputeGameCaller(cfg.GameAddress, client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind the fault dispute game contract: %w", err)
	}

	loader := fault.NewLoader(logger, contract)
	responder, err := fault.NewFaultResponder(logger, txMgr, cfg.GameAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create the responder: %w", err)
	}
	gameDepth := 4
	trace := fault.NewAlphabetProvider(cfg.AlphabetTrace, uint64(gameDepth))

	agent := fault.NewAgent(loader, gameDepth, trace, responder, cfg.AgreeWithProposedOutput, logger)

	return &Challenger{agent, logger}, nil
}

// Loop polls the specific fault dispute game contract & attempts to perform actions on a regular interval.
func (c *Challenger) Loop() {
	for {
		c.Act()
		time.Sleep(300 * time.Millisecond)
	}
}

// Act runs all currently possible actions on a dispute game contract & then exits.
func (c *Challenger) Act() {
	c.logger.Info("Performing action")
	_ = c.agent.Act()
}

// MainLoop is the programmatic entry-point for running op-challenger
func MainLoop(logger log.Logger, cfg *config.Config) error {
	c, err := NewChallenger(logger, cfg)
	if err != nil {
		return err
	}

	logger.Info("Fault game started")
	c.Loop()
	return nil
}

// MainAct is the programmatic entry-point for running op-challenger
func MainAct(logger log.Logger, cfg *config.Config) error {
	c, err := NewChallenger(logger, cfg)
	if err != nil {
		return err
	}

	logger.Info("Fault game started in single action mode")
	c.Act()
	return nil
}
