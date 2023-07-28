package fault

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// Service provides a clean interface for the challenger to interact
// with the fault package.
type Service interface {
	// MonitorGame monitors the fault dispute game and attempts to progress it.
	MonitorGame(context.Context) error
}

type service struct {
	agent                   *Agent
	agreeWithProposedOutput bool
	caller                  *FaultCaller
	logger                  log.Logger
}

// NewService creates a new Service.
func NewService(ctx context.Context, logger log.Logger, cfg *config.Config) (*service, error) {
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

	loader := NewLoader(contract)
	gameLogger := logger.New("game", cfg.GameAddress)
	responder, err := NewFaultResponder(gameLogger, txMgr, cfg.GameAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create the responder: %w", err)
	}

	var trace types.TraceProvider
	switch cfg.TraceType {
	case config.TraceTypeCannon:
		trace = cannon.NewCannonTraceProvider(logger, cfg)
	case config.TraceTypeAlphabet:
		trace = alphabet.NewAlphabetProvider(cfg.AlphabetTrace, uint64(cfg.GameDepth))
	default:
		return nil, fmt.Errorf("unsupported trace type: %v", cfg.TraceType)
	}

	agent := NewAgent(loader, cfg.GameDepth, trace, responder, cfg.AgreeWithProposedOutput, gameLogger)

	caller, err := NewFaultCallerFromBindings(cfg.GameAddress, client, gameLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to bind the fault contract: %w", err)
	}

	return &service{
		agent:                   agent,
		agreeWithProposedOutput: cfg.AgreeWithProposedOutput,
		caller:                  caller,
		logger:                  gameLogger,
	}, nil
}

// MonitorGame monitors the fault dispute game and attempts to progress it.
func (s *service) MonitorGame(ctx context.Context) error {
	return MonitorGame(ctx, s.logger, s.agreeWithProposedOutput, s.agent, s.caller)
}
