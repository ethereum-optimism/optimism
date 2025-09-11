package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/url"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/ethclient"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
)

func main() {
	// go run main.go --el-rpc http://localhost:54101 --cl-rpc http://localhost:8547 --target 100
	app := &cli.App{
		Name:  "signaller",
		Usage: "Fetch a block from EL and post it as an unsafe payload to CL",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "el-rpc",
				Usage:    "EL RPC endpoint",
				Required: true,
				EnvVars:  []string{"EL_RPC"},
			},
			&cli.StringFlag{
				Name:     "cl-rpc",
				Usage:    "CL RPC endpoint",
				Required: true,
				EnvVars:  []string{"CL_RPC"},
			},
			&cli.Uint64Flag{
				Name:     "target",
				Usage:    "Target block number to fetch from EL",
				Required: true,
				EnvVars:  []string{"TARGET_BLOCK"},
			},
			&cli.DurationFlag{
				Name:    "timeout",
				Usage:   "Overall request timeout",
				Value:   30 * time.Second,
				EnvVars: []string{"SIGNALLER_TIMEOUT"},
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	s := slog.New(handler)
	gethLogger := gethlog.NewLogger(handler)

	elRPC := c.String("el-rpc")
	clRPC := c.String("cl-rpc")
	target := c.Uint64("target")
	timeout := c.Duration("timeout")

	if err := validateURL(elRPC); err != nil {
		return fmt.Errorf("invalid --el-rpc %q: %w", elRPC, err)
	}
	if err := validateURL(clRPC); err != nil {
		return fmt.Errorf("invalid --cl-rpc %q: %w", clRPC, err)
	}

	ctx, cancel := context.WithTimeout(c.Context, timeout)
	defer cancel()

	s.Info("Starting signaller",
		"el_rpc", elRPC,
		"cl_rpc", clRPC,
		"target_block", target,
		"timeout", timeout)

	elClient, err := ethclient.DialContext(ctx, elRPC)
	if err != nil {
		s.Error("Failed to connect to EL", "err", err)
		return fmt.Errorf("failed to connect EL: %w", err)
	}
	defer elClient.Close()

	block, err := elClient.BlockByNumber(ctx, new(big.Int).SetUint64(target))
	if err != nil {
		s.Error("Failed to fetch block from EL", "block", target, "err", err)
		return fmt.Errorf("failed to fetch block %d: %w", target, err)
	}
	s.Info("Fetched block from EL",
		"number", block.Number().Uint64(),
		"hash", block.Hash().Hex(),
		"txs", len(block.Transactions()))

	cfg := &params.ChainConfig{}
	cfg.CanyonTime = new(uint64)  // zeroed
	cfg.IsthmusTime = new(uint64) // zeroed

	payloadEnv, err := eth.BlockAsPayloadEnv(block, cfg)
	if err != nil {
		s.Error("fFiled to convert block to payload env", "err", err)
		return fmt.Errorf("block->payload: %w", err)
	}

	rollupClient, err := dial.DialRollupClientWithTimeout(ctx, gethLogger, clRPC)
	if err != nil {
		s.Error("Failed to connect to CL (rollup)", "err", err)
		return fmt.Errorf("connect CL: %w", err)
	}
	defer rollupClient.Close()

	if err := rollupClient.PostUnsafePayload(ctx, payloadEnv); err != nil {
		s.Error("Failed to post unsafe payload", "err", err)
		return fmt.Errorf("post unsafe payload: %w", err)
	}
	s.Info("Successfully posted unsafe payload",
		"block", block.Number().Uint64(), "hash", block.Hash().Hex())

	return nil
}

func validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme == "" || u.Host == "" {
		return errors.New("must be an absolute URL with scheme and host")
	}
	return nil
}
