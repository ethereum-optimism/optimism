package devnet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

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

// RPCReplayOrRecord returns a replay proxy configured based on environment and fixture state:
//
//  1. If fixtures exist on disk and RPC_REPLAY_RECORD is not set: replays from fixtures (fast, offline).
//  2. If RPC_REPLAY_RECORD=true and SEPOLIA_RPC_URL is set: records to fixturePath.
//  3. If no fixtures and SEPOLIA_RPC_URL is set: passes through to live RPC via RetryProxy.
//  4. If no fixtures and no SEPOLIA_RPC_URL: returns an error.
//
// Returns the proxy and a cleanup function. The proxy's Endpoint() can be used
// anywhere an RPC URL is expected.
func RPCReplayOrRecord(lgr log.Logger, fixturePath string) (*RPCReplayProxy, CleanupFunc, error) {
	rpcURL := os.Getenv("SEPOLIA_RPC_URL")

	// Prefer replay from existing fixtures unless recording is explicitly requested
	if !IsRPCRecordMode() {
		if _, err := os.Stat(fixturePath); err == nil {
			proxy := NewRPCReplayProxy(lgr, "", fixturePath, RPCReplayModeReplay)
			if err := proxy.Start(); err != nil {
				return nil, nil, fmt.Errorf("failed to start replay proxy: %w", err)
			}
			return proxy, func() error { return proxy.Stop() }, nil
		}
	}

	if rpcURL == "" {
		return nil, nil, fmt.Errorf("no fixture file at %s and SEPOLIA_RPC_URL not set", fixturePath)
	}

	if IsRPCRecordMode() {
		proxy := NewRPCReplayProxy(lgr, rpcURL, fixturePath, RPCReplayModeRecord)
		if err := proxy.Start(); err != nil {
			return nil, nil, fmt.Errorf("failed to start record proxy: %w", err)
		}
		return proxy, func() error { return proxy.Stop() }, nil
	}

	// Pass-through: forward to live RPC without recording
	retryProxy := NewRetryProxy(lgr, rpcURL)
	if err := retryProxy.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start retry proxy: %w", err)
	}
	proxy := NewRPCReplayProxy(lgr, retryProxy.Endpoint(), fixturePath, RPCReplayModePassthrough)
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

	proxy.SetForkBlock(block)

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

// NewForkedSepoliaFromFixture creates a forked Sepolia anvil instance that reads
// its fork block from the fixture metadata. During recording, it queries
// eth_blockNumber from SEPOLIA_RPC_URL to determine the fork block. During replay,
// it reads the block from the fixture. If the fixture has no metadata, fallbackBlock
// is used (backward compatibility with fixtures recorded before metadata support).
func NewForkedSepoliaFromFixture(lgr log.Logger, fixturePath string, fallbackBlock uint64) (*Anvil, CleanupFunc, error) {
	block := FixtureForkBlock(fixturePath, fallbackBlock)

	if IsRPCRecordMode() {
		rpcURL := os.Getenv("SEPOLIA_RPC_URL")
		if rpcURL != "" {
			latestBlock, err := queryLatestBlock(rpcURL)
			if err != nil {
				lgr.Warn("failed to query latest block, using fallback", "err", err, "fallback", fallbackBlock)
			} else {
				block = latestBlock
			}
		}
	}

	return NewForkedSepoliaFromBlockWithReplay(lgr, block, fixturePath)
}

// queryLatestBlock queries eth_blockNumber from the given RPC URL.
func queryLatestBlock(rpcURL string) (uint64, error) {
	reqBody := []byte(`{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}`)
	resp, err := http.Post(rpcURL, "application/json", bytes.NewReader(reqBody)) //nolint:gosec,noctx
	if err != nil {
		return 0, fmt.Errorf("eth_blockNumber request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}
	var result struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}
	blockNum, err := strconv.ParseUint(strings.TrimPrefix(result.Result, "0x"), 16, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse block number %q: %w", result.Result, err)
	}
	return blockNum, nil
}
