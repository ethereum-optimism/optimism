package main

import (
	"context"
	"os"

	"github.com/HashKeyChain/verse/op-service/ctxinterrupt"

	opservice "github.com/HashKeyChain/verse/op-service"
	"github.com/urfave/cli/v2"

	"github.com/HashKeyChain/verse/op-proposer/flags"
	"github.com/HashKeyChain/verse/op-proposer/metrics"
	"github.com/HashKeyChain/verse/op-proposer/proposer"
	"github.com/HashKeyChain/verse/op-service/cliapp"
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
	app.Name = "op-proposer"
	app.Usage = "L2 Output Submitter"
	app.Description = "Service for generating and proposing L2 Outputs"
	app.Action = cliapp.LifecycleCmd(proposer.Main(Version))
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
