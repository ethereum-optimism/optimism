package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Reference points for initial block estimation.
// These provide a starting point for binary search.
const (
	// Mainnet reference: Block 23964895, mined at timestamp 1765158191 (Dec 7, 2025)
	mainnetRefBlock     uint64 = 23964895
	mainnetRefTimestamp uint64 = 1765158191
	mainnetBlockTime    uint64 = 12
	mainnetChainID      uint64 = 1

	// Sepolia reference: Block 9819372, mined at timestamp 1765482912 (Dec 11, 2025)
	sepoliaRefBlock     uint64 = 9819372
	sepoliaRefTimestamp uint64 = 1765482912
	sepoliaBlockTime    uint64 = 12
	sepoliaChainID      uint64 = 11155111
)

// Default RPC URLs
const (
	defaultMainnetRPC = "https://eth.llamarpc.com"
	defaultSepoliaRPC = "https://ethereum-sepolia.publicnode.com"
)

func main() {
	chain := flag.String("chain", "mainnet", "Chain to calculate block for (mainnet or sepolia)")
	rpcURL := flag.String("rpc-url", "", "RPC URL (defaults to ETH_RPC_URL env var, then public RPC)")
	debug := flag.Bool("debug", false, "Enable debug timing output")
	flag.Parse()

	var refBlock, refTimestamp, blockTime, expectedChainID uint64
	var defaultRPC, chainName string

	switch *chain {
	case "mainnet":
		refBlock = mainnetRefBlock
		refTimestamp = mainnetRefTimestamp
		blockTime = mainnetBlockTime
		expectedChainID = mainnetChainID
		defaultRPC = defaultMainnetRPC
		chainName = "Ethereum Mainnet"
	case "sepolia":
		refBlock = sepoliaRefBlock
		refTimestamp = sepoliaRefTimestamp
		blockTime = sepoliaBlockTime
		expectedChainID = sepoliaChainID
		defaultRPC = defaultSepoliaRPC
		chainName = "Sepolia"
	default:
		fmt.Fprintf(os.Stderr, "unknown chain: %s (use mainnet or sepolia)\n", *chain)
		os.Exit(1)
	}

	// Priority: --rpc-url flag > ETH_RPC_URL env var > default
	rpc := *rpcURL
	if rpc == "" {
		rpc = os.Getenv("ETH_RPC_URL")
	}
	if rpc == "" {
		rpc = defaultRPC
	}

	// Validate chain ID
	actualChainID, err := getChainID(rpc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting chain ID: %v\n", err)
		os.Exit(1)
	}
	if actualChainID != expectedChainID {
		fmt.Fprintf(os.Stderr, "error: RPC returned chain ID %d, expected %d (%s)\n", actualChainID, expectedChainID, chainName)
		os.Exit(1)
	}

	targetTimestamp := sundayMidnightUTC()

	// Get initial estimate using linear extrapolation
	estimate := estimateBlock(refBlock, refTimestamp, blockTime, targetTimestamp)

	// Binary search to find the block (within ~15 min accuracy = 75 blocks)
	// This is sufficient since the cache warming job runs at 01:00 UTC,
	// giving 30+ minutes of buffer after Sunday 00:00 UTC.
	const accuracyBlocks = 75 // ~15 minutes at 12s/block
	blockNumber, err := binarySearchBlock(rpc, targetTimestamp, estimate, accuracyBlocks, *debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error finding block: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(blockNumber)
}

// sundayMidnightUTC returns the Unix timestamp of the most recent Sunday at 00:00 UTC.
func sundayMidnightUTC() uint64 {
	now := time.Now().UTC()
	daysSinceSunday := int(now.Weekday())
	sunday := now.AddDate(0, 0, -daysSinceSunday)
	sundayMidnight := time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 0, 0, 0, 0, time.UTC)
	return uint64(sundayMidnight.Unix())
}

// estimateBlock uses linear extrapolation to estimate the block number at a target timestamp.
func estimateBlock(refBlock, refTimestamp, blockTime, targetTimestamp uint64) uint64 {
	if targetTimestamp >= refTimestamp {
		secondsDiff := targetTimestamp - refTimestamp
		blocksDiff := secondsDiff / blockTime
		return refBlock + blocksDiff
	}
	secondsDiff := refTimestamp - targetTimestamp
	blocksDiff := secondsDiff / blockTime
	return refBlock - blocksDiff
}

// rpcClient is a reusable HTTP client for RPC calls
var rpcClient = &http.Client{Timeout: 30 * time.Second}

// jsonRPCRequest represents a JSON-RPC request
type jsonRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// jsonRPCResponse represents a JSON-RPC response
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// rpcCall makes a JSON-RPC call and returns the result
func rpcCall(rpcURL, method string, params []interface{}) (json.RawMessage, error) {
	reqBody, err := json.Marshal(jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := rpcClient.Post(rpcURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("RPC request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

// hexToUint64 converts a hex string (with 0x prefix) to uint64
func hexToUint64(hex string) (uint64, error) {
	// Remove 0x prefix if present
	if len(hex) >= 2 && hex[:2] == "0x" {
		hex = hex[2:]
	}
	var result uint64
	for _, c := range hex {
		result *= 16
		switch {
		case c >= '0' && c <= '9':
			result += uint64(c - '0')
		case c >= 'a' && c <= 'f':
			result += uint64(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			result += uint64(c - 'A' + 10)
		default:
			return 0, fmt.Errorf("invalid hex character: %c", c)
		}
	}
	return result, nil
}

// getBlockTimestamp fetches the timestamp of a block via JSON-RPC.
func getBlockTimestamp(rpcURL string, blockNumber uint64) (uint64, error) {
	blockHex := fmt.Sprintf("0x%x", blockNumber)
	result, err := rpcCall(rpcURL, "eth_getBlockByNumber", []interface{}{blockHex, false})
	if err != nil {
		return 0, err
	}

	var block struct {
		Timestamp string `json:"timestamp"`
	}
	if err := json.Unmarshal(result, &block); err != nil {
		return 0, fmt.Errorf("failed to unmarshal block: %w", err)
	}

	return hexToUint64(block.Timestamp)
}

// getLatestBlockNumber fetches the latest block number via JSON-RPC.
func getLatestBlockNumber(rpcURL string) (uint64, error) {
	result, err := rpcCall(rpcURL, "eth_blockNumber", []interface{}{})
	if err != nil {
		return 0, err
	}

	var hexNum string
	if err := json.Unmarshal(result, &hexNum); err != nil {
		return 0, fmt.Errorf("failed to unmarshal block number: %w", err)
	}

	return hexToUint64(hexNum)
}

// getChainID fetches the chain ID via JSON-RPC.
func getChainID(rpcURL string) (uint64, error) {
	result, err := rpcCall(rpcURL, "eth_chainId", []interface{}{})
	if err != nil {
		return 0, err
	}

	var hexNum string
	if err := json.Unmarshal(result, &hexNum); err != nil {
		return 0, fmt.Errorf("failed to unmarshal chain ID: %w", err)
	}

	return hexToUint64(hexNum)
}

// binarySearchBlock finds a block number with timestamp close to targetTimestamp.
// The accuracyBlocks parameter controls how precise the search is - once the search
// range is within this many blocks, it stops and returns the lower bound.
// Set accuracyBlocks to 0 for exact search (finds highest block with timestamp <= target).
func binarySearchBlock(rpcURL string, targetTimestamp, estimate uint64, accuracyBlocks uint64, debug bool) (uint64, error) {
	rpcCalls := 0
	start := time.Now()

	// Get the latest block to cap our search range
	latestBlock, err := getLatestBlockNumber(rpcURL)
	rpcCalls++
	if debug {
		fmt.Fprintf(os.Stderr, "[debug] getLatestBlockNumber took %v (call #%d)\n", time.Since(start), rpcCalls)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get latest block: %w", err)
	}

	// Start with a range around the estimate
	// Use a buffer of ~1 day worth of blocks (7200 blocks at 12s each)
	buffer := uint64(7200)

	// If estimate is beyond latest block, adjust to be within valid range
	if estimate > latestBlock {
		estimate = latestBlock
	}

	low := estimate - buffer
	high := estimate + buffer

	// Cap high at the latest block
	if high > latestBlock {
		high = latestBlock
	}

	// Ensure low doesn't underflow
	if low > estimate {
		low = 1
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[debug] Initial range: low=%d, high=%d, estimate=%d\n", low, high, estimate)
	}

	// First, verify our range is valid by checking the bounds
	callStart := time.Now()
	lowTs, err := getBlockTimestamp(rpcURL, low)
	rpcCalls++
	if debug {
		fmt.Fprintf(os.Stderr, "[debug] getBlockTimestamp(low=%d) took %v (call #%d)\n", low, time.Since(callStart), rpcCalls)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get low bound timestamp: %w", err)
	}

	callStart = time.Now()
	highTs, err := getBlockTimestamp(rpcURL, high)
	rpcCalls++
	if debug {
		fmt.Fprintf(os.Stderr, "[debug] getBlockTimestamp(high=%d) took %v (call #%d)\n", high, time.Since(callStart), rpcCalls)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get high bound timestamp: %w", err)
	}

	// Expand low bound if needed (target is before our range)
	for lowTs > targetTimestamp {
		low -= buffer
		if low < 1 {
			low = 1
		}
		callStart = time.Now()
		lowTs, err = getBlockTimestamp(rpcURL, low)
		rpcCalls++
		if debug {
			fmt.Fprintf(os.Stderr, "[debug] expand low getBlockTimestamp(%d) took %v (call #%d)\n", low, time.Since(callStart), rpcCalls)
		}
		if err != nil {
			return 0, fmt.Errorf("failed to expand low bound: %w", err)
		}
	}

	// Expand high bound if needed (target is after our range)
	// But cap at latest block
	for highTs < targetTimestamp && high < latestBlock {
		high += buffer
		if high > latestBlock {
			high = latestBlock
		}
		callStart = time.Now()
		highTs, err = getBlockTimestamp(rpcURL, high)
		rpcCalls++
		if debug {
			fmt.Fprintf(os.Stderr, "[debug] expand high getBlockTimestamp(%d) took %v (call #%d)\n", high, time.Since(callStart), rpcCalls)
		}
		if err != nil {
			return 0, fmt.Errorf("failed to expand high bound: %w", err)
		}
	}

	// Binary search until we're within the accuracy threshold
	iteration := 0
	for high-low > accuracyBlocks {
		iteration++
		mid := (low + high + 1) / 2
		callStart = time.Now()
		midTs, err := getBlockTimestamp(rpcURL, mid)
		rpcCalls++
		if debug {
			fmt.Fprintf(os.Stderr, "[debug] binary search iter %d: getBlockTimestamp(%d) took %v, range=%d (call #%d)\n", iteration, mid, time.Since(callStart), high-low, rpcCalls)
		}
		if err != nil {
			return 0, fmt.Errorf("failed to get mid timestamp: %w", err)
		}

		if midTs <= targetTimestamp {
			low = mid
		} else {
			high = mid - 1
		}
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[debug] Total: %d RPC calls, %v elapsed, final range: %d blocks\n", rpcCalls, time.Since(start), high-low)
	}

	return low, nil
}
