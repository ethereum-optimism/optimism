package main

import (
	"os"

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

	app.Action = func(ctx *cli.Context) error {
		log.Debug("Challenger not implemented...")
		return nil
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}
