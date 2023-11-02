package main

import (
	"context"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/indexer"
	"github.com/ethereum-optimism/optimism/indexer/api"
	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
)

var (
	ConfigFlag = &cli.StringFlag{
		Name:    "config",
		Value:   "./indexer.toml",
		Aliases: []string{"c"},
		Usage:   "path to config file",
		EnvVars: []string{"INDEXER_CONFIG"},
	}
	MigrationsFlag = &cli.StringFlag{
		Name:    "migrations-dir",
		Value:   "./migrations",
		Usage:   "path to migrations folder",
		EnvVars: []string{"INDEXER_MIGRATIONS_DIR"},
	}
)

func runIndexer(ctx *cli.Context, shutdown context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx)).New("role", "indexer")
	oplog.SetGlobalLogHandler(log.GetHandler())
	log.Info("running indexer...")

	cfg, err := config.LoadConfig(log, ctx.String(ConfigFlag.Name))
	if err != nil {
		log.Error("failed to load config", "err", err)
		return nil, err
	}

	return indexer.NewIndexer(ctx.Context, log, &cfg, shutdown)
}

func runApi(ctx *cli.Context, _ context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx)).New("role", "api")
	oplog.SetGlobalLogHandler(log.GetHandler())
	log.Info("running api...")

	cfg, err := config.LoadConfig(log, ctx.String(ConfigFlag.Name))
	if err != nil {
		log.Error("failed to load config", "err", err)
		return nil, err
	}

	apiCfg := &api.Config{
		DB:            &api.DBConfigConnector{DBConfig: cfg.DB},
		HTTPServer:    cfg.HTTPServer,
		MetricsServer: cfg.MetricsServer,
	}

	return api.NewApi(ctx.Context, log, apiCfg)
}

func runMigrations(ctx *cli.Context) error {
	// We don't maintain a complicated lifecycle here, just interrupt to shut down.
	ctx.Context = opio.CancelOnInterrupt(ctx.Context)

	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx)).New("role", "migrations")
	oplog.SetGlobalLogHandler(log.GetHandler())
	log.Info("running migrations...")

	cfg, err := config.LoadConfig(log, ctx.String(ConfigFlag.Name))
	if err != nil {
		log.Error("failed to load config", "err", err)
		return err
	}

	db, err := database.NewDB(ctx.Context, log, cfg.DB)
	if err != nil {
		log.Error("failed to connect to database", "err", err)
		return err
	}
	defer db.Close()

	migrationsDir := ctx.String(MigrationsFlag.Name)
	return db.ExecuteSQLMigration(migrationsDir)
}

func newCli(GitCommit string, GitDate string) *cli.App {
	flags := []cli.Flag{ConfigFlag}
	flags = append(flags, oplog.CLIFlags("INDEXER")...)
	migrationFlags := []cli.Flag{MigrationsFlag, ConfigFlag}
	migrationFlags = append(migrationFlags, oplog.CLIFlags("INDEXER")...)
	return &cli.App{
		Version:              params.VersionWithCommit(GitCommit, GitDate),
		Description:          "An indexer of all optimism events with a serving api layer",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:        "api",
				Flags:       flags,
				Description: "Runs the api service",
				Action:      cliapp.LifecycleCmd(runApi),
			},
			{
				Name:        "index",
				Flags:       flags,
				Description: "Runs the indexing service",
				Action:      cliapp.LifecycleCmd(runIndexer),
			},
			{
				Name:        "migrate",
				Flags:       migrationFlags,
				Description: "Runs the database migrations",
				Action:      runMigrations,
			},
			{
				Name:        "version",
				Description: "print version",
				Action: func(ctx *cli.Context) error {
					cli.ShowVersion(ctx)
					return nil
				},
			},
		},
	}
}
