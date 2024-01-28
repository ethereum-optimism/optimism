package main

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-preimage/stack"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
)

// OracleService is a simple service that runs through the preimage-oracle communication
// between the source (server) and sink (client).
type OracleService struct {
	logger log.Logger

	source stack.Source
	sink   stack.Sink

	stopped atomic.Bool
	stopper stack.Stoppable

	cfg *RunConfig
}

func NewOracleService(logger log.Logger, cfg *RunConfig, closeApp context.CancelCauseFunc) *OracleService {
	var source stack.Source
	if cfg.HostCommand == "" {
		source = stack.GlobalSource()
	} else {
		source = stack.ExecSource(cfg.HostCommand, os.Stdout, os.Stderr, closeApp)
	}

	var sink stack.Sink
	if cfg.ClientCommand == "" {
		sink = stack.GlobalSink()
	} else {
		sink = stack.ExecSink(cfg.ClientCommand, os.Stdout, os.Stderr, closeApp)
	}

	return &OracleService{
		logger: logger,
		source: source,
		sink:   sink,
		cfg:    cfg,
	}
}

func (b *OracleService) Start(ctx context.Context) error {
	stop, err := run(b.logger, b.source, b.sink, b.cfg)
	if err != nil {
		return fmt.Errorf("failed to start: %w", err)
	}
	b.stopper = stop
	return nil
}

func (b *OracleService) Stop(ctx context.Context) error {
	defer b.stopped.Store(true)
	return b.stopper.Stop(ctx)
}

func (b *OracleService) Stopped() bool {
	return b.stopped.Load()
}

var _ cliapp.Lifecycle = (*OracleService)(nil)
