package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/cmd/doc"

	"github.com/urfave/cli"

	"github.com/ethereum/go-ethereum/log"

	opnode "github.com/ethereum-optimism/optimism/op-node"
	"github.com/ethereum-optimism/optimism/op-node/cmd/genesis"
	"github.com/ethereum-optimism/optimism/op-node/cmd/p2p"
	"github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/heartbeat"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/version"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
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
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Version = VersionWithMeta
	app.Flags = flags.Flags
	app.Name = "op-node"
	app.Usage = "Optimism Rollup Node"
	app.Description = "The Optimism Rollup Node derives L2 block inputs from L1 data and drives an external L2 Execution Engine to build a L2 chain."
	app.Action = RollupNodeMain
	app.Commands = []cli.Command{
		{
			Name:        "p2p",
			Subcommands: p2p.Subcommands,
		},
		{
			Name:        "genesis",
			Subcommands: genesis.Subcommands,
		},
		{
			Name:        "doc",
			Subcommands: doc.Subcommands,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}

func RollupNodeMain(ctx *cli.Context) error {
	log.Info("Initializing Rollup Node")
	logCfg := oplog.ReadCLIConfig(ctx)
	if err := logCfg.Check(); err != nil {
		log.Error("Unable to create the log config", "error", err)
		return err
	}
	log := oplog.NewLogger(logCfg)
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

	// Only pretty-print the banner if it is a terminal log. Other log it as key-value pairs.
	if logCfg.Format == "terminal" {
		log.Info("rollup config:\n" + cfg.Rollup.Description(chaincfg.L2ChainIDToNetworkName))
	} else {
		cfg.Rollup.LogDescription(log, chaincfg.L2ChainIDToNetworkName)
	}

	n, err := node.New(context.Background(), cfg, log, snapshotLog, VersionWithMeta, m)
	if err != nil {
		log.Error("Unable to create the rollup node", "error", err)
		return err
	}
	log.Info("Starting rollup node", "version", VersionWithMeta)

	if err := n.Start(context.Background()); err != nil {
		log.Error("Unable to start rollup node", "error", err)
		return err
	}
	defer n.Close()

	m.RecordInfo(VersionWithMeta)
	m.RecordUp()
	log.Info("Rollup node started")

	if cfg.Heartbeat.Enabled {
		var peerID string
		if cfg.P2P.Disabled() {
			peerID = "disabled"
		} else {
			peerID = n.P2P().Host().ID().String()
		}

		beatCtx, beatCtxCancel := context.WithCancel(context.Background())
		payload := &heartbeat.Payload{
			Version: version.Version,
			Meta:    version.Meta,
			Moniker: cfg.Heartbeat.Moniker,
			PeerID:  peerID,
			ChainID: cfg.Rollup.L2ChainID.Uint64(),
		}
		go func() {
			if err := heartbeat.Beat(beatCtx, log, cfg.Heartbeat.URL, payload); err != nil {
				log.Error("heartbeat goroutine crashed", "err", err)
			}
		}()
		defer beatCtxCancel()
	}

	if cfg.Pprof.Enabled {
		pprofCtx, pprofCancel := context.WithCancel(context.Background())
		go func() {
			log.Info("pprof server started", "addr", net.JoinHostPort(cfg.Pprof.ListenAddr, strconv.Itoa(cfg.Pprof.ListenPort)))
			if err := oppprof.ListenAndServe(pprofCtx, cfg.Pprof.ListenAddr, cfg.Pprof.ListenPort); err != nil {
				log.Error("error starting pprof", "err", err)
			}
		}()
		defer pprofCancel()
	}

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
