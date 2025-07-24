package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"github.com/HashKeyChain/verse/op-conductor/conductor"
	"github.com/HashKeyChain/verse/op-conductor/flags"
	opservice "github.com/HashKeyChain/verse/op-service"
	"github.com/HashKeyChain/verse/op-service/cliapp"
	"github.com/HashKeyChain/verse/op-service/ctxinterrupt"
	oplog "github.com/HashKeyChain/verse/op-service/log"
)

var (
	Version   = "v0.0.1"
	GitCommit = ""
	GitDate   = ""
)

func main() {
	oplog.SetupDefaults()

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
	logCfg := oplog.ReadCLIConfig(ctx)
	log := oplog.NewLogger(oplog.AppOut(ctx), logCfg)
	oplog.SetGlobalLogHandler(log.Handler())
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
