package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-fjord/checks"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
)

var (
	prefix     = "CHECK_FJORD"
	EndpointL2 = &cli.StringFlag{
		Name:    "l2",
		Usage:   "L2 execution RPC endpoint",
		EnvVars: op_service.PrefixEnvVar(prefix, "L2"),
		Value:   "http://localhost:9545",
	}
	AccountKey = &cli.StringFlag{
		Name:    "account",
		Usage:   "Private key (hex-formatted string) of test account to perform test txs with",
		EnvVars: op_service.PrefixEnvVar(prefix, "ACCOUNT"),
	}
)

type CheckAction func(ctx context.Context, env *checks.CheckFjordConfig) error

func makeFlags() []cli.Flag {
	flags := []cli.Flag{
		EndpointL2,
		AccountKey,
	}
	return append(flags, oplog.CLIFlags(prefix)...)
}

func makeCommand(name string, fn CheckAction) *cli.Command {
	return &cli.Command{
		Name:   name,
		Action: makeCommandAction(fn),
		Flags:  cliapp.ProtectFlags(makeFlags()),
	}
}

func makeCommandAction(fn CheckAction) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		logCfg := oplog.ReadCLIConfig(c)
		logger := oplog.NewLogger(c.App.Writer, logCfg)

		c.Context = ctxinterrupt.WithCancelOnInterrupt(c.Context)
		l2Cl, err := ethclient.DialContext(c.Context, c.String(EndpointL2.Name))
		if err != nil {
			return fmt.Errorf("failed to dial L2 RPC: %w", err)
		}
		key, err := crypto.HexToECDSA(c.String(AccountKey.Name))
		if err != nil {
			return fmt.Errorf("failed to parse test private key: %w", err)
		}
		if err := fn(c.Context, &checks.CheckFjordConfig{
			Log:  logger,
			L2:   l2Cl,
			Key:  key,
			Addr: crypto.PubkeyToAddress(key.PublicKey),
		}); err != nil {
			return fmt.Errorf("command error: %w", err)
		}
		return nil
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "check-fjord"
	app.Usage = "Check Fjord upgrade results."
	app.Description = "Check Fjord upgrade results."
	app.Action = func(c *cli.Context) error {
		return errors.New("see sub-commands")
	}
	app.Writer = os.Stdout
	app.ErrWriter = os.Stderr
	app.Commands = []*cli.Command{
		makeCommand("all", checks.CheckAll),
		makeCommand("rip-7212", checks.CheckRIP7212),
		{
			Name: "fast-lz",
			Subcommands: []*cli.Command{
				makeCommand("gas-price-oracle", checks.CheckGasPriceOracle),
				makeCommand("tx-empty", checks.CheckTxEmpty),
				makeCommand("tx-all-zero", checks.CheckTxAllZero),
				makeCommand("tx-all-42", checks.CheckTxAll42),
				makeCommand("tx-random", checks.CheckTxRandom),
				makeCommand("all", checks.CheckAllFastLz),
			},
			Flags:  makeFlags(),
			Action: makeCommandAction(checks.CheckAllFastLz),
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Application failed: %v\n", err)
		os.Exit(1)
	}
}
