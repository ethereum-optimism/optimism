package main

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-review-period/reviewperiod"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
)

var (
	prefix     = "CHECK_REVIEW_PERIOD"
	EndpointL1 = &cli.StringFlag{
		Name:     "l1",
		Usage:    "L1 execution RPC endpoint. It must serve state at the block before the upgrade.",
		EnvVars:  op_service.PrefixEnvVar(prefix, "L1"),
		Required: true,
	}
	SystemConfigs = &cli.StringSliceFlag{
		Name:     "system-config",
		Usage:    "SystemConfig proxy address of a chain to check (repeatable)",
		EnvVars:  op_service.PrefixEnvVar(prefix, "SYSTEM_CONFIG"),
		Required: true,
	}
	UpgradeTx = &cli.StringFlag{
		Name:    "upgrade-tx",
		Usage:   "Hash of the upgrade transaction. The delays are compared at the block before it and the latest block.",
		EnvVars: op_service.PrefixEnvVar(prefix, "UPGRADE_TX"),
	}
	BeforeBlock = &cli.Uint64Flag{
		Name:    "before-block",
		Usage:   "L1 block before the upgrade, used instead of --upgrade-tx",
		EnvVars: op_service.PrefixEnvVar(prefix, "BEFORE_BLOCK"),
	}
)

func main() {
	app := cli.NewApp()
	app.Name = "check-review-period"
	app.Usage = "Check that an upgrade leaves the review period delays unchanged."
	app.Description = "Compare the proof maturity, dispute game finality and DelayedWETH withdrawal delays " +
		"before and after an upgrade."
	app.Action = func(c *cli.Context) error {
		return errors.New("see sub-commands")
	}
	app.Writer = os.Stdout
	app.ErrWriter = os.Stderr
	app.Commands = []*cli.Command{
		{
			Name:   "check",
			Usage:  "Compare the delays of each chain before and after an upgrade",
			Flags:  cliapp.ProtectFlags(append([]cli.Flag{EndpointL1, SystemConfigs, UpgradeTx, BeforeBlock}, logcli.CLIFlags(prefix)...)),
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

func checkAction(c *cli.Context) error {
	logger := logcli.NewLogger(c.App.Writer, logcli.ReadCLIConfig(c))
	c.Context = ctxinterrupt.WithCancelOnInterrupt(c.Context)
	client, err := ethclient.DialContext(c.Context, c.String(EndpointL1.Name))
	if err != nil {
		return fmt.Errorf("failed to dial L1 RPC: %w", err)
	}
	defer client.Close()
	l1 := l1Caller{client}

	before, receipt, err := beforeBlock(c, l1)
	if err != nil {
		return err
	}
	after, err := l1.BlockNumber(c.Context)
	if err != nil {
		return fmt.Errorf("failed to get L1 block number: %w", err)
	}
	if after <= before {
		return fmt.Errorf("latest L1 block %d is not after the before block %d", after, before)
	}
	logger.Info("Comparing review period delays", "beforeBlock", before, "afterBlock", after)

	failures := 0
	for _, addr := range c.StringSlice(SystemConfigs.Name) {
		if !common.IsHexAddress(addr) {
			return fmt.Errorf("invalid SystemConfig address %q", addr)
		}
		systemConfig := common.HexToAddress(addr)
		pre, err := reviewperiod.ReadChainDelays(c.Context, l1, systemConfig, rpc.BlockNumber(before))
		if err != nil {
			return fmt.Errorf("failed to read the delays at block %d (the RPC must serve state at that block): %w", before, err)
		}
		post, err := reviewperiod.ReadChainDelays(c.Context, l1, systemConfig, rpc.BlockNumber(after))
		if err != nil {
			return fmt.Errorf("failed to read the delays at block %d: %w", after, err)
		}
		upgrades, err := upgradedProxies(c.Context, client, receipt, pre, before, after)
		if err != nil {
			return err
		}
		failures += compare(logger, pre, post, upgrades)
	}
	if failures > 0 {
		return fmt.Errorf("%d check(s) failed between L1 blocks %d and %d", failures, before, after)
	}
	logger.Info("All delays unchanged", "chains", len(c.StringSlice(SystemConfigs.Name)))
	return nil
}

// beforeBlock returns the block before the upgrade, from --before-block or the block of --upgrade-tx.
// The receipt is nil when --before-block is used.
func beforeBlock(c *cli.Context, l1 l1Caller) (uint64, *types.Receipt, error) {
	hasTx, hasBlock := c.IsSet(UpgradeTx.Name), c.IsSet(BeforeBlock.Name)
	if hasTx == hasBlock {
		return 0, nil, errors.New("set exactly one of --upgrade-tx and --before-block")
	}
	if hasBlock {
		return c.Uint64(BeforeBlock.Name), nil, nil
	}
	receipt, err := l1.TransactionReceipt(c.Context, common.HexToHash(c.String(UpgradeTx.Name)))
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get the upgrade transaction receipt: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return 0, nil, fmt.Errorf("upgrade transaction %s failed", receipt.TxHash)
	}
	return bigs.Uint64Strict(receipt.BlockNumber) - 1, receipt, nil
}

// upgradedProxies counts the Upgraded events emitted by the chain's proxies, as they were before the
// upgrade, in the upgrade transaction or, without one, in the blocks after the before block.
func upgradedProxies(ctx context.Context, client *ethclient.Client, receipt *types.Receipt, pre reviewperiod.ChainDelays, before, after uint64) (map[string]int, error) {
	if receipt != nil {
		logs := make([]types.Log, len(receipt.Logs))
		for i, l := range receipt.Logs {
			logs[i] = *l
		}
		return reviewperiod.CountUpgrades(pre, logs), nil
	}
	logs, err := reviewperiod.UpgradedLogs(ctx, client, pre, before+1, after)
	if err != nil {
		return nil, err
	}
	return reviewperiod.CountUpgrades(pre, logs), nil
}

// compare logs the result for one chain and returns the number of failed checks.
func compare(logger log.Logger, pre, post reviewperiod.ChainDelays, upgrades map[string]int) int {
	l := logger.New("chainID", post.L2ChainID, "systemConfig", post.SystemConfig)
	failures := 0

	if len(upgrades) == 0 {
		failures++
		l.Error("No proxy of the chain was upgraded between the two blocks")
	} else {
		l.Info("Upgraded", "proxies", upgrades, "fromOPCMVersion", pre.LastUsedOPCMVersion, "toOPCMVersion", post.LastUsedOPCMVersion)
	}

	for _, d := range reviewperiod.CompareDelays(pre, post) {
		if d.Changed() {
			failures++
			l.Error("Delay changed", "delay", d.Name, "before", d.Before, "after", d.After)
		} else {
			l.Info("Delay unchanged", "delay", d.Name, "value", d.After)
		}
	}

	for _, a := range []struct {
		name      string
		pre, post common.Address
	}{
		{"optimismPortal", pre.OptimismPortal, post.OptimismPortal},
		{"anchorStateRegistry", pre.AnchorStateRegistry, post.AnchorStateRegistry},
		{"delayedWETH", pre.DelayedWETH, post.DelayedWETH},
	} {
		if a.pre != a.post {
			l.Warn("Contract address changed", "contract", a.name, "before", a.pre, "after", a.post)
		}
	}
	return failures
}
