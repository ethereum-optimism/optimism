package cli

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ethereum-optimism/optimism/indexer"
	"github.com/ethereum-optimism/optimism/indexer/api"
	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
	"github.com/ethereum/go-ethereum/params"

	"github.com/urfave/cli/v2"
)

type Cli struct {
	GitVersion string
	GitCommit  string
	GitDate    string
	app        *cli.App
	Flags      []cli.Flag
}

func runIndexer(ctx *cli.Context) error {
	logger := log.NewLogger(log.ReadCLIConfig(ctx))

	configPath := ctx.String(ConfigFlag.Name)
	cfg, err := config.LoadConfig(logger, configPath)
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
		indexerCancel()
	}()

	return indexer.Run(indexerCtx)
}

func runApi(ctx *cli.Context) error {
	logger := log.NewLogger(log.ReadCLIConfig(ctx))

	configPath := ctx.String(ConfigFlag.Name)
	cfg, err := config.LoadConfig(logger, configPath)
	if err != nil {
		logger.Error("failed to load config", "err", err)
		return err
	}

	db, err := database.NewDB(cfg.DB)

	if err != nil {
		logger.Crit("Failed to connect to database", "err", err)
	}

	server := api.NewApi(db.BridgeTransfers, logger)

	return server.Listen(strconv.Itoa(cfg.API.Port))
}

var (
	ConfigFlag = &cli.StringFlag{
		Name:    "config",
		Value:   "./indexer.toml",
		Aliases: []string{"c"},
		Usage:   "path to config file",
		EnvVars: []string{"INDEXER_CONFIG"},
	}
)

// make a instance method on Cli called Run that runs cli
// and returns an error
func (c *Cli) Run(args []string) error {
	return c.app.Run(args)
}

func NewCli(GitVersion string, GitCommit string, GitDate string) *Cli {
	flags := []cli.Flag{ConfigFlag}
	flags = append(flags, log.CLIFlags("INDEXER")...)
	app := &cli.App{
		Version:     fmt.Sprintf("%s-%s", GitVersion, params.VersionWithCommit(GitCommit, GitDate)),
		Description: "An indexer of all optimism events with a serving api layer",
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
		},
	}

	return &Cli{
		app:   app,
		Flags: flags,
	}
}
