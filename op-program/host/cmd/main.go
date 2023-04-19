package main

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-program/host"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum-optimism/optimism/op-program/host/flags"
	"github.com/ethereum-optimism/optimism/op-program/host/version"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"
)

var (
	GitCommit = ""
	GitDate   = ""
)

// VersionWithMeta holds the textual version string including the metadata.
var VersionWithMeta = func() string {
	v := version.Version
	if GitCommit != "" {
		v += "-" + GitCommit[:8]
	}
	if GitDate != "" {
		v += "-" + GitDate
	}
	if version.Meta != "" {
		v += "-" + version.Meta
	}
	return v
}()

func main() {
	args := os.Args
	err := run(args, host.FaultProofProgram)
	if err != nil {
		log.Crit("Application failed", "message", err)
	} else {
		log.Info("Claim successfully verified")
	}
}

type ConfigAction func(log log.Logger, config *config.Config) error

// run parses the supplied args to create a config.Config instance, sets up logging
// then calls the supplied ConfigAction.
// This allows testing the translation from CLI arguments to Config
func run(args []string, action ConfigAction) error {
	// Set up logger with a default INFO level in case we fail to parse flags,
	// otherwise the final critical log won't show what the parsing error was.
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Version = VersionWithMeta
	app.Flags = flags.Flags
	app.Name = "op-program"
	app.Usage = "Optimism Fault Proof Program"
	app.Description = "The Optimism Fault Proof Program fault proof program that runs through the rollup state-transition to verify an L2 output from L1 inputs."
	app.Action = func(ctx *cli.Context) error {
		logger, err := setupLogging(ctx)
		if err != nil {
			return err
		}
		logger.Info("Starting fault proof program", "version", VersionWithMeta)

		cfg, err := config.NewConfigFromCLI(ctx)
		if err != nil {
			return err
		}
		return action(logger, cfg)
	}

	return app.Run(args)
}

func setupLogging(ctx *cli.Context) (log.Logger, error) {
	logCfg := oplog.ReadCLIConfig(ctx)
	if err := logCfg.Check(); err != nil {
		return nil, fmt.Errorf("log config error: %w", err)
	}
	logger := oplog.NewLogger(logCfg)
	return logger, nil
}
