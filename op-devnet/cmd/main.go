package main

import (
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	devnet "github.com/ethereum-optimism/optimism/op-devnet/devnet"
	flags "github.com/ethereum-optimism/optimism/op-devnet/flags"
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
	app.Name = "devnet"
	app.Usage = "Brings up a local devnet"
	app.Description = "Brings up a local devnet"
	app.Action = devnet.Main

	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}
