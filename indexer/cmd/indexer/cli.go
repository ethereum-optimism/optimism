package main

import (
	"context"
	"fmt"
	"math/big"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/indexer"
	"github.com/ethereum-optimism/optimism/indexer/api"
	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
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
	ReorgFlag = &cli.Uint64Flag{
		Name:    "l1-height",
		Aliases: []string{"height"},
		Usage: `the lowest l1 height that has been reorg'd. All L1 data and derived L2 state will be deleted. Since not all L1 blocks are
		indexed, this will find the maximum indexed height <= the marker, which may result in slightly more deleted state.`,
		Required: true,
	}
)

func runIndexer(ctx *cli.Context, shutdown context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx)).New("role", "indexer")
	oplog.SetGlobalLogHandler(log.Handler())
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
	oplog.SetGlobalLogHandler(log.Handler())
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
	oplog.SetGlobalLogHandler(log.Handler())
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

func runReorgDeletion(ctx *cli.Context) error {
	fromL1Height := ctx.Uint64(ReorgFlag.Name)

	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx)).New("role", "reorg-deletion")
	oplog.SetGlobalLogHandler(log.Handler())
	cfg, err := config.LoadConfig(log, ctx.String(ConfigFlag.Name))
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	l1Clnt, err := node.DialEthClient(ctx.Context, cfg.RPCs.L1RPC, node.NewMetrics(metrics.NewRegistry(), "l1"))
	if err != nil {
		return fmt.Errorf("failed to dial L1 client: %w", err)
	}
	l1Header, err := l1Clnt.BlockHeaderByNumber(big.NewInt(int64(fromL1Height)))
	if err != nil {
		return fmt.Errorf("failed to query L1 header at height: %w", err)
	} else if l1Header == nil {
		return fmt.Errorf("no header found at height")
	}

	db, err := database.NewDB(ctx.Context, log, cfg.DB)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	defer db.Close()
	return db.Transaction(func(db *database.DB) error {
		return db.Blocks.DeleteReorgedState(l1Header.Time)
	})
}

func newCli(GitCommit string, GitDate string) *cli.App {
	flags := append([]cli.Flag{ConfigFlag}, oplog.CLIFlags("INDEXER")...)
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
				Flags:       append(flags, MigrationsFlag),
				Description: "Runs the database migrations",
				Action:      runMigrations,
			},
			{
				Name:        "reorg-delete",
				Aliases:     []string{"reorg"},
				Flags:       append(flags, ReorgFlag),
				Description: "Deletes data that has been reorg'ed out of the canonical L1 chain",
				Action:      runReorgDeletion,
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
