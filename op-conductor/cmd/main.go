package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-conductor/conductor"
	"github.com/ethereum-optimism/optimism/op-conductor/flags"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
)

var (
	Version   = "v0.0.1"
	GitCommit = ""
	GitDate   = ""
)

func main() {
	logcli.SetupDefaults()

	app := cli.NewApp()
	app.Flags = cliapp.ProtectFlags(flags.Flags)
	app.Version = opservice.FormatVersion(Version, GitCommit, GitDate, "")
	app.Name = "op-conductor"
	app.Usage = "Optimism Sequencer Conductor Service"
	app.Description = "op-conductor help sequencer to run in highly available mode"
	app.Action = cliapp.LifecycleCmd(OpConductorMain)
	app.Commands = []*cli.Command{}

	ctx := ctxinterrupt.WithSignalWaiterMain(context.Background())
	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}

func OpConductorMain(ctx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	logCfg := logcli.ReadCLIConfig(ctx)
	log := logcli.NewLogger(logcli.AppOut(ctx), logCfg)
	logcli.SetGlobalLogHandler(log.Handler())
	opservice.ValidateEnvVars(flags.EnvVarPrefix, flags.Flags, log)

	cfg, err := conductor.NewConfig(ctx, log)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	c, err := conductor.New(ctx.Context, cfg, log, Version)
	if err != nil {
		return nil, fmt.Errorf("failed to create conductor: %w", err)
	}

	return c, nil
}
