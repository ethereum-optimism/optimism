package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli"

	opnode "github.com/ethereum-optimism/optimism/op-node"
	"github.com/ethereum-optimism/optimism/op-node/cmd/genesis"
	"github.com/ethereum-optimism/optimism/op-node/cmd/p2p"
	"github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/heartbeat"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/version"
	"github.com/ethereum/go-ethereum/log"
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
	}

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

	if cfg.Heartbeat.Enabled {
		var peerID string
		if cfg.P2P == nil {
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
		var srv http.Server
		srv.Addr = net.JoinHostPort(cfg.Pprof.ListenAddr, cfg.Pprof.ListenPort)
		// Start pprof server + register it's shutdown
		go func() {
			log.Info("pprof server started", "addr", srv.Addr)
			if err := srv.ListenAndServe(); err != http.ErrServerClosed {
				log.Error("error in pprof server", "err", err)
			} else {
				log.Info("pprof server shutting down")
			}

		}()
		defer func() {
			shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err := srv.Shutdown(shutCtx)
			log.Info("pprof server shut down", "err", err)
		}()
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
