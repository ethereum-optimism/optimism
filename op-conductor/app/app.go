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

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-conductor/conductor"
	"github.com/ethereum-optimism/optimism/op-conductor/flags"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

// Config describes how to build the application.
type Config struct {
	// Name and Usage default to op-conductor's own if left empty.
	Name  string
	Usage string

	Version   string
	GitCommit string
	GitDate   string

	// ExtraFlags are registered alongside op-conductor's own flags.
	ExtraFlags []cli.Flag

	// Options is called once the CLI context is available, so that embedders can
	// build conductor options from their own flags. May be nil.
	Options func(ctx *cli.Context, log log.Logger) ([]conductor.Option, error)
}

func (c Config) name() string {
	if c.Name != "" {
		return c.Name
	}
	return "op-conductor"
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
	app.Flags = cliapp.ProtectFlags(append(append([]cli.Flag{}, flags.Flags...), cfg.ExtraFlags...))
	app.Version = opservice.FormatVersion(cfg.version(), cfg.GitCommit, cfg.GitDate, "")
	app.Name = cfg.name()
	if cfg.Usage != "" {
		app.Usage = cfg.Usage
	} else {
		app.Usage = "Optimism Sequencer Conductor Service"
	}
	app.Description = "op-conductor help sequencer to run in highly available mode"
	app.Action = cliapp.LifecycleCmd(Lifecycle(cfg))
	app.Commands = []*cli.Command{}
	return app
}

// Lifecycle returns the lifecycle action that constructs an OpConductor from the
// parsed CLI context.
func Lifecycle(cfg Config) cliapp.LifecycleAction {
	return func(ctx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
		logCfg := oplog.ReadCLIConfig(ctx)
		logger := oplog.NewLogger(oplog.AppOut(ctx), logCfg)
		oplog.SetGlobalLogHandler(logger.Handler())
		opservice.ValidateEnvVars(flags.EnvVarPrefix, flags.Flags, logger)

		conductorCfg, err := conductor.NewConfig(ctx, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}

		var opts []conductor.Option
		if cfg.Options != nil {
			opts, err = cfg.Options(ctx, logger)
			if err != nil {
				return nil, fmt.Errorf("failed to build conductor options: %w", err)
			}
		}

		c, err := conductor.New(ctx.Context, conductorCfg, logger, cfg.version(), opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create conductor: %w", err)
		}
		return c, nil
	}
}

// Run builds and runs the application, blocking until it exits. It is the whole
// body of a typical main().
func Run(cfg Config) {
	oplog.SetupDefaults()

	app := NewApp(cfg)
	ctx := ctxinterrupt.WithSignalWaiterMain(context.Background())
	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Crit("Application failed", "message", err)
	}
}
