package main

import (
	"context"
	"fmt"
	_ "net/http/pprof"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/opio"

	"github.com/ethereum-optimism/optimism/op-challenger/challenger"

	"github.com/ethereum-optimism/optimism/op-service/pprof"
	"github.com/ethereum-optimism/optimism/op-service/rpc"
)

// Main is the entrypoint into the Challenger. This method executes the
// service and blocks until the service exits.
func Main(logger log.Logger, version string, cfg *config.Config) error {
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	m := metrics.NewMetrics("default")
	logger.Info("Initializing Challenger")

	service, err := challenger.NewChallenger(*cfg, logger, m)
	if err != nil {
		logger.Error("Unable to create the Challenger", "error", err)
		return err
	}

	logger.Info("Starting Challenger")
	ctx, cancel := context.WithCancel(context.Background())
	if err := service.Start(); err != nil {
		cancel()
		logger.Error("Unable to start Challenger", "error", err)
		return err
	}
	defer service.Stop()

	logger.Info("Challenger started")
	pprofConfig := cfg.PprofConfig
	if pprofConfig.Enabled {
		logger.Info("starting pprof", "addr", pprofConfig.ListenAddr, "port", pprofConfig.ListenPort)
		go func() {
			if err := pprof.ListenAndServe(ctx, pprofConfig.ListenAddr, pprofConfig.ListenPort); err != nil {
				logger.Error("error starting pprof", "err", err)
			}
		}()
	}

	metricsCfg := cfg.MetricsConfig
	if metricsCfg.Enabled {
		log.Info("starting metrics server", "addr", metricsCfg.ListenAddr, "port", metricsCfg.ListenPort)
		go func() {
			if err := m.Serve(ctx, metricsCfg.ListenAddr, metricsCfg.ListenPort); err != nil {
				logger.Error("error starting metrics server", err)
			}
		}()
		m.StartBalanceMetrics(ctx, logger, service.Client(), service.From())
	}

	rpcCfg := cfg.RPCConfig
	server := rpc.NewServer(rpcCfg.ListenAddr, rpcCfg.ListenPort, version, rpc.WithLogger(logger))
	if err := server.Start(); err != nil {
		cancel()
		return fmt.Errorf("error starting RPC server: %w", err)
	}

	m.RecordInfo(version)
	m.RecordUp()

	opio.BlockOnInterrupts()
	cancel()

	return nil
}
