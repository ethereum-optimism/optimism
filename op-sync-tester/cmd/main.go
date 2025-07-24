package main

import (
	"context"
	"io"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/log"

	opservice "github.com/HashKeyChain/verse/op-service"
	"github.com/HashKeyChain/verse/op-service/cliapp"
	"github.com/HashKeyChain/verse/op-service/ctxinterrupt"
	oplog "github.com/HashKeyChain/verse/op-service/log"
	"github.com/HashKeyChain/verse/op-service/metrics/doc"
	"github.com/HashKeyChain/verse/op-sync-tester/config"
	"github.com/HashKeyChain/verse/op-sync-tester/flags"
	"github.com/HashKeyChain/verse/op-sync-tester/metrics"
	"github.com/HashKeyChain/verse/op-sync-tester/synctester"
)

var (
	Version   = "v0.0.0"
	GitCommit = ""
	GitDate   = ""
)

func main() {
	ctx := ctxinterrupt.WithSignalWaiterMain(context.Background())
	err := run(ctx, os.Stdout, os.Stderr, os.Args, fromConfig)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}

func run(ctx context.Context, w io.Writer, ew io.Writer, args []string, fn synctester.MainFn) error {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Writer = w
	app.ErrWriter = ew
	app.Flags = cliapp.ProtectFlags(flags.Flags)
	app.Version = opservice.FormatVersion(Version, GitCommit, GitDate, "")
	app.Name = "op-sync-tester"
	app.Usage = "op-sync-tester mocks EL layer to test CL sync"
	app.Description = "op-sync-tester mocks EL layer to test CL sync"
	app.Action = cliapp.LifecycleCmd(synctester.Main(app.Version, fn))
	app.Commands = []*cli.Command{
		{
			Name:        "doc",
			Subcommands: doc.NewSubcommands(metrics.NewMetrics("default")),
		},
	}
	return app.RunContext(ctx, args)
}

func fromConfig(ctx context.Context, cfg *config.Config, logger log.Logger) (cliapp.Lifecycle, error) {
	return synctester.FromConfig(ctx, cfg, logger)
}
