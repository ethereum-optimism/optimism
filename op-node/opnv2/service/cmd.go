package service

import (
	"context"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/config"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/flags"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/metrics"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/metrics/doc"
)

type VersionConfig struct {
	Version   string
	GitCommit string
	GitDate   string
}

func RunCmd(ctx context.Context, args []string, versionCfg VersionConfig, fn MainFn) error {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Flags = cliapp.ProtectFlags(flags.Flags)
	app.Version = opservice.FormatVersion(versionCfg.Version, versionCfg.GitCommit, versionCfg.GitDate, "opnv2")
	app.Name = "op-node"
	app.Usage = "op-node V2 OP-Stack consensus-layer node"
	app.Description = "The op-node syncs and verifies OP-Stack chains."
	app.Action = cliapp.LifecycleCmd(Main(app.Version, fn))
	app.Commands = []*cli.Command{
		{
			Name:        "doc",
			Subcommands: doc.NewSubcommands(metrics.NewMetrics()),
		},
	}
	return app.RunContext(ctx, args)
}

func LifecycleFromConfig(ctx context.Context, cfg *config.Config, logger log.Logger) (cliapp.Lifecycle, error) {
	logger.Info("Starting op-node V2!")
	return FromConfig(ctx, cfg, logger)
}
