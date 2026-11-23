// Package app wires op-conductor into a runnable CLI application.
//
// It exists so that the standard binary and any embedding binary share the same
// flag set, config parsing and lifecycle handling, instead of each duplicating
// the wiring.
package app

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
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
)

// Config describes how to build the application.
type Config struct {
	Version   string
	GitCommit string
	GitDate   string
}

func (c Config) version() string {
	if c.Version != "" {
		return c.Version
	}
	return "v0.0.0"
}

// NewApp builds the CLI application without running it.
func NewApp(cfg Config) *cli.App {
	app := cli.NewApp()
	app.Flags = cliapp.ProtectFlags(flags.Flags)
	app.Version = opservice.FormatVersion(cfg.version(), cfg.GitCommit, cfg.GitDate, "")
	app.Name = "op-conductor"
	app.Usage = "Optimism Sequencer Conductor Service"
	app.Description = "op-conductor help sequencer to run in highly available mode"
	app.Action = cliapp.LifecycleCmd(Lifecycle(cfg))
	app.Commands = []*cli.Command{}
	return app
}

// Lifecycle returns the lifecycle action that constructs an OpConductor from the
// parsed CLI context.
func Lifecycle(cfg Config) cliapp.LifecycleAction {
	return func(ctx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
		logCfg := logcli.ReadCLIConfig(ctx)
		logger := logcli.NewLogger(logcli.AppOut(ctx), logCfg)
		logcli.SetGlobalLogHandler(logger.Handler())
		opservice.ValidateEnvVars(flags.EnvVarPrefix, flags.Flags, logger)

		conductorCfg, err := conductor.NewConfig(ctx, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}

		c, err := conductor.New(ctx.Context, conductorCfg, logger, cfg.version())
		if err != nil {
			return nil, fmt.Errorf("failed to create conductor: %w", err)
		}
		return c, nil
	}
}

// Run builds and runs the application, blocking until it exits. It is the whole
// body of a typical main().
func Run(cfg Config) {
	logcli.SetupDefaults()

	app := NewApp(cfg)
	ctx := ctxinterrupt.WithSignalWaiterMain(context.Background())
	if err := app.RunContext(ctx, os.Args); err != nil {
		oplog.Crit("Application failed", "message", err)
	}
}
