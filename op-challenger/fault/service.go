package fault

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/client"
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
	txMgr, err := txmgr.NewSimpleTxManager("challenger", logger, &metrics.NoopTxMetrics{}, cfg.TxMgrConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create the transaction manager: %w", err)
	}

	client, err := client.DialEthClientWithTimeout(client.DefaultDialTimeout, logger, cfg.L1EthRpc)
	if err != nil {
		return nil, fmt.Errorf("failed to dial L1: %w", err)
	}

	contract, err := bindings.NewFaultDisputeGameCaller(cfg.GameAddress, client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind the fault dispute game contract: %w", err)
	}

	loader := NewLoader(contract)

	gameDepth, err := loader.FetchGameDepth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the game depth: %w", err)
	}
	gameDepth = uint64(gameDepth)

	var trace types.TraceProvider
	var updater types.OracleUpdater
	switch cfg.TraceType {
	case config.TraceTypeCannon:
		trace, err = cannon.NewTraceProvider(ctx, logger, cfg, client)
		if err != nil {
			return nil, fmt.Errorf("create cannon trace provider: %w", err)
		}
		updater, err = cannon.NewOracleUpdater(ctx, logger, txMgr, cfg.GameAddress, client)
		if err != nil {
			return nil, fmt.Errorf("failed to create the cannon updater: %w", err)
		}
	case config.TraceTypeAlphabet:
		trace = alphabet.NewTraceProvider(cfg.AlphabetTrace, gameDepth)
		updater = alphabet.NewOracleUpdater(logger)
	default:
		return nil, fmt.Errorf("unsupported trace type: %v", cfg.TraceType)
	}

	return newTypedService(ctx, logger, cfg, loader, gameDepth, client, trace, updater, txMgr)
}

// newTypedService creates a new Service from a provided trace provider.
func newTypedService(ctx context.Context, logger log.Logger, cfg *config.Config, loader Loader, gameDepth uint64, client *ethclient.Client, provider types.TraceProvider, updater types.OracleUpdater, txMgr txmgr.TxManager) (*service, error) {
	if err := ValidateAbsolutePrestate(ctx, provider, loader); err != nil {
		return nil, fmt.Errorf("failed to validate absolute prestate: %w", err)
	}

	gameLogger := logger.New("game", cfg.GameAddress)
	responder, err := NewFaultResponder(gameLogger, txMgr, cfg.GameAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create the responder: %w", err)
	}

	caller, err := NewFaultCallerFromBindings(cfg.GameAddress, client, gameLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to bind the fault contract: %w", err)
	}

	return &service{
		agent:                   NewAgent(loader, int(gameDepth), provider, responder, updater, cfg.AgreeWithProposedOutput, gameLogger),
		agreeWithProposedOutput: cfg.AgreeWithProposedOutput,
		caller:                  caller,
		logger:                  gameLogger,
	}, nil
}

// ValidateAbsolutePrestate validates the absolute prestate of the fault game.
func ValidateAbsolutePrestate(ctx context.Context, trace types.TraceProvider, loader Loader) error {
	providerPrestate, err := trace.AbsolutePreState(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the trace provider's absolute prestate: %w", err)
	}
	providerPrestateHash, err := trace.StateHash(ctx, providerPrestate)
	if err != nil {
		return fmt.Errorf("failed to compute state-hash: %w", err)
	}
	onchainPrestateHash, err := loader.FetchAbsolutePrestateHash(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the onchain absolute prestate: %w", err)
	}
	if providerPrestateHash != onchainPrestateHash {
		return fmt.Errorf("trace provider's absolute prestate does not match onchain absolute prestate")
	}
	return nil
}

// MonitorGame monitors the fault dispute game and attempts to progress it.
func (s *service) MonitorGame(ctx context.Context) error {
	return MonitorGame(ctx, s.logger, s.agreeWithProposedOutput, s.agent, s.caller)
}
