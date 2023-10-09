package batcher

import (
	"context"
	"fmt"
	_ "net/http/pprof"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-batcher/rpc"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
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

	l := oplog.NewLogger(oplog.AppOut(cliCtx), cfg.LogConfig)
	oplog.SetGlobalLogHandler(l.GetHandler())
	opservice.ValidateEnvVars(flags.EnvVarPrefix, flags.Flags, l)
	procName := "default"
	m := metrics.NewMetrics(procName)
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

	defer batchSubmitter.StopIfRunning(context.Background())

	pprofConfig := cfg.PprofConfig
	if pprofConfig.Enabled {
		l.Debug("starting pprof", "addr", pprofConfig.ListenAddr, "port", pprofConfig.ListenPort)
		pprofSrv, err := oppprof.StartServer(pprofConfig.ListenAddr, pprofConfig.ListenPort)
		if err != nil {
			l.Error("failed to start pprof server", "err", err)
			return err
		}
		l.Info("started pprof server", "addr", pprofSrv.Addr())
		defer pprofSrv.Close()
	}

	metricsCfg := cfg.MetricsConfig
	if metricsCfg.Enabled {
		l.Debug("starting metrics server", "addr", metricsCfg.ListenAddr, "port", metricsCfg.ListenPort)
		metricsSrv, err := m.Start(metricsCfg.ListenAddr, metricsCfg.ListenPort)
		if err != nil {
			return fmt.Errorf("failed to start metrics server: %w", err)
		}
		l.Info("started metrics server", "addr", metricsSrv.Addr())
		defer metricsSrv.Close()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		m.StartBalanceMetrics(ctx, l, batchSubmitter.L1Client, batchSubmitter.TxManager.From())
	}

	server := oprpc.NewServer(
		cfg.RPCFlag.ListenAddr,
		cfg.RPCFlag.ListenPort,
		version,
		oprpc.WithLogger(l),
	)
	if cfg.RPCFlag.EnableAdmin {
		adminAPI := rpc.NewAdminAPI(batchSubmitter, &m.RPCMetrics, l)
		server.AddAPI(rpc.GetAdminAPI(adminAPI))
		l.Info("Admin RPC enabled")
	}
	if err := server.Start(); err != nil {
		return fmt.Errorf("error starting RPC server: %w", err)
	}

	m.RecordInfo(version)
	m.RecordUp()

	opio.BlockOnInterrupts()
	if err := server.Stop(); err != nil {
		l.Error("Error shutting down http server: %w", err)
	}
	return nil
}
