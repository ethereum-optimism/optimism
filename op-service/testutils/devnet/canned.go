package devnet

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ethereum/go-ethereum/log"
)

type CleanupFunc func() error

func NewForked(lgr log.Logger, rpcURL string, anvilOpts ...AnvilOption) (*Anvil, CleanupFunc, error) {
	retryProxy := NewRetryProxy(lgr, rpcURL)
	if err := retryProxy.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start retry proxy: %w", err)
	}

	anvil, err := NewAnvil(lgr, append([]AnvilOption{WithForkURL(retryProxy.Endpoint()), WithBlockTime(3)}, anvilOpts...)...)
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

// RPCReplayFixtureDir returns the directory where RPC replay fixtures are stored.
// Fixtures live in op-service/testutils/devnet/fixtures/ relative to this source file.
func RPCReplayFixtureDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "fixtures")
}

// RPCReplayFixturePath returns the fixture file path for a given test name.
func RPCReplayFixturePath(name string) string {
	return filepath.Join(RPCReplayFixtureDir(), name+".json")
}

// IsRPCRecordMode returns true if RPC_REPLAY_RECORD=true is set in the environment.
func IsRPCRecordMode() bool {
	return os.Getenv("RPC_REPLAY_RECORD") == "true"
}

// RPCReplayOrRecord returns a replay proxy configured based on environment:
//   - If SEPOLIA_RPC_URL is set and RPC_REPLAY_RECORD=true: records to fixturePath
//   - If SEPOLIA_RPC_URL is not set: replays from fixturePath (no network needed)
//   - If SEPOLIA_RPC_URL is set but RPC_REPLAY_RECORD is not: acts as plain proxy (pass-through)
//
// Returns the proxy and a cleanup function. The proxy's Endpoint() can be used
// anywhere an RPC URL is expected.
func RPCReplayOrRecord(lgr log.Logger, fixturePath string) (*RPCReplayProxy, CleanupFunc, error) {
	rpcURL := os.Getenv("SEPOLIA_RPC_URL")

	if rpcURL == "" {
		// Replay mode: no external RPC needed
		proxy := NewRPCReplayProxy(lgr, "", fixturePath, RPCReplayModeReplay)
		if err := proxy.Start(); err != nil {
			return nil, nil, fmt.Errorf("failed to start replay proxy: %w", err)
		}
		return proxy, func() error { return proxy.Stop() }, nil
	}

	if IsRPCRecordMode() {
		// Record mode: forward to real Sepolia and save fixtures
		proxy := NewRPCReplayProxy(lgr, rpcURL, fixturePath, RPCReplayModeRecord)
		if err := proxy.Start(); err != nil {
			return nil, nil, fmt.Errorf("failed to start record proxy: %w", err)
		}
		return proxy, func() error { return proxy.Stop() }, nil
	}

	// Pass-through mode: just use RetryProxy as before (SEPOLIA_RPC_URL is set but not recording)
	retryProxy := NewRetryProxy(lgr, rpcURL)
	if err := retryProxy.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start retry proxy: %w", err)
	}
	// Wrap in a replay proxy that just passes through
	proxy := NewRPCReplayProxy(lgr, retryProxy.Endpoint(), fixturePath, RPCReplayModeRecord)
	if err := proxy.Start(); err != nil {
		_ = retryProxy.Stop()
		return nil, nil, fmt.Errorf("failed to start pass-through proxy: %w", err)
	}
	return proxy, func() error {
		err1 := proxy.Stop()
		err2 := retryProxy.Stop()
		if err1 != nil {
			return err1
		}
		return err2
	}, nil
}

// NewForkedSepoliaFromBlockWithReplay creates a forked Sepolia anvil instance using
// the replay proxy. In replay mode, no SEPOLIA_RPC_URL is needed.
func NewForkedSepoliaFromBlockWithReplay(lgr log.Logger, block uint64, fixturePath string) (*Anvil, CleanupFunc, error) {
	proxy, proxyCleanup, err := RPCReplayOrRecord(lgr, fixturePath)
	if err != nil {
		return nil, nil, err
	}

	anvil, err := NewAnvil(lgr, WithForkURL(proxy.Endpoint()), WithBlockTime(3), WithForkBlockNumber(block))
	if err != nil {
		_ = proxyCleanup()
		return nil, nil, fmt.Errorf("failed to create Anvil: %w", err)
	}

	if err := anvil.Start(); err != nil {
		_ = proxyCleanup()
		return nil, nil, fmt.Errorf("failed to start Anvil: %w", err)
	}

	cleanup := func() error {
		if err := anvil.Stop(); err != nil {
			return fmt.Errorf("failed to stop Anvil: %w", err)
		}
		return proxyCleanup()
	}

	return anvil, cleanup, nil
}
