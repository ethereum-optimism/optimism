package devnet

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-service/log"
)

type CleanupFunc func() error

func NewForked(lgr log.Logger, rpcURL string, anvilOpts ...AnvilOption) (*Anvil, CleanupFunc, error) {
	retryProxy := NewRetryProxy(lgr, rpcURL)
	if err := retryProxy.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start retry proxy: %w", err)
	}

	defaultOpts := []AnvilOption{
		WithForkURL(retryProxy.Endpoint()),
		WithBlockTime(3),
		// TODO(#23640): Remove once the repo supports Amsterdam. Anvil forks Sepolia under its live
		// Amsterdam hardfork, where op-deployer's broadcasts run out of gas on anvil's own estimates.
		WithHardfork("bpo2"),
	}
	anvil, err := NewAnvil(lgr, append(defaultOpts, anvilOpts...)...)
	if err != nil {
		_ = retryProxy.Stop()
		return nil, nil, fmt.Errorf("failed to create Anvil: %w", err)
	}

	if err := anvil.Start(); err != nil {
		_ = retryProxy.Stop()
		return nil, nil, fmt.Errorf("failed to start Anvil: %w", err)
	}

	cleanup := func() error {
		if err := anvil.Stop(); err != nil {
			return fmt.Errorf("failed to stop Anvil: %w", err)
		}
		if err := retryProxy.Stop(); err != nil {
			return fmt.Errorf("failed to stop retry proxy: %w", err)
		}
		return nil
	}

	return anvil, cleanup, nil
}

func NewForkedSepolia(lgr log.Logger) (*Anvil, CleanupFunc, error) {
	url := os.Getenv("SEPOLIA_RPC_URL")
	if url == "" {
		return nil, nil, fmt.Errorf("SEPOLIA_RPC_URL not set")
	}
	return NewForked(lgr, url)
}

func NewForkedSepoliaFromBlock(lgr log.Logger, block uint64) (*Anvil, CleanupFunc, error) {
	url := os.Getenv("SEPOLIA_RPC_URL")
	if url == "" {
		return nil, nil, fmt.Errorf("SEPOLIA_RPC_URL not set")
	}
	return NewForked(lgr, url, WithForkBlockNumber(block))
}
