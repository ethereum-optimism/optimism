package main

import (
	"os"

	challenger "github.com/ethereum-optimism/optimism/op-challenger/challenger"
	flags "github.com/ethereum-optimism/optimism/op-challenger/flags"

	log "github.com/ethereum/go-ethereum/log"
	cli "github.com/urfave/cli"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

const Version = "0.1.0"

func main() {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Flags = flags.Flags
	app.Version = Version
	app.Name = "op-challenger"
	app.Usage = "Challenge invalid L2OutputOracle outputs"
	app.Description = "A modular op-stack challenge agent for dispute games written in golang."
	app.Action = curryMain(Version)
	app.Commands = []cli.Command{}

	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}

// curryMain transforms the challenger.Main function into an app.Action
// This is done to capture the Version of the challenger.
func curryMain(version string) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		return challenger.Main(version, ctx)
	}
}
