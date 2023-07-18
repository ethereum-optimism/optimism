package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/ethereum-optimism/optimism/indexer"
	"github.com/ethereum-optimism/optimism/indexer/config"

	"github.com/ethereum/go-ethereum/log"
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
	configPath := ctx.String(ConfigFlag.Name)
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return err
	}

	// setup logger
	cfg.Logger = log.New()
	cfg.Logger.SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stdout, log.TerminalFormat(true))))

	indexer, err := indexer.NewIndexer(cfg)
	if err != nil {
		return err
	}

	signalChannel := make(chan os.Signal, 1)
	indexerCtx, indexerCancel := context.WithCancel(context.Background())
	signal.Notify(signalChannel, os.Interrupt)
	go func() {
		<-signalChannel
		indexerCancel()
	}()

	return indexer.Run(indexerCtx)
}

func runApi(ctx *cli.Context) error {
	configPath := ctx.String(ConfigFlag.Name)
	conf, err := config.LoadConfig(configPath)

	fmt.Println(conf)

	if err != nil {
		log.Crit("Failed to load config", "message", err)
	}

	// finish me
	return nil
}

var (
	ConfigFlag = &cli.StringFlag{
		Name:    "config",
		Value:   "./indexer.toml",
		Aliases: []string{"c"},
		Usage:   "path to config file",
		EnvVars: []string{"INDEXER_CONFIG"},
	}
	// Not used yet.  Use this flag to run legacy app instead
	// Remove me after indexer is released
	IndexerRefreshFlag = &cli.BoolFlag{
		Name:    "indexer-refresh",
		Value:   false,
		Aliases: []string{"i"},
		Usage:   "run new unreleased indexer by passing in flag",
		EnvVars: []string{"INDEXER_REFRESH"},
	}
)

// make a instance method on Cli called Run that runs cli
// and returns an error
func (c *Cli) Run(args []string) error {
	return c.app.Run(args)
}

func NewCli(GitVersion string, GitCommit string, GitDate string) *Cli {
	log.Root().SetHandler(
		log.LvlFilterHandler(
			log.LvlInfo,
			log.StreamHandler(os.Stdout, log.TerminalFormat(true)),
		),
	)

	flags := []cli.Flag{
		ConfigFlag,
	}

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
				Name:        "indexer",
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
