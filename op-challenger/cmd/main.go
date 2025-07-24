package main

import (
	"context"
	"os"

	"github.com/HashKeyChain/verse/op-challenger/metrics"
	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/log"

	challenger "github.com/HashKeyChain/verse/op-challenger"
	"github.com/HashKeyChain/verse/op-challenger/config"
	"github.com/HashKeyChain/verse/op-challenger/flags"
	"github.com/HashKeyChain/verse/op-challenger/version"
	opservice "github.com/HashKeyChain/verse/op-service"
	"github.com/HashKeyChain/verse/op-service/cliapp"
	"github.com/HashKeyChain/verse/op-service/ctxinterrupt"
	oplog "github.com/HashKeyChain/verse/op-service/log"
)

var (
	GitCommit = ""
	GitDate   = ""
)

// VersionWithMeta holds the textual version string including the metadata.
var VersionWithMeta = opservice.FormatVersion(version.Version, GitCommit, GitDate, version.Meta)

func main() {
	args := os.Args
	ctx := ctxinterrupt.WithSignalWaiterMain(context.Background())
	if err := run(ctx, args, func(ctx context.Context, l log.Logger, config *config.Config) (cliapp.Lifecycle, error) {
		return challenger.Main(ctx, l, config, metrics.NewMetrics())
	}); err != nil {
		log.Crit("Application failed", "err", err)
	}
}

type ConfiguredLifecycle func(ctx context.Context, log log.Logger, config *config.Config) (cliapp.Lifecycle, error)

func run(ctx context.Context, args []string, action ConfiguredLifecycle) error {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Version = VersionWithMeta
	app.Flags = cliapp.ProtectFlags(flags.Flags)
	app.Name = "op-challenger"
	app.Usage = "Challenge outputs"
	app.Description = "Ensures that on chain outputs are correct."
	app.Commands = []*cli.Command{
		ListGamesCommand,
		ListClaimsCommand,
		ListCreditsCommand,
		CreateGameCommand,
		MoveCommand,
		ResolveCommand,
		ResolveClaimCommand,
		RunTraceCommand,
	}
	app.Action = cliapp.LifecycleCmd(func(ctx *cli.Context, close context.CancelCauseFunc) (cliapp.Lifecycle, error) {
		logger, err := setupLogging(ctx)
		if err != nil {
			return nil, err
		}
		logger.Info("Starting op-challenger", "version", VersionWithMeta)

		cfg, err := flags.NewConfigFromCLI(ctx, logger)
		if err != nil {
			return nil, err
		}
		return action(ctx.Context, logger, cfg)
	})
	return app.RunContext(ctx, args)
}

func setupLogging(ctx *cli.Context) (log.Logger, error) {
	logCfg := oplog.ReadCLIConfig(ctx)
	logger := oplog.NewLogger(oplog.AppOut(ctx), logCfg)
	oplog.SetGlobalLogHandler(logger.Handler())
	return logger, nil
}
