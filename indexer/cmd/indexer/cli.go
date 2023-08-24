package main

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

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
	logger := log.NewLogger(log.ReadCLIConfig(ctx)).New("role", "indexer")
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
	logger := log.NewLogger(log.ReadCLIConfig(ctx)).New("role", "api")
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

func runAll(cliCtx *cli.Context) error {
	logger := log.NewLogger(log.ReadCLIConfig(cliCtx))

	var wg sync.WaitGroup
	errCh := make(chan error, 2) // represents that 2 goroutines will be running

	_, sharedCancel := context.WithCancel(context.Background())
	defer sharedCancel()

	run := func(startFunc func(*cli.Context) error) {
		wg.Add(1)
		defer func() {
			if err := recover(); err != nil {
				log.NewLogger(log.ReadCLIConfig(cliCtx)).Error("halting on panic", "err", err)
				debug.PrintStack()
				errCh <- fmt.Errorf("panic: %v", err)
			}

			sharedCancel()
			wg.Done()
		}()

		err := startFunc(cliCtx)
		if err != nil {
			logger.Error("halting on error", "err", err)
		}

		errCh <- err
	}

	go run(runIndexer)
	go run(runApi)

	err := <-errCh

	wg.Wait()

	return err
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
