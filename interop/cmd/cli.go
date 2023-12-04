package main

import (
	"context"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/interop"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

func newCli() *cli.App {
	flags := oplog.CLIFlags("INTEROP_POSTIE")
	return &cli.App{
		Description: "",
		Commands: []*cli.Command{
			{
				Name:   "interop-postie",
				Flags:  flags,
				Action: cliapp.LifecycleCmd(runPostie),
			},
		},
	}
}

func runPostie(cli *cli.Context, shutdown context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	return interop.NewPostie(), nil
}
