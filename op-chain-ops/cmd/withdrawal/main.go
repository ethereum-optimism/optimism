package main

import (
	"context"
	"os"

	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

func main() {
	ctx := ctxinterrupt.WithSignalWaiterMain(context.Background())
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Name = "withdrawal"
	app.Usage = "Tool to perform withdrawals from OP Stack chains"
	app.Commands = []*cli.Command{
		InitCommand,
		ProveCommand,
		FinalizeCommand,
	}
	app.Action = func(c *cli.Context) error {
		return cli.ShowAppHelp(c)
	}
	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Crit("Application failed", "err", err)
	}
}
