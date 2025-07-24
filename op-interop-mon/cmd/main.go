package main

import (
	"context"
	"os"

	"github.com/HashKeyChain/verse/op-service/ctxinterrupt"

	opservice "github.com/HashKeyChain/verse/op-service"
	"github.com/urfave/cli/v2"

	"github.com/HashKeyChain/verse/op-interop-mon/flags"
	"github.com/HashKeyChain/verse/op-interop-mon/metrics"
	"github.com/HashKeyChain/verse/op-interop-mon/monitor"

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
	app.Name = "op-interop-mon"
	app.Usage = "Interop Monitoring Service"
	app.Description = "Service for monitoring interop transactions across the Superchain"
	app.Action = cliapp.LifecycleCmd(monitor.Main(Version))
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
