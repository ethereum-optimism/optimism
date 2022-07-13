package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum-optimism/optimism/op-node/metrics"

	opnode "github.com/ethereum-optimism/optimism/op-node"

	"github.com/ethereum-optimism/optimism/op-node/version"

	"github.com/ethereum-optimism/optimism/op-node/flags"

	"github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"
)

var (
	GitCommit = ""
	GitDate   = ""
)

// VersionWithMeta holds the textual version string including the metadata.
var VersionWithMeta = func() string {
	v := version.Version
	if GitCommit != "" {
		v += "-" + GitCommit[:8]
	}
	if GitDate != "" {
		v += "-" + GitDate
	}
	if version.Meta != "" {
		v += "-" + version.Meta
	}
	return v
}()

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
	app.Version = VersionWithMeta
	app.Name = "opnode"
	app.Usage = "Optimism Rollup Node"
	app.Description = "The deposit only rollup node drives the L2 execution engine based on L1 deposits."

	app.Action = RollupNodeMain
	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}

func RollupNodeMain(ctx *cli.Context) error {
	log.Info("Initializing Rollup Node")
	logCfg, err := opnode.NewLogConfig(ctx)
	if err != nil {
		log.Error("Unable to create the log config", "error", err)
		return err
	}
	log := logCfg.NewLogger()
	m := metrics.NewMetrics("default")

	cfg, err := opnode.NewConfig(ctx, log)
	if err != nil {
		log.Error("Unable to create the rollup node config", "error", err)
		return err
	}
	snapshotLog, err := opnode.NewSnapshotLogger(ctx)
	if err != nil {
		log.Error("Unable to create snapshot root logger", "error", err)
		return err
	}

	n, err := node.New(context.Background(), cfg, log, snapshotLog, VersionWithMeta, m)
	if err != nil {
		log.Error("Unable to create the rollup node", "error", err)
		return err
	}
	log.Info("Starting rollup node")

	if err := n.Start(context.Background()); err != nil {
		log.Error("Unable to start rollup node", "error", err)
		return err
	}
	defer n.Close()

	m.RecordInfo(VersionWithMeta)
	m.RecordUp()
	log.Info("Rollup node started")

	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, []os.Signal{
		os.Interrupt,
		os.Kill,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	}...)
	<-interruptChannel

	return nil

}
