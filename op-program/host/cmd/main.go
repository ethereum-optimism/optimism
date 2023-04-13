package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	cldr "github.com/ethereum-optimism/optimism/op-program/client/driver"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum-optimism/optimism/op-program/host/flags"
	"github.com/ethereum-optimism/optimism/op-program/host/l1"
	"github.com/ethereum-optimism/optimism/op-program/host/l2"
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

var (
	ErrClaimNotValid = errors.New("invalid claim")
)

func main() {
	args := os.Args
	err := run(args, FaultProofProgram)
	if err != nil {
		log.Crit("Application failed", "message", err)
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

// FaultProofProgram is the programmatic entry-point for the fault proof program
func FaultProofProgram(logger log.Logger, cfg *config.Config) error {
	cfg.Rollup.LogDescription(logger, chaincfg.L2ChainIDToNetworkName)
	if !cfg.FetchingEnabled() {
		return errors.New("offline mode not supported")
	}

	ctx := context.Background()
	logger.Info("Connecting to L1 node", "l1", cfg.L1URL)
	l1Source, err := l1.NewFetchingL1(ctx, logger, cfg)
	if err != nil {
		return fmt.Errorf("connect l1 oracle: %w", err)
	}

	logger.Info("Connecting to L2 node", "l2", cfg.L2URL)
	l2Source, err := l2.NewFetchingEngine(ctx, logger, cfg)
	if err != nil {
		return fmt.Errorf("connect l2 oracle: %w", err)
	}

	d := cldr.NewDriver(logger, cfg.Rollup, l1Source, l2Source)
	for {
		if err = d.Step(ctx); errors.Is(err, io.EOF) {
			break
		} else if cfg.FetchingEnabled() && errors.Is(err, derive.ErrTemporary) {
			// When in fetching mode, recover from temporary errors to allow us to keep fetching data
			// TODO(CLI-3780) Ideally the retry would happen in the fetcher so this is not needed
			logger.Warn("Temporary error in pipeline", "err", err)
			time.Sleep(5 * time.Second)
		} else if err != nil {
			return err
		}
	}
	claim := cfg.L2Claim
	if !d.ValidateClaim(eth.Bytes32(claim)) {
		return ErrClaimNotValid
	}
	return nil
}
