package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	gn "github.com/ethereum/go-ethereum/node"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-interop/checks"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

const prefix = "CHECK_INTEROP"

var (
	ConfigFile = &cli.StringFlag{
		Name:    "config",
		Usage:   "Path to a TOML config file supplying flag values. CLI flags and env vars override it.",
		EnvVars: op_service.PrefixEnvVar(prefix, "CONFIG"),
	}
	EndpointL2A = &cli.StringFlag{
		Name:    "l2-a",
		Usage:   "L2 chain A execution RPC endpoint",
		EnvVars: op_service.PrefixEnvVar(prefix, "L2_A"),
		Value:   "http://localhost:9545",
	}
	EndpointL2B = &cli.StringFlag{
		Name:    "l2-b",
		Usage:   "L2 chain B execution RPC endpoint",
		EnvVars: op_service.PrefixEnvVar(prefix, "L2_B"),
		Value:   "http://localhost:9546",
	}
	AccountKey = &cli.StringFlag{
		Name:    "account",
		Usage:   "Private key (hex-formatted string) of a funded test account on both chains",
		EnvVars: op_service.PrefixEnvVar(prefix, "ACCOUNT"),
	}
	FilterAdminRPC = &cli.StringFlag{
		Name:    "filter.admin-rpc",
		Usage:   "op-interop-filter admin RPC endpoint (used to toggle failsafe)",
		EnvVars: op_service.PrefixEnvVar(prefix, "FILTER_ADMIN_RPC"),
	}
	FilterJWTSecret = &cli.StringFlag{
		Name:    "filter.jwt-secret",
		Usage:   "Path to the JWT secret authenticating the op-interop-filter admin RPC",
		EnvVars: op_service.PrefixEnvVar(prefix, "FILTER_JWT_SECRET"),
	}
	RelayTimeout = &cli.DurationFlag{
		Name:    "relay-timeout",
		Usage:   "Maximum time to wait for a relayed cross-chain message (and other txs) to be included",
		EnvVars: op_service.PrefixEnvVar(prefix, "RELAY_TIMEOUT"),
		Value:   2 * time.Minute,
	}
	PropagationWait = &cli.DurationFlag{
		Name:    "propagation-wait",
		Usage:   "How long to wait for an initiating message to propagate before the failsafe-blocked attempt",
		EnvVars: op_service.PrefixEnvVar(prefix, "PROPAGATION_WAIT"),
		Value:   6 * time.Second,
	}
	Iterations = &cli.IntFlag{
		Name:    "iterations",
		Usage:   "Number of A->B->A round-trips to perform",
		EnvVars: op_service.PrefixEnvVar(prefix, "ITERATIONS"),
		Value:   3,
	}
)

func baseFlags() []cli.Flag {
	return append([]cli.Flag{
		ConfigFile,
		altsrc.NewStringFlag(EndpointL2A),
		altsrc.NewStringFlag(EndpointL2B),
		altsrc.NewStringFlag(AccountKey),
		altsrc.NewDurationFlag(RelayTimeout),
	}, oplog.CLIFlags(prefix)...)
}

func roundtripFlags() []cli.Flag {
	return append(baseFlags(), altsrc.NewIntFlag(Iterations))
}

func failsafeFlags() []cli.Flag {
	return append(baseFlags(),
		altsrc.NewStringFlag(FilterAdminRPC),
		altsrc.NewStringFlag(FilterJWTSecret),
		altsrc.NewDurationFlag(PropagationWait),
	)
}

// setup reads the logging config and wires interrupt cancellation onto the context.
func setup(c *cli.Context) (log.Logger, context.Context) {
	logCfg := oplog.ReadCLIConfig(c)
	logger := oplog.NewLogger(c.App.Writer, logCfg)
	c.Context = ctxinterrupt.WithCancelOnInterrupt(c.Context)
	return logger, c.Context
}

// baseConfig dials both L2 clients, parses the test account, and verifies the
// two endpoints are distinct chains.
func baseConfig(ctx context.Context, logger log.Logger, c *cli.Context) (cfg *checks.CheckInteropConfig, err error) {
	var l2A, l2B *sources.EthClient
	defer func() {
		if err != nil {
			if l2A != nil {
				l2A.Close()
			}
			if l2B != nil {
				l2B.Close()
			}
		}
	}()

	l2A, err = dialEthClient(ctx, logger, c.String(EndpointL2A.Name))
	if err != nil {
		return nil, fmt.Errorf("failed to dial L2 chain A: %w", err)
	}
	l2B, err = dialEthClient(ctx, logger, c.String(EndpointL2B.Name))
	if err != nil {
		return nil, fmt.Errorf("failed to dial L2 chain B: %w", err)
	}
	keyHex := c.String(AccountKey.Name)
	if keyHex == "" {
		return nil, errors.New("test account private key is required: set --account, the env var, or 'account' in --config")
	}
	key, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to parse test private key: %w", err)
	}

	chainA, err := l2A.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch L2 chain A id: %w", err)
	}
	chainB, err := l2B.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch L2 chain B id: %w", err)
	}
	if chainA.Cmp(chainB) == 0 {
		return nil, fmt.Errorf("--l2-a and --l2-b must be different chains, both report chain id %s", chainA)
	}

	addr := crypto.PubkeyToAddress(key.PublicKey)
	logger.Info("running interop check", "chainA", chainA, "chainB", chainB, "account", addr)
	return &checks.CheckInteropConfig{
		Log:          logger,
		L2A:          l2A,
		L2B:          l2B,
		Key:          key,
		L2AChainID:   eth.ChainIDFromBig(chainA),
		L2BChainID:   eth.ChainIDFromBig(chainB),
		RelayTimeout: c.Duration(RelayTimeout.Name),
	}, nil
}

func dialEthClient(ctx context.Context, logger log.Logger, url string) (*sources.EthClient, error) {
	rpcCl, err := client.NewRPC(ctx, logger, url)
	if err != nil {
		return nil, err
	}
	return sources.NewEthClient(rpcCl, logger, nil, sources.DefaultEthClientConfig(10))
}

func dialFilterAdmin(ctx context.Context, logger log.Logger, url, jwtPath string) (client.RPC, error) {
	if url == "" || jwtPath == "" {
		return nil, errors.New("both --filter.admin-rpc and --filter.jwt-secret are required for the failsafe check")
	}
	secret, err := oprpc.ObtainJWTSecret(logger, jwtPath, false)
	if err != nil {
		return nil, fmt.Errorf("failed to load filter admin JWT secret: %w", err)
	}
	return client.NewRPC(ctx, logger, url,
		client.WithGethRPCOptions(gethrpc.WithHTTPAuth(gn.NewJWTAuth([32]byte(secret)))))
}

func roundTripAction(c *cli.Context) error {
	logger, ctx := setup(c)
	cfg, err := baseConfig(ctx, logger, c)
	if err != nil {
		return err
	}
	defer cfg.Close()
	return checks.CheckRoundTrip(ctx, cfg, c.Int(Iterations.Name))
}

func failsafeAction(c *cli.Context) error {
	logger, ctx := setup(c)
	cfg, err := baseConfig(ctx, logger, c)
	if err != nil {
		return err
	}
	defer cfg.Close()
	admin, err := dialFilterAdmin(ctx, logger, c.String(FilterAdminRPC.Name), c.String(FilterJWTSecret.Name))
	if err != nil {
		return err
	}
	cfg.FilterAdmin = admin
	cfg.PropagationWait = c.Duration(PropagationWait.Name)
	return checks.CheckFailsafe(ctx, cfg)
}

func main() {
	app := cli.NewApp()
	app.Name = "check-interop"
	app.Usage = "Smoke-test interop cross-chain messaging and the interop filter failsafe."
	app.Description = "Smoke-test interop cross-chain messaging and the interop filter failsafe."
	app.Action = func(c *cli.Context) error {
		return errors.New("see sub-commands")
	}
	app.Writer = os.Stdout
	app.ErrWriter = os.Stderr
	roundtripCmdFlags := cliapp.ProtectFlags(roundtripFlags())
	failsafeCmdFlags := cliapp.ProtectFlags(failsafeFlags())
	tomlSource := altsrc.NewTomlSourceFromFlagFunc(ConfigFile.Name)
	app.Commands = []*cli.Command{
		{
			Name:   "roundtrip",
			Usage:  "Send an interop message A -> B and B -> A, relaying each.",
			Flags:  roundtripCmdFlags,
			Before: altsrc.InitInputSourceWithContext(roundtripCmdFlags, tomlSource),
			Action: roundTripAction,
		},
		{
			Name:   "failsafe",
			Usage:  "Verify interop messages succeed, are blocked while failsafe is enabled, then succeed again.",
			Flags:  failsafeCmdFlags,
			Before: altsrc.InitInputSourceWithContext(failsafeCmdFlags, tomlSource),
			Action: failsafeAction,
		},
	}

	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Application failed: %v\n", err)
		os.Exit(1)
	}
}
