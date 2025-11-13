package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

func main() {
	app := cli.NewApp()
	app.Name = "check-jovian"
	app.Usage = "Check Jovian upgrade results."
	app.Description = "Check Jovian upgrade results."
	app.Action = func(c *cli.Context) error {
		return errors.New("see sub-commands")
	}
	app.Writer = os.Stdout
	app.ErrWriter = os.Stderr
	app.Commands = []*cli.Command{
		{
			Name: "contracts",
			Subcommands: []*cli.Command{
				makeCommand("gpo", checkGPO),
				makeCommand("l1block", checkL1Block),
			},
		},
		makeCommand("block-header", checkBlockHeader),
		makeCommand("all", checkAll),
	}

	err := app.Run(os.Args)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Application failed: %v\n", err)
		os.Exit(1)
	}
}

type actionEnv struct {
	log log.Logger
	l2  *ethclient.Client
}

type CheckAction func(ctx context.Context, env *actionEnv) error

var (
	prefix     = "CHECK_JOVIAN"
	EndpointL2 = &cli.StringFlag{
		Name:    "l2",
		Usage:   "L2 execution RPC endpoint",
		EnvVars: op_service.PrefixEnvVar(prefix, "L2"),
		Value:   "http://localhost:9545",
	}
)

func makeFlags() []cli.Flag {
	flags := []cli.Flag{
		EndpointL2,
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
		if err := fn(c.Context, &actionEnv{
			log: logger,
			l2:  l2Cl,
		}); err != nil {
			return fmt.Errorf("command error: %w", err)
		}
		return nil
	}
}

// checkGPO checks that GasPriceOracle.isJovian returns true
func checkGPO(ctx context.Context, env *actionEnv) error {
	cl, err := bindings.NewGasPriceOracle(predeploys.GasPriceOracleAddr, env.l2)
	if err != nil {
		return fmt.Errorf("failed to create bindings around GasPriceOracle contract: %w", err)
	}
	isJovian, err := cl.IsJovian(nil)
	if err != nil {
		return fmt.Errorf("failed to get jovian status: %w", err)
	}
	if !isJovian {
		return fmt.Errorf("GPO is not set to jovian")
	}
	env.log.Info("GasPriceOracle test: success", "isJovian", isJovian)
	return nil
}

// checkL1Block checks that L1Block.DAFootprintGasScalar returns a number
func checkL1Block(ctx context.Context, env *actionEnv) error {
	cl, err := bindings.NewL1Block(predeploys.L1BlockAddr, env.l2)
	if err != nil {
		return fmt.Errorf("failed to create bindings around L1Block contract: %w", err)
	}
	daFootprintGasScalar, err := cl.DaFootprintGasScalar(nil)
	if err != nil {
		return fmt.Errorf("failed to get DA footprint gas scalar from L1Block contract: %w", err)
	}
	if daFootprintGasScalar == 0 {
		env.log.Warn("DA footprint gas scalar is set to 0. SystemConfig needs to emit scalar change to update.")
	}
	env.log.Info("L1Block test: success", "daFootprintGasScalar", daFootprintGasScalar)
	return nil
}

// checkBlockHeader checks that the latest block header has a non-nil blobgasused field
func checkBlockHeader(ctx context.Context, env *actionEnv) error {
	latest, err := env.l2.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get latest block: %w", err)
	}
	if latest.BlobGasUsed == nil {
		return fmt.Errorf("block %d has nil BlobGasUsed field", latest.Number)
	}
	env.log.Info("Block header test: success",
		"blockNumber", latest.Number,
		"blobGasUsed", *latest.BlobGasUsed)
	return nil
}

// checkAll runs all Jovian checks
func checkAll(ctx context.Context, env *actionEnv) error {
	env.log.Info("starting Jovian checks")

	if err := checkGPO(ctx, env); err != nil {
		return fmt.Errorf("failed: GPO contract error: %w", err)
	}
	if err := checkL1Block(ctx, env); err != nil {
		return fmt.Errorf("failed: L1Block contract error: %w", err)
	}
	if err := checkBlockHeader(ctx, env); err != nil {
		return fmt.Errorf("failed: block header error: %w", err)
	}

	env.log.Info("completed all tests successfully!")

	return nil
}
