package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/l2os"
	"github.com/ethereum-optimism/optimism/l2os/flags"
)

var (
	Version   = ""
	GitCommit = ""
	GitDate   = ""
)

func main() {
	// Set up logger with a default INFO level in case we fail to parse flags,
	// otherwise the final critical log won't show what the parsing error was.
	log.Root().SetHandler(
		log.LvlFilterHandler(
			log.LvlInfo,
			log.StreamHandler(os.Stdout, log.TerminalFormat(true)),
		),
	)

	app := cli.NewApp()
	app.Flags = flags.Flags
	app.Version = fmt.Sprintf("%s-%s-%s", Version, GitCommit, GitDate)
	app.Name = "l2os"
	app.Usage = "L2Output Submitter"
	app.Description = "Service for generating and submitting L2 Output " +
		"checkpoints to the L2OutputOracle contract"

	app.Action = l2os.Main(Version)
	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}
