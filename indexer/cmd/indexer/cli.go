package main

import (
	"sync"

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

func runAll(ctx *cli.Context) error {
	log := log.NewLogger(log.ReadCLIConfig(ctx))

	// Ensure both processes complete before returning.
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		err := runApi(ctx)
		if err != nil {
			log.Error("api process non-zero exit", "err", err)
		}
	}()
	go func() {
		defer wg.Done()
		err := runIndexer(ctx)
		if err != nil {
			log.Error("indexer process non-zero exit", "err", err)
		}
	}()

	// We purposefully return no error since the indexer and api
	// have no inter-dependencies. We simply rely on the logs to
	// report a non-zero exit for either process.
	wg.Wait()
	return nil
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
				Name:        "all",
				Flags:       flags,
				Description: "Runs both the api service and the indexing service",
				Action:      runAll,
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
