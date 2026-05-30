package main

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-glamsterdam/glamsterdamtest"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

const prefix = "CHECK_GLAMSTERDAM"

var (
	EndpointL1 = &cli.StringFlag{
		Name:     "l1",
		Usage:    "L1 execution RPC endpoint",
		EnvVars:  op_service.PrefixEnvVar(prefix, "L1"),
		Required: true,
	}
	EndpointL2 = &cli.StringFlag{
		Name:    "l2",
		Usage:   "L2 execution RPC endpoint",
		EnvVars: op_service.PrefixEnvVar(prefix, "L2"),
		Value:   "http://localhost:9545",
	}
	EndpointRollup = &cli.StringFlag{
		Name:     "rollup-rpc",
		Usage:    "op-node (rollup) RPC endpoint",
		EnvVars:  op_service.PrefixEnvVar(prefix, "ROLLUP_RPC"),
		Required: true,
	}
	AccountKey = &cli.StringFlag{
		Name:    "account",
		Usage:   "Hex-encoded L2 private key. Required when --spam is set; ignored otherwise.",
		EnvVars: op_service.PrefixEnvVar(prefix, "ACCOUNT"),
	}
	PollInterval = &cli.DurationFlag{
		Name:    "poll-interval",
		Usage:   "How often to poll the L1 and rollup-node RPCs",
		EnvVars: op_service.PrefixEnvVar(prefix, "POLL_INTERVAL"),
		Value:   2 * time.Second,
	}
	GasThreshold = &cli.Uint64Flag{
		Name:    "gas-threshold",
		Usage:   "Minimum gas-used the post-Amsterdam safe block must carry. 0 = skip the check.",
		EnvVars: op_service.PrefixEnvVar(prefix, "GAS_THRESHOLD"),
		Value:   0,
	}
	Spam = &cli.BoolFlag{
		Name:    "spam",
		Usage:   "Spam L2 transactions while waiting for the safe head to advance. Requires --account.",
		EnvVars: op_service.PrefixEnvVar(prefix, "SPAM"),
		Value:   false,
	}
	SpamRPS = &cli.Float64Flag{
		Name:    "spam-rps",
		Usage:   "Spammer transactions-per-second when --spam is set.",
		EnvVars: op_service.PrefixEnvVar(prefix, "SPAM_RPS"),
		Value:   50,
	}
	SpamGasLimit = &cli.Uint64Flag{
		Name:    "spam-gas",
		Usage:   "Per-tx gas limit for the spammer.",
		EnvVars: op_service.PrefixEnvVar(prefix, "SPAM_GAS"),
		Value:   50_000,
	}
)

func commonFlags() []cli.Flag {
	flags := []cli.Flag{EndpointL1, EndpointL2, EndpointRollup, PollInterval}
	return append(flags, oplog.CLIFlags(prefix)...)
}

type checkEnv struct {
	ctx     context.Context
	logger  log.Logger
	l1      *ethclient.Client
	l2      *ethclient.Client
	rollup  *sources.RollupClient
	rpcConn client.RPC // kept alive so it isn't GC'd
	poll    time.Duration
}

func (e *checkEnv) close() {
	if e.l1 != nil {
		e.l1.Close()
	}
	if e.l2 != nil {
		e.l2.Close()
	}
	if e.rpcConn != nil {
		e.rpcConn.Close()
	}
}

func resolveEnv(c *cli.Context) (*checkEnv, error) {
	logCfg := oplog.ReadCLIConfig(c)
	logger := oplog.NewLogger(c.App.Writer, logCfg)
	c.Context = ctxinterrupt.WithCancelOnInterrupt(c.Context)

	l1Cl, err := ethclient.DialContext(c.Context, c.String(EndpointL1.Name))
	if err != nil {
		return nil, fmt.Errorf("dial L1: %w", err)
	}
	l2Cl, err := ethclient.DialContext(c.Context, c.String(EndpointL2.Name))
	if err != nil {
		l1Cl.Close()
		return nil, fmt.Errorf("dial L2: %w", err)
	}
	rpcCl, err := client.NewRPC(c.Context, logger, c.String(EndpointRollup.Name))
	if err != nil {
		l1Cl.Close()
		l2Cl.Close()
		return nil, fmt.Errorf("dial rollup-rpc: %w", err)
	}
	return &checkEnv{
		ctx:     c.Context,
		logger:  logger,
		l1:      l1Cl,
		l2:      l2Cl,
		rollup:  sources.NewRollupClient(rpcCl),
		rpcConn: rpcCl,
		poll:    c.Duration(PollInterval.Name),
	}, nil
}

func parseAccount(c *cli.Context) (*ecdsa.PrivateKey, error) {
	raw := c.String(AccountKey.Name)
	if raw == "" {
		return nil, errors.New("--account is required when --spam is set")
	}
	key, err := crypto.HexToECDSA(strings.TrimPrefix(raw, "0x"))
	if err != nil {
		return nil, fmt.Errorf("parse --account: %w", err)
	}
	return key, nil
}

func l1HeaderFn(l1 *ethclient.Client) glamsterdamtest.LatestL1Header {
	return func(ctx context.Context) (*types.Header, error) {
		return l1.HeaderByNumber(ctx, nil)
	}
}

func l2GasUsedFn(l2 *ethclient.Client) glamsterdamtest.L2BlockGasUsed {
	return func(ctx context.Context, hash common.Hash) (uint64, error) {
		blk, err := l2.BlockByHash(ctx, hash)
		if err != nil {
			return 0, err
		}
		return blk.GasUsed(), nil
	}
}

func runWaitAmsterdam(c *cli.Context) error {
	env, err := resolveEnv(c)
	if err != nil {
		return err
	}
	defer env.close()
	if _, err := glamsterdamtest.WaitForAmsterdamOnL1(env.ctx, env.logger, l1HeaderFn(env.l1), env.poll); err != nil {
		return fmt.Errorf("wait-amsterdam: %w", err)
	}
	return nil
}

func runSafeHead(c *cli.Context) error {
	env, err := resolveEnv(c)
	if err != nil {
		return err
	}
	defer env.close()

	var stopSpam func()
	if c.Bool(Spam.Name) {
		key, err := parseAccount(c)
		if err != nil {
			return err
		}
		stopSpam, err = glamsterdamtest.SpamL2(env.ctx, env.logger, env.l2, key, c.Float64(SpamRPS.Name), c.Uint64(SpamGasLimit.Name))
		if err != nil {
			return fmt.Errorf("start spammer: %w", err)
		}
		defer stopSpam()
	}

	amsterdamHdr, err := glamsterdamtest.WaitForAmsterdamOnL1(env.ctx, env.logger, l1HeaderFn(env.l1), env.poll)
	if err != nil {
		return fmt.Errorf("wait-amsterdam: %w", err)
	}
	safe, err := glamsterdamtest.WaitForSafeHeadPastL1(env.ctx, env.logger, env.rollup, amsterdamHdr.Number.Uint64(), env.poll)
	if err != nil {
		return fmt.Errorf("wait safe head: %w", err)
	}

	threshold := c.Uint64(GasThreshold.Name)
	if threshold > 0 {
		if err := glamsterdamtest.CheckSafeHeadTraffic(env.ctx, env.logger, l2GasUsedFn(env.l2), safe.Hash, threshold); err != nil {
			return fmt.Errorf("traffic check: %w", err)
		}
	}
	env.logger.Info("check-glamsterdam OK",
		"amsterdamL1", amsterdamHdr.Number.Uint64(),
		"safeL2", safe.Number,
		"safeL2L1Origin", safe.L1Origin.Number,
	)
	return nil
}

func makeWaitAmsterdamCommand() *cli.Command {
	return &cli.Command{
		Name:   "wait-amsterdam",
		Usage:  "Block until the L1 produces a header with a non-nil SlotNumber.",
		Flags:  cliapp.ProtectFlags(commonFlags()),
		Action: runWaitAmsterdam,
	}
}

func makeSafeHeadCommand() *cli.Command {
	flags := append(commonFlags(), AccountKey, GasThreshold, Spam, SpamRPS, SpamGasLimit)
	return &cli.Command{
		Name:   "safe-head",
		Usage:  "Wait for Amsterdam on L1, then for the L2 safe head's L1 origin to reach that block.",
		Flags:  cliapp.ProtectFlags(flags),
		Action: runSafeHead,
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "check-glamsterdam"
	app.Usage = "Check Glamsterdam (Amsterdam) post-fork conformance against an external OP Stack network."
	app.Description = "Runs the same conformance loops as TestSafeHeadAdvancesAfterGlamsterdam, but against any reachable L1 / L2 / rollup-rpc endpoint."
	app.Action = func(c *cli.Context) error { return errors.New("see sub-commands") }
	app.Writer = os.Stdout
	app.ErrWriter = os.Stderr
	app.Commands = []*cli.Command{
		makeWaitAmsterdamCommand(),
		makeSafeHeadCommand(),
	}
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "check-glamsterdam failed: %v\n", err)
		os.Exit(1)
	}
}
