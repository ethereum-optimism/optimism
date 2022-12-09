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
	app.Flags = signer.CLIFlags("SIGNER")
	app.Version = fmt.Sprintf("%s-%s-%s", Version, GitCommit, GitDate)
	app.Name = "signer"
	app.Usage = "Signing Service"
	app.Description = ""
	app.Commands = []cli.Command{
		{
			Name:  "client",
			Usage: "test client for signer service",
			Subcommands: []cli.Command{
				{
					Name:   "sign",
					Usage:  "sign a transaction",
					Action: signer.ClientSign(Version),
					Flags:  signer.ClientSignCLIFlags("SIGNER"),
				},
			},
		},
	}

	app.Action = signer.Server(Version)
	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}
