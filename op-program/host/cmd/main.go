package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-program/host"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum-optimism/optimism/op-program/host/flags"
	"github.com/ethereum-optimism/optimism/op-program/host/version"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
)

var (
	GitCommit = ""
	GitDate   = ""
)

// VersionWithMeta holds the textual version string including the metadata.
var VersionWithMeta = opservice.FormatVersion(version.Version, GitCommit, GitDate, version.Meta)

func main() {
	// Set up logger with a default INFO level in case we fail to parse flags,
	// otherwise the final critical log won't show what the parsing error was.
	oplog.SetupDefaults()

	ctx := opio.WithInterruptBlocker(context.Background())
	if err := run(ctx, os.Args, host.Main); err != nil {
		log.Crit("Application failed", "err", err)
	}
}

type ConfigAction func(logger log.Logger, cfg *config.Config, onComplete context.CancelCauseFunc) (cliapp.Lifecycle, error)

// run parses the supplied args to create a config.Config instance, sets up logging
// then calls the supplied ConfigAction.
// This allows testing the translation from CLI arguments to Config.
// With an oppio.WithBlocker(ctx, fn) as ctx, the app-interruption can be customized.
func run(ctx context.Context, args []string, action ConfigAction) error {
	app := cli.NewApp()
	app.Version = VersionWithMeta
	app.Flags = flags.Flags
	app.Name = "op-program"
	app.Usage = "Optimism Fault Proof Program"
	app.Description = "The Optimism Fault Proof Program fault proof program that runs through the rollup state-transition to verify an L2 output from L1 inputs."
	app.Action = cliapp.LifecycleCmd(Main(action))

	return app.RunContext(ctx, args)
}

func Main(action ConfigAction) cliapp.LifecycleAction {
	return func(cliCtx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
		logger, err := setupLogging(cliCtx)
		if err != nil {
			return nil, err
		}
		opservice.ValidateEnvVars(flags.EnvVarPrefix, flags.Flags, logger)

		logger.Info("Starting fault proof program", "version", VersionWithMeta)

		cfg, err := config.NewConfigFromCLI(logger, cliCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
		return action(logger, cfg, closeApp)
	}
}

func setupLogging(ctx *cli.Context) (log.Logger, error) {
	logCfg := oplog.ReadCLIConfig(ctx)
	logger := oplog.NewLogger(oplog.AppOut(ctx), logCfg)
	oplog.SetGlobalLogHandler(logger.GetHandler())
	return logger, nil
}
