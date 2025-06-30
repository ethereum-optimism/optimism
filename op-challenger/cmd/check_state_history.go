package main

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

var (
	HistoryDepthFlag = &cli.Uint64Flag{
		Name:    "history-depth",
		Usage:   "Number of blocks to check for state history availability (default: 1000)",
		Value:   1000,
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "HISTORY_DEPTH"),
	}
	L2RpcFlag = &cli.StringFlag{
		Name:     "l2-eth-rpc",
		Usage:    "HTTP provider URL for L2 execution engine.",
		Required: true,
		EnvVars:  opservice.PrefixEnvVar(flags.EnvVarPrefix, "L2_ETH_RPC"),
	}
	RollupRpcFlag = &cli.StringFlag{
		Name:    "rollup-rpc",
		Usage:   "HTTP provider URL for the rollup node.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "ROLLUP_RPC"),
	}
)

func CheckStateHistory(ctx *cli.Context) error {
	logger, err := setupLogging(ctx)
	if err != nil {
		return err
	}

	l1RpcUrl := ctx.String(flags.L1EthRpcFlag.Name)
	if l1RpcUrl == "" {
		return fmt.Errorf("missing %v", flags.L1EthRpcFlag.Name)
	}

	l2RpcUrl := ctx.String(L2RpcFlag.Name)
	if l2RpcUrl == "" {
		return fmt.Errorf("missing %v", L2RpcFlag.Name)
	}

	rollupRpcUrl := ctx.String(RollupRpcFlag.Name)
	historyDepth := ctx.Uint64(HistoryDepthFlag.Name)

	logger.Info("Checking state history availability",
		"l1_rpc", l1RpcUrl,
		"l2_rpc", l2RpcUrl,
		"rollup_rpc", rollupRpcUrl,
		"history_depth", historyDepth)

	// Check L1 state history
	l1Available, l1Head, l1Earliest, err := checkL1StateHistory(ctx.Context, logger, l1RpcUrl, historyDepth)
	if err != nil {
		return fmt.Errorf("failed to check L1 state history: %w", err)
	}

	// Check L2 state history
	l2Available, l2Head, l2Earliest, err := checkL2StateHistory(ctx.Context, logger, l2RpcUrl, historyDepth)
	if err != nil {
		return fmt.Errorf("failed to check L2 state history: %w", err)
	}

	// Check rollup state history if RPC provided
	var rollupAvailable bool
	var rollupHead, rollupEarliest uint64
	if rollupRpcUrl != "" {
		rollupAvailable, rollupHead, rollupEarliest, err = checkRollupStateHistory(ctx.Context, logger, rollupRpcUrl, historyDepth)
		if err != nil {
			logger.Warn("Failed to check rollup state history", "err", err)
			rollupAvailable = false
		}
	}

	// Print results
	fmt.Printf("State History Availability Check Results:\n\n")

	fmt.Printf("L1 Execution Client:\n")
	fmt.Printf("  RPC URL: %s\n", l1RpcUrl)
	fmt.Printf("  Head Block: %d\n", l1Head)
	fmt.Printf("  Earliest Available: %d\n", l1Earliest)
	fmt.Printf("  Available Depth: %d blocks\n", l1Head-l1Earliest)
	fmt.Printf("  Required Depth: %d blocks\n", historyDepth)
	if l1Available {
		fmt.Printf("  Status: ✅ SUFFICIENT\n")
	} else {
		fmt.Printf("  Status: ❌ INSUFFICIENT\n")
	}
	fmt.Printf("\n")

	fmt.Printf("L2 Execution Client:\n")
	fmt.Printf("  RPC URL: %s\n", l2RpcUrl)
	fmt.Printf("  Head Block: %d\n", l2Head)
	fmt.Printf("  Earliest Available: %d\n", l2Earliest)
	fmt.Printf("  Available Depth: %d blocks\n", l2Head-l2Earliest)
	fmt.Printf("  Required Depth: %d blocks\n", historyDepth)
	if l2Available {
		fmt.Printf("  Status: ✅ SUFFICIENT\n")
	} else {
		fmt.Printf("  Status: ❌ INSUFFICIENT\n")
	}
	fmt.Printf("\n")

	if rollupRpcUrl != "" {
		fmt.Printf("Rollup Node:\n")
		fmt.Printf("  RPC URL: %s\n", rollupRpcUrl)
		fmt.Printf("  Head Block: %d\n", rollupHead)
		fmt.Printf("  Earliest Available: %d\n", rollupEarliest)
		fmt.Printf("  Available Depth: %d blocks\n", rollupHead-rollupEarliest)
		fmt.Printf("  Required Depth: %d blocks\n", historyDepth)
		if rollupAvailable {
			fmt.Printf("  Status: ✅ SUFFICIENT\n")
		} else {
			fmt.Printf("  Status: ❌ INSUFFICIENT\n")
		}
		fmt.Printf("\n")
	}

	fmt.Printf("Overall Status: ")
	if l1Available && l2Available && (rollupRpcUrl == "" || rollupAvailable) {
		fmt.Printf("✅ SUFFICIENT - All required state history is available\n")
		return nil
	} else {
		fmt.Printf("❌ INSUFFICIENT - Some required state history is missing\n")
		return fmt.Errorf("insufficient state history available")
	}
}

func checkL1StateHistory(ctx context.Context, logger log.Logger, rpcUrl string, requiredDepth uint64) (bool, uint64, uint64, error) {
	l1Client, err := dial.DialEthClientWithTimeout(ctx, dial.DefaultDialTimeout, logger, rpcUrl)
	if err != nil {
		return false, 0, 0, fmt.Errorf("failed to dial L1: %w", err)
	}
	defer l1Client.Close()

	head, err := l1Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return false, 0, 0, fmt.Errorf("failed to get L1 head: %w", err)
	}

	headNumber := head.Number.Uint64()

	// Try to access blocks going back to find the earliest available
	earliest := headNumber
	for i := uint64(1); i <= requiredDepth && i <= headNumber; i++ {
		blockNumber := headNumber - i
		_, err := l1Client.HeaderByNumber(ctx, new(big.Int).SetUint64(blockNumber))
		if err != nil {
			// If we can't fetch this block, this is our earliest available + 1
			earliest = blockNumber + 1
			break
		}
		earliest = blockNumber
	}

	available := (headNumber - earliest) >= requiredDepth
	return available, headNumber, earliest, nil
}

func checkL2StateHistory(ctx context.Context, logger log.Logger, rpcUrl string, requiredDepth uint64) (bool, uint64, uint64, error) {
	l2Client, err := dial.DialEthClientWithTimeout(ctx, dial.DefaultDialTimeout, logger, rpcUrl)
	if err != nil {
		return false, 0, 0, fmt.Errorf("failed to dial L2: %w", err)
	}
	defer l2Client.Close()

	head, err := l2Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return false, 0, 0, fmt.Errorf("failed to get L2 head: %w", err)
	}

	headNumber := head.Number.Uint64()

	// Try to access blocks going back to find the earliest available
	earliest := headNumber
	for i := uint64(1); i <= requiredDepth && i <= headNumber; i++ {
		blockNumber := headNumber - i
		_, err := l2Client.HeaderByNumber(ctx, new(big.Int).SetUint64(blockNumber))
		if err != nil {
			// If we can't fetch this block, this is our earliest available + 1
			earliest = blockNumber + 1
			break
		}
		earliest = blockNumber
	}

	available := (headNumber - earliest) >= requiredDepth
	return available, headNumber, earliest, nil
}

func checkRollupStateHistory(ctx context.Context, logger log.Logger, rpcUrl string, requiredDepth uint64) (bool, uint64, uint64, error) {
	rollupClient, err := dial.DialRollupClientWithTimeout(ctx, dial.DefaultDialTimeout, logger, rpcUrl)
	if err != nil {
		return false, 0, 0, fmt.Errorf("failed to dial rollup: %w", err)
	}
	defer rollupClient.Close()

	// Get current sync status to find head
	syncStatus, err := rollupClient.SyncStatus(ctx)
	if err != nil {
		return false, 0, 0, fmt.Errorf("failed to get rollup sync status: %w", err)
	}

	headNumber := syncStatus.UnsafeL2.Number

	// Try to access L2 block refs going back to find the earliest available
	earliest := headNumber
	for i := uint64(1); i <= requiredDepth && i <= headNumber; i++ {
		blockNumber := headNumber - i
		_, err := rollupClient.L2BlockRefByNumber(ctx, blockNumber)
		if err != nil {
			// If we can't fetch this block ref, this is our earliest available + 1
			earliest = blockNumber + 1
			break
		}
		earliest = blockNumber
	}

	available := (headNumber - earliest) >= requiredDepth
	return available, headNumber, earliest, nil
}

func checkStateHistoryFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		flags.L1EthRpcFlag,
		L2RpcFlag,
		RollupRpcFlag,
		HistoryDepthFlag,
	}
	cliFlags = append(cliFlags, oplog.CLIFlags(flags.EnvVarPrefix)...)
	return cliFlags
}

var CheckStateHistoryCommand = &cli.Command{
	Name:        "check-state-history",
	Usage:       "Check if sufficient state history is available for the challenger to operate",
	Description: "Verifies that L1, L2, and rollup nodes have sufficient historical state data available for the challenger to access when needed for dispute games",
	Action:      Interruptible(CheckStateHistory),
	Flags:       checkStateHistoryFlags(),
}
