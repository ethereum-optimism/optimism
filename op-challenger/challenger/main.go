package challenger

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	cli "github.com/urfave/cli"

	metrics "github.com/optimism/op-challenger/metrics"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
)

// Main is the entrypoint into the Challenger.
// This executes and blocks until the service exits.
func Main(version string, cliCtx *cli.Context) error {
	cfg := NewConfig(cliCtx)
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid CLI flags: %w", err)
	}

	l := oplog.NewLogger(cfg.LogConfig)
	m := metrics.NewMetrics("default")

	challengerConfig, err := NewChallengerConfigFromCLIConfig(cfg, l, m)
	if err != nil {
		l.Error("Unable to create the Challenger", "error", err)
		return err
	}

	challenger, err := NewChallenger(*challengerConfig, l, m)
	if err != nil {
		l.Error("Unable to create the Challenger", "error", err)
		return err
	}

	l.Info("Starting Challenger")
	ctx, cancel := context.WithCancel(context.Background())
	if err := challenger.Start(); err != nil {
		cancel()
		l.Error("Unable to start Challenger", "error", err)
		return err
	}
	defer challenger.Stop()

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
		m.StartBalanceMetrics(ctx, l, challengerConfig.L1Client, challengerConfig.From)
	}

	// rpcCfg := cfg.RPCConfig
	// server := oprpc.NewServer(rpcCfg.ListenAddr, rpcCfg.ListenPort, version, oprpc.WithLogger(l))
	// if err := server.Start(); err != nil {
	// 	cancel()
	// 	return fmt.Errorf("error starting RPC server: %w", err)
	// }

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
	cancel()

	return nil
}
