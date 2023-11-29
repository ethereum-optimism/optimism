package main

import (
	"context"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-blob-archiver/archiver"
	"github.com/ethereum-optimism/optimism/op-blob-archiver/flags"

	// "github.com/ethereum-optimism/optimism/op-blob-archiver/metrics"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
	"github.com/ethereum/go-ethereum/log"
)

var (
	Version   = "v0.1.0"
	GitCommit = ""
	GitDate   = ""
)

func main() {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Flags = cliapp.ProtectFlags(flags.Flags)
	app.Version = opservice.FormatVersion(Version, GitCommit, GitDate, "")
	app.Name = "op-blob-archiver"
	app.Usage = "Blob Archiver Service"
	app.Description = "Service for archiving blobs"
	// change this line
	app.Action = cliapp.LifecycleCmd(archiver.Main(Version))
	// app.Commands = []*cli.Command{
	// 	{
	// 		Name:        "doc",
	// 		Subcommands: doc.NewSubcommands(metrics.NewMetrics("default")),
	// 	},
	// }

	ctx := opio.WithInterruptBlocker(context.Background())
	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}
