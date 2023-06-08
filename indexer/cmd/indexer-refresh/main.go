package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/indexer"
	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/flags"
)

var (
	GitVersion = ""
	GitCommit  = ""
	GitDate    = ""
)

func main() {
	// Set up logger with a default INFO level in case we fail to parse flags.
	// Otherwise the final crtiical log won't show what the parsing error was.
	log.Root().SetHandler(
		log.LvlFilterHandler(
			log.LvlInfo,
			log.StreamHandler(os.Stdout, log.TerminalFormat(true)),
		),
	)

	// TODO https://linear.app/optimism/issue/DX-55/api-implement-rest-api-with-mocked-data
	// don't hardcode this
	conf, err := config.LoadConfig("../../indexer.toml")

	if err != nil {
		log.Crit("Failed to load config", "message", err)
	}

	log.Debug("Loaded config", "config", conf)

	app := cli.NewApp()
	app.Flags = []cli.Flag{flags.LogLevelFlag, flags.L1EthRPCFlag, flags.L2EthRPCFlag, flags.DBNameFlag}
	app.Version = fmt.Sprintf("%s-%s", GitVersion, params.VersionWithCommit(GitCommit, GitDate))
	app.Name = "indexer"
	app.Usage = "Indexer Service"
	app.Description = "Service for indexing deposits and withdrawals " +
		"by account on L1 and L2"

	app.Action = indexer.Main(GitVersion)
	if err := app.Run(os.Args); err != nil {
		log.Crit("Application failed", "message", err)
	}
}
