package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Reference points for initial block estimation.
// These provide a starting point for binary search.
const (
	// Mainnet reference: Block 23964895, mined at timestamp 1765158191 (Dec 7, 2025)
	mainnetRefBlock     uint64 = 23964895
	mainnetRefTimestamp uint64 = 1765158191
	mainnetBlockTime    uint64 = 12

	// Sepolia reference: Block 9819372, mined at timestamp 1765482912 (Dec 11, 2025)
	sepoliaRefBlock     uint64 = 9819372
	sepoliaRefTimestamp uint64 = 1765482912
	sepoliaBlockTime    uint64 = 12
)

// Default RPC URLs
const (
	defaultMainnetRPC = "https://eth.llamarpc.com"
	defaultSepoliaRPC = "https://ethereum-sepolia.publicnode.com"
)

func main() {
	chain := flag.String("chain", "mainnet", "Chain to calculate block for (mainnet or sepolia)")
	rpcURL := flag.String("rpc-url", "", "RPC URL (defaults to ETH_RPC_URL env var, then public RPC)")
	flag.Parse()

	var refBlock, refTimestamp, blockTime uint64
	var defaultRPC string

	switch *chain {
	case "mainnet":
		refBlock = mainnetRefBlock
		refTimestamp = mainnetRefTimestamp
		blockTime = mainnetBlockTime
		defaultRPC = defaultMainnetRPC
	case "sepolia":
		refBlock = sepoliaRefBlock
		refTimestamp = sepoliaRefTimestamp
		blockTime = sepoliaBlockTime
		defaultRPC = defaultSepoliaRPC
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

	targetTimestamp := sundayMidnightUTC()

	// Get initial estimate using linear extrapolation
	estimate := estimateBlock(refBlock, refTimestamp, blockTime, targetTimestamp)

	// Binary search to find the exact block
	blockNumber, err := binarySearchBlock(rpc, targetTimestamp, estimate)
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

// getBlockTimestamp fetches the timestamp of a block using cast.
func getBlockTimestamp(rpcURL string, blockNumber uint64) (uint64, error) {
	cmd := exec.Command("cast", "block", strconv.FormatUint(blockNumber, 10), "--rpc-url", rpcURL, "-f", "timestamp")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("cast block failed: %w", err)
	}
	timestamp, err := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse timestamp: %w", err)
	}
	return timestamp, nil
}

// getLatestBlockNumber fetches the latest block number using cast.
func getLatestBlockNumber(rpcURL string) (uint64, error) {
	cmd := exec.Command("cast", "block-number", "--rpc-url", rpcURL)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("cast block-number failed: %w", err)
	}
	blockNum, err := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse block number: %w", err)
	}
	return blockNum, nil
}

// binarySearchBlock finds the highest block number with timestamp <= targetTimestamp.
func binarySearchBlock(rpcURL string, targetTimestamp, estimate uint64) (uint64, error) {
	// Get the latest block to cap our search range
	latestBlock, err := getLatestBlockNumber(rpcURL)
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

	// First, verify our range is valid by checking the bounds
	lowTs, err := getBlockTimestamp(rpcURL, low)
	if err != nil {
		return 0, fmt.Errorf("failed to get low bound timestamp: %w", err)
	}

	highTs, err := getBlockTimestamp(rpcURL, high)
	if err != nil {
		return 0, fmt.Errorf("failed to get high bound timestamp: %w", err)
	}

	// Expand low bound if needed (target is before our range)
	for lowTs > targetTimestamp {
		low -= buffer
		if low < 1 {
			low = 1
		}
		lowTs, err = getBlockTimestamp(rpcURL, low)
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
		highTs, err = getBlockTimestamp(rpcURL, high)
		if err != nil {
			return 0, fmt.Errorf("failed to expand high bound: %w", err)
		}
	}

	// Binary search for the exact block
	for low < high {
		mid := (low + high + 1) / 2
		midTs, err := getBlockTimestamp(rpcURL, mid)
		if err != nil {
			return 0, fmt.Errorf("failed to get mid timestamp: %w", err)
		}

		if midTs <= targetTimestamp {
			low = mid
		} else {
			high = mid - 1
		}
	}

	return low, nil
}
