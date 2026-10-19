package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-review-period/reviewperiod"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
)

var (
	prefix     = "CHECK_REVIEW_PERIOD"
	EndpointL1 = &cli.StringFlag{
		Name:     "l1",
		Usage:    "L1 execution RPC endpoint",
		EnvVars:  op_service.PrefixEnvVar(prefix, "L1"),
		Required: true,
	}
	SystemConfigs = &cli.StringSliceFlag{
		Name:     "system-config",
		Usage:    "SystemConfig proxy address of a chain to snapshot (repeatable)",
		EnvVars:  op_service.PrefixEnvVar(prefix, "SYSTEM_CONFIG"),
		Required: true,
	}
	OutFile = &cli.StringFlag{
		Name:     "out",
		Usage:    "Path to write the snapshot to",
		EnvVars:  op_service.PrefixEnvVar(prefix, "OUT"),
		Required: true,
	}
	SnapshotFile = &cli.StringFlag{
		Name:     "snapshot",
		Usage:    "Path to a snapshot written by the snapshot command",
		EnvVars:  op_service.PrefixEnvVar(prefix, "SNAPSHOT"),
		Required: true,
	}
)

type snapshot struct {
	L1Block uint64                     `json:"l1Block"`
	Chains  []reviewperiod.ChainDelays `json:"chains"`
}

func main() {
	app := cli.NewApp()
	app.Name = "check-review-period"
	app.Usage = "Check that an upgrade leaves the review period delays unchanged."
	app.Description = "Snapshot the proof maturity, dispute game finality and DelayedWETH withdrawal delays " +
		"before an upgrade, then compare them after."
	app.Action = func(c *cli.Context) error {
		return errors.New("see sub-commands")
	}
	app.Writer = os.Stdout
	app.ErrWriter = os.Stderr
	app.Commands = []*cli.Command{
		{
			Name:   "snapshot",
			Usage:  "Record the delays of each chain",
			Flags:  cliapp.ProtectFlags(append([]cli.Flag{EndpointL1, SystemConfigs, OutFile}, logcli.CLIFlags(prefix)...)),
			Action: snapshotAction,
		},
		{
			Name:   "check",
			Usage:  "Compare the delays of each chain against a snapshot",
			Flags:  cliapp.ProtectFlags(append([]cli.Flag{EndpointL1, SnapshotFile}, logcli.CLIFlags(prefix)...)),
			Action: checkAction,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Application failed: %v\n", err)
		os.Exit(1)
	}
}

// l1Caller adapts an ethclient to reviewperiod.Caller.
type l1Caller struct {
	*ethclient.Client
}

func (c l1Caller) Call(ctx context.Context, msg ethereum.CallMsg, block rpc.BlockNumber) ([]byte, error) {
	return c.CallContract(ctx, msg, big.NewInt(block.Int64()))
}

func setup(c *cli.Context) (log.Logger, l1Caller, error) {
	logger := logcli.NewLogger(c.App.Writer, logcli.ReadCLIConfig(c))
	c.Context = ctxinterrupt.WithCancelOnInterrupt(c.Context)
	l1, err := ethclient.DialContext(c.Context, c.String(EndpointL1.Name))
	if err != nil {
		return nil, l1Caller{}, fmt.Errorf("failed to dial L1 RPC: %w", err)
	}
	return logger, l1Caller{l1}, nil
}

func snapshotAction(c *cli.Context) error {
	logger, l1, err := setup(c)
	if err != nil {
		return err
	}
	defer l1.Close()

	block, err := l1.BlockNumber(c.Context)
	if err != nil {
		return fmt.Errorf("failed to get L1 block number: %w", err)
	}
	snap := snapshot{L1Block: block}
	for _, addr := range c.StringSlice(SystemConfigs.Name) {
		if !common.IsHexAddress(addr) {
			return fmt.Errorf("invalid SystemConfig address %q", addr)
		}
		delays, err := reviewperiod.ReadChainDelays(c.Context, l1, common.HexToAddress(addr), rpc.BlockNumber(block))
		if err != nil {
			return err
		}
		logger.Info("Recorded delays", "systemConfig", delays.SystemConfig,
			"proofMaturityDelaySeconds", delays.ProofMaturityDelaySeconds,
			"disputeGameFinalityDelaySeconds", delays.DisputeGameFinalityDelaySeconds,
			"withdrawalDelaySeconds", delays.WithdrawalDelaySeconds)
		snap.Chains = append(snap.Chains, delays)
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode snapshot: %w", err)
	}
	if err := os.WriteFile(c.String(OutFile.Name), data, 0o644); err != nil {
		return fmt.Errorf("failed to write snapshot: %w", err)
	}
	logger.Info("Wrote snapshot", "path", c.String(OutFile.Name), "l1Block", block, "chains", len(snap.Chains))
	return nil
}

func checkAction(c *cli.Context) error {
	logger, l1, err := setup(c)
	if err != nil {
		return err
	}
	defer l1.Close()

	data, err := os.ReadFile(c.String(SnapshotFile.Name))
	if err != nil {
		return fmt.Errorf("failed to read snapshot: %w", err)
	}
	var snap snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return fmt.Errorf("failed to decode snapshot: %w", err)
	}

	block, err := l1.BlockNumber(c.Context)
	if err != nil {
		return fmt.Errorf("failed to get L1 block number: %w", err)
	}
	mismatches := 0
	for _, before := range snap.Chains {
		after, err := reviewperiod.ReadChainDelays(c.Context, l1, before.SystemConfig, rpc.BlockNumber(block))
		if err != nil {
			return err
		}
		for _, d := range []struct {
			name          string
			before, after uint64
		}{
			{"proofMaturityDelaySeconds", before.ProofMaturityDelaySeconds, after.ProofMaturityDelaySeconds},
			{"disputeGameFinalityDelaySeconds", before.DisputeGameFinalityDelaySeconds, after.DisputeGameFinalityDelaySeconds},
			{"withdrawalDelaySeconds", before.WithdrawalDelaySeconds, after.WithdrawalDelaySeconds},
		} {
			if d.before != d.after {
				mismatches++
				logger.Error("Delay changed", "systemConfig", before.SystemConfig, "delay", d.name, "before", d.before, "after", d.after)
			} else {
				logger.Info("Delay unchanged", "systemConfig", before.SystemConfig, "delay", d.name, "value", d.after)
			}
		}
		for _, a := range []struct {
			name          string
			before, after common.Address
		}{
			{"optimismPortal", before.OptimismPortal, after.OptimismPortal},
			{"anchorStateRegistry", before.AnchorStateRegistry, after.AnchorStateRegistry},
			{"delayedWETH", before.DelayedWETH, after.DelayedWETH},
		} {
			if a.before != a.after {
				logger.Warn("Contract address changed", "systemConfig", before.SystemConfig, "contract", a.name, "before", a.before, "after", a.after)
			}
		}
	}
	if mismatches > 0 {
		return fmt.Errorf("%d delay(s) changed since the snapshot at L1 block %d", mismatches, snap.L1Block)
	}
	logger.Info("All delays unchanged", "chains", len(snap.Chains), "snapshotL1Block", snap.L1Block, "l1Block", block)
	return nil
}
