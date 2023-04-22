package main

import (
	"os"

	challenger "github.com/refcell/op-challenger/challenger"
	flags "github.com/refcell/op-challenger/flags"

	log "github.com/ethereum/go-ethereum/log"
	cli "github.com/urfave/cli"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

const Version = "1.0.0"

func main() {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Flags = flags.Flags
	app.Version = Version
	app.Name = "op-challenger"
	app.Usage = "Modular Challenger Agent"
	app.Description = "A modular op-stack challenge agent for output dispute games written in golang."

	app.Action = func(ctx *cli.Context) error {
		return challenger.Main(Version, ctx)
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}
