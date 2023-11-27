package main

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-preimage/stack"
)

func run(logger log.Logger, source stack.Source, sink stack.Sink, cfg *RunConfig) (stack.Stoppable, error) {
	hostPreimageCh, hostHintCh, stopHost, err := source()
	if err != nil {
		return nil, fmt.Errorf("failed to init source: %w", err)
	}

	// The forwarders are attached to the source
	preimageForward := stack.PreimageForwarder(hostPreimageCh)
	hintForward := stack.HintForwarder(hostHintCh)

	preimageFn := func(key [32]byte) ([]byte, error) {
		if cfg.LogPreimageKeys {
			logger.Info("processing preimage request", "key", hexutil.Bytes(key[:]))
		}
		value, err := preimageForward(key)
		if err != nil {
			logger.Error("preimage request error", "err", err)
			return nil, err
		}
		if cfg.LogPreimageValues {
			logger.Info("received preimage response",
				"key", hexutil.Bytes(key[:]), "value", hexutil.Bytes(value))
		}
		return value, nil
	}
	hintFn := func(hint string) error {
		if cfg.LogHints {
			logger.Info("processing hint", "hint", hint)
		}
		return hintForward(hint)
	}

	pClientRW, pHostRW, hClientRW, hHostRW, stopPipes, err := stack.MiddlewarePipes()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to create middleware data pipes: %w", err), stopHost.Stop(context.Background()))
	}

	// one end of the channels is attached to the forwarders
	var handlerErrGrp errgroup.Group
	handlerErrGrp.Go(func() error {
		if err := stack.HandlePreimages(logger, pClientRW, preimageFn); err != nil {
			_ = hClientRW.Close() // stop hint handling also
			return err
		}
		return nil
	})
	handlerErrGrp.Go(func() error {
		if err := stack.HandleHints(logger, hClientRW, hintFn); err != nil {
			_ = pClientRW.Close() // stop preimage handling also
			return err
		}
		return nil
	})

	// other end of the channels is attached to the sink
	stopClient, err := sink(pHostRW, hHostRW)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to init sink: %w", err),
			stopHost.Stop(context.Background()), stopPipes.Stop(context.Background()))
	}

	return stack.StopFn(func(ctx context.Context) error {
		var result error
		result = errors.Join(result, stopClient.Stop(ctx))
		result = errors.Join(result, stopHost.Stop(ctx))
		result = errors.Join(result, stopPipes.Stop(ctx))
		result = errors.Join(result, handlerErrGrp.Wait())
		return result
	}), nil
}
