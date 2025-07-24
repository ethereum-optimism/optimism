package main

import (
	"context"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/HashKeyChain/verse/op-batcher/batcher"
	"github.com/HashKeyChain/verse/op-batcher/flags"
	"github.com/HashKeyChain/verse/op-batcher/metrics"
	opservice "github.com/HashKeyChain/verse/op-service"
	"github.com/HashKeyChain/verse/op-service/cliapp"
	"github.com/HashKeyChain/verse/op-service/ctxinterrupt"
	oplog "github.com/HashKeyChain/verse/op-service/log"
	"github.com/HashKeyChain/verse/op-service/metrics/doc"
	"github.com/ethereum/go-ethereum/log"
)

var (
	Version   = "v0.0.0"
	GitCommit = ""
	GitDate   = ""
)

func main() {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Flags = cliapp.ProtectFlags(flags.Flags)
	app.Version = opservice.FormatVersion(Version, GitCommit, GitDate, "")
	app.Name = "op-batcher"
	app.Usage = "Batch Submitter Service"
	app.Description = "Service for generating and submitting L2 tx batches to L1"
	app.Action = cliapp.LifecycleCmd(batcher.Main(Version))
	app.Commands = []*cli.Command{
		{
			Name:        "doc",
			Subcommands: doc.NewSubcommands(metrics.NewMetrics("default")),
		},
	}

	ctx := ctxinterrupt.WithSignalWaiterMain(context.Background())
	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}
