package app

import (
	"fmt"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

type Service interface {
	Flags(envVarPrefix string) []cli.Flag
	Subcommands() cli.Commands
	Init(logger log.Logger, ctx *cli.Context) error
}

type PreConfigure func(app *cli.App)

type Action func(l log.Logger, ctx *cli.Context) error

func Run(args []string, envVarPrefix string, configure PreConfigure, action Action, services ...Service) error {
	oplog.SetupDefaults()

	app := cli.NewApp()
	configure(app)

	app.Flags = append(app.Flags, oplog.CLIFlags(envVarPrefix)...)
	for _, service := range services {
		app.Flags = append(app.Flags, service.Flags(envVarPrefix)...)
		app.Commands = append(app.Commands, service.Subcommands()...)
	}

	app.Action = func(ctx *cli.Context) error {
		logger, err := setupLogging(ctx)
		if err != nil {
			return err
		}
		for _, service := range services {
			if err := service.Init(logger, ctx); err != nil {
				return err
			}
		}
		logger.Info("Starting", "app", app.Name, "version", app.Version)

		// TODO: Add interrupt handling here
		return action(logger, ctx)
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
