package main

import (
	"context"
	"os"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

var Version = "v0.0.1"

func main() {

	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Flags = cliapp.ProtectFlags(Flags)
	app.Version = opservice.FormatVersion(Version, "", "", "")
	app.Name = "avail-da-server"
	app.Usage = "Plasma Avail DA Service"
	app.Description = "Service for interacting with Avail DA"
	app.Action = StartDAServer

	ctx := opio.WithInterruptBlocker(context.Background())
	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}
