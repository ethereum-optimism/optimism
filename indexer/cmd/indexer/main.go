package main

import (
	"os"

	"github.com/ethereum/go-ethereum/log"
)

var (
	GitVersion = ""
	GitCommit  = ""
	GitDate    = ""
)

func main() {
	app := newCli(GitCommit, GitDate)
	if err := app.Run(os.Args); err != nil {
		log.Crit("Application failed", "message", err)
	}
}
