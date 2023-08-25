package main

import (
	"context"

	"github.com/ethereum-optimism/optimism/indexer"
	"github.com/ethereum-optimism/optimism/indexer/api"
	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
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
)

func runIndexer(ctx *cli.Context) error {
	logger := log.NewLogger(log.ReadCLIConfig(ctx))
	cfg, err := config.LoadConfig(logger, ctx.String(ConfigFlag.Name))
	if err != nil {
		logger.Error("failed to load config", "err", err)
		return err
	}

	db, err := database.NewDB(cfg.DB)
	if err != nil {
		return err
	}

	indexer, err := indexer.NewIndexer(logger, cfg.Chain, cfg.RPCs, db)
	if err != nil {
		return err
	}

	indexerCtx, indexerCancel := context.WithCancel(context.Background())
	go func() {
		opio.BlockOnInterrupts()
		logger.Error("caught interrupt, shutting down...")
		indexerCancel()
	}()

	return indexer.Run(indexerCtx)
}

func runApi(ctx *cli.Context) error {
	logger := log.NewLogger(log.ReadCLIConfig(ctx))
	cfg, err := config.LoadConfig(logger, ctx.String(ConfigFlag.Name))
	if err != nil {
		logger.Error("failed to load config", "err", err)
		return err
	}

	db, err := database.NewDB(cfg.DB)
	if err != nil {
		logger.Crit("Failed to connect to database", "err", err)
	}

	apiCtx, apiCancel := context.WithCancel(context.Background())
	api := api.NewApi(logger, db.BridgeTransfers)
	go func() {
		opio.BlockOnInterrupts()
		logger.Error("caught interrupt, shutting down...")
		apiCancel()
	}()

	return api.Listen(apiCtx, cfg.API.Port)
}

func runAll(ctx *cli.Context) error {
	// Run the indexer
	go func() {
		if err := runIndexer(ctx); err != nil {
			log.NewLogger(log.ReadCLIConfig(ctx)).Error("Error running the indexer", "err", err)
		}
	}()

	// Run the API and return its error, if any
	return runApi(ctx)
}

func newCli(GitCommit string, GitDate string) *cli.App {
	flags := []cli.Flag{ConfigFlag}
	flags = append(flags, log.CLIFlags("INDEXER")...)
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
				Name:        "version",
				Description: "print version",
				Action: func(ctx *cli.Context) error {
					cli.ShowVersion(ctx)
					return nil
				},
			},
			{
				Name:        "all",
				Flags:       flags,
				Description: "Runs both the api service and the indexing service",
				Action:      runAll,
			},
		},
	}
}
