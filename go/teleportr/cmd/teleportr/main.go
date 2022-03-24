package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/go/teleportr"
	"github.com/ethereum-optimism/optimism/go/teleportr/flags"
)

var (
	GitVersion = ""
	GitCommit  = ""
	GitDate    = ""
)

func main() {
	// Set up logger with a default INFO level in case we fail to parse flags.
	// Otherwise the final critical log won't show what the parsing error was.
	log.Root().SetHandler(
		log.LvlFilterHandler(
			log.LvlInfo,
			log.StreamHandler(os.Stdout, log.TerminalFormat(true)),
		),
	)

	app := cli.NewApp()
	app.Flags = flags.Flags
	app.Version = fmt.Sprintf("%s-%s-%s", GitVersion, GitCommit, GitDate)
	app.Name = "teleportr"
	app.Usage = "Teleportr"
	app.Description = "Teleportr bridge from L1 to L2"
	app.Commands = []cli.Command{
		{
			Name:   "migrate",
			Usage:  "Migrates teleportr's database",
			Action: teleportr.Migrate(),
		},
	}

	app.Action = teleportr.Main(GitVersion)
	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}
