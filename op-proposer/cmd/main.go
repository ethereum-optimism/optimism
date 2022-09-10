package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"

	proposer "github.com/ethereum-optimism/optimism/op-proposer"
	"github.com/ethereum-optimism/optimism/op-proposer/flags"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
)

var (
	Version   = ""
	GitCommit = ""
	GitDate   = ""
)

func main() {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Flags = flags.Flags
	app.Version = fmt.Sprintf("%s-%s-%s", Version, GitCommit, GitDate)
	app.Name = "op-proposer"
	app.Usage = "L2Output Submitter"
	app.Description = "Service for generating and submitting L2 Output " +
		"checkpoints to the L2OutputOracle contract"

	app.Action = proposer.Main(Version)
	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}
