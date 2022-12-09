package main

import (
	"fmt"
	"os"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"

	signer "github.com/ethereum-optimism/optimism/signer"
)

var (
	Version   = ""
	GitCommit = ""
	GitDate   = ""
)

func main() {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Flags = signer.CLIFlags("SIGNER_")
	app.Version = fmt.Sprintf("%s-%s-%s", Version, GitCommit, GitDate)
	app.Name = "signer"
	app.Usage = "Signing Service"
	app.Description = ""

	app.Action = signer.Main(Version)
	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}
