package server

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/ethereum-optimism/optimism/op-service/rpc"
)

type TestServer struct {
	rpcSrv *rpc.Server

	stopped atomic.Bool
}

func (ts *TestServer) Stopped() bool {
	return ts.stopped.Load()
}

func FromCLIConfig(cfg *CLIConfig) (*TestServer, error) {
	var ts TestServer
	if err := ts.initFromCLIConfig(cfg); err != nil {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately, no need to be graceful with shutdown if we fail to start fully.
		if closeErr := ts.Stop(ctx); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close server after failed setup: %w", closeErr))
		}
		return nil, err
	}
	return &ts, nil
}

func (ts *TestServer) initFromCLIConfig(cfg *CLIConfig) error {
	ts.rpcSrv = rpc.NewServer(
		cfg.RPC.ListenAddr,
		cfg.RPC.ListenPort,
		cfg.Version)

	// TODO load resources config from cfg.Config
	return nil
}

func (ts *TestServer) Start(ctx context.Context) error {
	// TODO start workers
	//
	// TODO load preset resources of each worker

	if err := ts.rpcSrv.Start(); err != nil {
		return fmt.Errorf("failed to start RPC server: %w", err)
	}

	return nil
}

func (ts *TestServer) Stop(ctx context.Context) error {
	var result error
	if ts.rpcSrv != nil {
		if err := ts.rpcSrv.Stop(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop RPC server: %w", err))
		}
	}
	ts.stopped.Store(true)
	return result
}
