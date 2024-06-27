package backend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/ethereum-optimism/optimism/op-supervisor/config"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/source"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/frontend"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type Metrics interface {
	source.Metrics
}

type SupervisorBackend struct {
	started atomic.Bool
	logger  log.Logger

	chainMonitors []*source.ChainMonitor

	// TODO(protocol-quest#287): collection of logdbs per chain
	// TODO(protocol-quest#288): collection of logdb updating services per chain
}

var _ frontend.Backend = (*SupervisorBackend)(nil)

var _ io.Closer = (*SupervisorBackend)(nil)

func NewSupervisorBackend(ctx context.Context, logger log.Logger, m Metrics, cfg *config.Config) (*SupervisorBackend, error) {
	chainMonitors := make([]*source.ChainMonitor, len(cfg.L2RPCs))
	for i, rpc := range cfg.L2RPCs {
		monitor, err := source.NewChainMonitor(ctx, logger, m, rpc)
		if err != nil {
			return nil, fmt.Errorf("failed to create monitor for rpc %v: %w", rpc, err)
		}
		chainMonitors[i] = monitor
	}
	return &SupervisorBackend{
		logger:        logger,
		chainMonitors: chainMonitors,
	}, nil
}

func (su *SupervisorBackend) Start(ctx context.Context) error {
	if !su.started.CompareAndSwap(false, true) {
		return errors.New("already started")
	}
	// TODO(protocol-quest#288): start logdb updating services of all chains
	for _, monitor := range su.chainMonitors {
		if err := monitor.Start(); err != nil {
			return fmt.Errorf("failed to start chain monitor: %w", err)
		}
	}
	return nil
}

func (su *SupervisorBackend) Stop(ctx context.Context) error {
	if !su.started.CompareAndSwap(true, false) {
		return errors.New("already stopped")
	}
	// TODO(protocol-quest#288): stop logdb updating services of all chains
	var errs error
	for _, monitor := range su.chainMonitors {
		if err := monitor.Stop(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to stop chain monitor: %w", err))
		}
	}
	return errs
}

func (su *SupervisorBackend) Close() error {
	// TODO(protocol-quest#288): close logdb of all chains
	return nil
}

func (su *SupervisorBackend) CheckMessage(identifier types.Identifier, payloadHash common.Hash) (types.SafetyLevel, error) {
	// TODO(protocol-quest#288): hook up to logdb lookup
	return types.CrossUnsafe, nil
}

func (su *SupervisorBackend) CheckBlock(chainID *hexutil.U256, blockHash common.Hash, blockNumber hexutil.Uint64) (types.SafetyLevel, error) {
	// TODO(protocol-quest#288): hook up to logdb lookup
	return types.CrossUnsafe, nil
}
