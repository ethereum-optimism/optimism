package batcher

import (
	"context"
	"fmt"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-batcher/rpc"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

const (
	// defaultDialTimeout is default duration the service will wait on
	// startup to make a connection to either the L1 or L2 backends.
	defaultDialTimeout = 5 * time.Second
)

// Main is the entrypoint into the Batch Submitter. This method returns a
// closure that executes the service and blocks until the service exits. The use
// of a closure allows the parameters bound to the top-level main package, e.g.
// GitVersion, to be captured and used once the function is executed.
func Main(version string, cliCtx *cli.Context) error {
	if err := flags.CheckRequired(cliCtx); err != nil {
		return err
	}
	cfg := NewConfig(cliCtx)
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid CLI flags: %w", err)
	}

	l := oplog.NewLogger(cfg.LogConfig)
	m := metrics.NewMetrics("default")
	l.Info("Initializing Batch Submitter")

	batchSubmitter, err := NewBatchSubmitterFromCLIConfig(cfg, l, m)
	if err != nil {
		l.Error("Unable to create Batch Submitter", "error", err)
		return err
	}

	if !cfg.Stopped {
		if err := batchSubmitter.Start(); err != nil {
			l.Error("Unable to start Batch Submitter", "error", err)
			return err
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Stop pprof and metrics only after main loop returns
	defer batchSubmitter.StopIfRunning(context.Background())

	pprofConfig := cfg.PprofConfig
	if pprofConfig.Enabled {
		l.Info("starting pprof", "addr", pprofConfig.ListenAddr, "port", pprofConfig.ListenPort)
		go func() {
			if err := oppprof.ListenAndServe(ctx, pprofConfig.ListenAddr, pprofConfig.ListenPort); err != nil {
				l.Error("error starting pprof", "err", err)
			}
		}()
	}

	metricsCfg := cfg.MetricsConfig
	if metricsCfg.Enabled {
		l.Info("starting metrics server", "addr", metricsCfg.ListenAddr, "port", metricsCfg.ListenPort)
		go func() {
			if err := m.Serve(ctx, metricsCfg.ListenAddr, metricsCfg.ListenPort); err != nil {
				l.Error("error starting metrics server", err)
			}
		}()
		m.StartBalanceMetrics(ctx, l, batchSubmitter.L1Client, batchSubmitter.TxManager.From())
	}

	rpcCfg := cfg.RPCConfig
	server := oprpc.NewServer(
		rpcCfg.ListenAddr,
		rpcCfg.ListenPort,
		version,
		oprpc.WithLogger(l),
	)
	if rpcCfg.EnableAdmin {
		server.AddAPI(gethrpc.API{
			Namespace: "admin",
			Service:   rpc.NewAdminAPI(batchSubmitter),
		})
		l.Info("Admin RPC enabled")
	}
	if err := server.Start(); err != nil {
		cancel()
		return fmt.Errorf("error starting RPC server: %w", err)
	}

	m.RecordInfo(version)
	m.RecordUp()

	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, []os.Signal{
		os.Interrupt,
		os.Kill,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	}...)
	<-interruptChannel
	if err := server.Stop(); err != nil {
		l.Error("Error shutting down http server: %w", err)
	}
	return nil
}
