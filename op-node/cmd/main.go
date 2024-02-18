package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum-optimism/optimism/op-node"
	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/cmd/genesis"
	"github.com/ethereum-optimism/optimism/op-node/cmd/networks"
	"github.com/ethereum-optimism/optimism/op-node/cmd/p2p"
	"github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/version"
	"github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/metrics/doc"
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
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	app := cli.NewApp()
	app.Version = VersionWithMeta
	app.Flags = cliapp.ProtectFlags(flags.Flags)
	app.Name = "op-node"
	app.Usage = "Optimism Rollup Node"
	app.Description = "The Optimism Rollup Node derives L2 block inputs from L1 data and drives an external L2 Execution Engine to build a L2 chain."
	app.Action = cliapp.LifecycleCmd(RollupNodeMain)
	app.Commands = []*cli.Command{
		{
			Name:        "p2p",
			Subcommands: p2p.Subcommands,
		},
		{
			Name:        "genesis",
			Subcommands: genesis.Subcommands,
		},
		{
			Name:        "doc",
			Subcommands: doc.NewSubcommands(metrics.NewMetrics("default")),
		},
		{
			Name:        "networks",
			Subcommands: networks.Subcommands,
		},
	}

	ctx := opio.WithInterruptBlocker(context.Background())
	err := app.RunContext(ctx, os.Args)
	if err != nil {
		logrus.Fatal("Application failed", err)
	}
}

func RollupNodeMain(ctx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := logrus.New()
	opservice.ValidateEnvVars(flags.EnvVarPrefix, flags.Flags, log)
	opservice.WarnOnDeprecatedFlags(ctx, flags.DeprecatedFlags, log)
	m := metrics.NewMetrics("default")

	cfg, err := opnode.NewConfig(ctx, log)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create the rollup node config")
	}
	cfg.Cancel = closeApp

	snapshotLog, err := opnode.NewSnapshotLogger(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create snapshot root logger")
	}
	
	// Only pretty-print the banner if it is a terminal log. Other log it as key-value pairs.
	if log.Formatter.(*logrus.TextFormatter).ForceColors {
		log.Info("rollup config:\n" + cfg.Rollup.Description(chaincfg.L2ChainIDToNetworkDisplayName))
	} else {
		cfg.Rollup.LogDescription(log, chaincfg.L2ChainIDToNetworkDisplayName)
	}

	n, err := node.New(ctx.Context, cfg, log, snapshotLog, VersionWithMeta, m)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create the rollup node")
	}

	return n, nil
}