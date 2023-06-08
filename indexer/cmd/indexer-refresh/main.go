package main

import (
	"os"

	"github.com/ethereum-optimism/optimism/indexer/cli"
	"github.com/ethereum/go-ethereum/log"
)

var (
	GitVersion = ""
	GitCommit  = ""
	GitDate    = ""
)

func main() {
	app := cli.NewCli(GitVersion, GitCommit, GitDate)

	if err := app.Run(os.Args); err != nil {
		log.Crit("Application failed", "message", err)
	}
}
