package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/log"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
)

func main() {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Flags = cliapp.ProtectFlags(Flags)
	app.Version = "dev"
	app.Name = "oracle"
	app.Usage = "Fault Proof Oracle util app"
	app.Description = "Oracle client/server util: this allows to run and inspect host/client interactions, through the standard pre-image oracle ABI."
	app.Action = cliapp.LifecycleCmd(Main)

	ctx := opio.WithInterruptBlocker(context.Background())
	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}

func Main(cliCtx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	cfg := ReadCLIConfig(cliCtx)
	if err := cfg.Check(); err != nil {
		return nil, fmt.Errorf("invalid CLI flags: %w", err)
	}

	l := oplog.NewLogger(oplog.AppOut(cliCtx), cfg.LogCfg)
	oplog.SetGlobalLogHandler(l.GetHandler())
	opservice.ValidateEnvVars(EnvVarPrefix, Flags, l)

	return NewOracleService(l, &cfg.RunConfig, closeApp), nil
}
