package main

import (
	"github.com/ethereum-optimism/optimism/indexer"
	"github.com/ethereum-optimism/optimism/indexer/api"
	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/urfave/cli/v2"
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

func runIndexer(ctx *cli.Context) error {
	log := log.NewLogger(log.ReadCLIConfig(ctx)).New("role", "indexer")
	cfg, err := config.LoadConfig(log, ctx.String(ConfigFlag.Name))
	if err != nil {
		log.Error("failed to load config", "err", err)
		return err
	}

	db, err := database.NewDB(cfg.DB)
	if err != nil {
		log.Error("failed to connect to database", "err", err)
		return err
	}
	defer db.Close()

	indexer, err := indexer.NewIndexer(log, db, cfg.Chain, cfg.RPCs, cfg.HTTPServer, cfg.MetricsServer)
	if err != nil {
		log.Error("failed to create indexer", "err", err)
		return err
	}

	return indexer.Run(ctx.Context)
}

func runApi(ctx *cli.Context) error {
	log := log.NewLogger(log.ReadCLIConfig(ctx)).New("role", "api")
	cfg, err := config.LoadConfig(log, ctx.String(ConfigFlag.Name))
	if err != nil {
		log.Error("failed to load config", "err", err)
		return err
	}

	db, err := database.NewDB(cfg.DB)
	if err != nil {
		log.Error("failed to connect to database", "err", err)
		return err
	}
	defer db.Close()

	api := api.NewApi(log, db.BridgeTransfers, cfg.HTTPServer, cfg.MetricsServer)
	return api.Start(ctx.Context)
}

func runMigrations(ctx *cli.Context) error {
	log := log.NewLogger(log.ReadCLIConfig(ctx)).New("role", "api")
	cfg, err := config.LoadConfig(log, ctx.String(ConfigFlag.Name))
	migrationsDir := ctx.String(MigrationsFlag.Name)
	if err != nil {
		log.Error("failed to load config", "err", err)
		return err
	}

	db, err := database.NewDB(cfg.DB)
	if err != nil {
		log.Error("failed to connect to database", "err", err)
		return err
	}
	defer db.Close()

	return db.ExecuteSQLMigration(migrationsDir)
}

func newCli(GitCommit string, GitDate string) *cli.App {
	flags := []cli.Flag{ConfigFlag}
	flags = append(flags, log.CLIFlags("INDEXER")...)
	migrationFlags := []cli.Flag{MigrationsFlag, ConfigFlag}
	migrationFlags = append(migrationFlags, log.CLIFlags("INDEXER")...)
	return &cli.App{
		Version:              params.VersionWithCommit(GitCommit, GitDate),
		Description:          "An indexer of all optimism events with a serving api layer",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:        "api",
				Flags:       flags,
				Description: "Runs the api service",
				Action:      runApi,
			},
			{
				Name:        "index",
				Flags:       flags,
				Description: "Runs the indexing service",
				Action:      runIndexer,
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
