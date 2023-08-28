package main

import (
	"context"
	"os"

	"github.com/ethereum-optimism/optimism/op-bootnode/bootnode"
	"github.com/ethereum-optimism/optimism/op-bootnode/flags"
	"github.com/ethereum-optimism/optimism/op-service/opio"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
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
	app.Name = "bootnode"
	app.Usage = "Rollup Bootnode"
	app.Description = "Broadcasts incoming P2P peers to each other, enabling peer bootstrapping."
	app.Action = bootnode.Main

	ctx, cancel := context.WithCancel(context.Background())
	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}

	opio.BlockOnInterrupts()
	log.Crit("Caught interrupt, shutting down...")
	cancel()
}
