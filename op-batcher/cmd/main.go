package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-batcher/cmd/doc"
	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
	"github.com/ethereum/go-ethereum/log"
)

var (
	Version   = "v0.10.14"
	GitCommit = ""
	GitDate   = ""
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Flags = flags.Flags
	app.Version = fmt.Sprintf("%s-%s-%s", Version, GitCommit, GitDate)
	app.Name = "op-batcher"
	app.Usage = "Batch Submitter Service"
	app.Description = "Service for generating and submitting L2 tx batches to L1"
	app.Action = curryMain(cancel, Version)
	app.Commands = []*cli.Command{
		{
			Name:        "doc",
			Subcommands: doc.Subcommands,
		},
	}

	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}

// curryMain transforms the batcher.Main function into an app.Action
// This is done to capture the Version and closure of the batcher.
func curryMain(cancel func(), version string) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		shutdown, err := batcher.Main(version, ctx)
		if err != nil {
			return err
		}

		opio.BlockOnInterrupts()
		log.Crit("Caught interrupt, shutting down...")
		cancel()
		shutdown()
		return nil
	}
}
