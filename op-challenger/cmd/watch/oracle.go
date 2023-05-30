package watch

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	log "github.com/ethereum/go-ethereum/log"

	challenger "github.com/ethereum-optimism/optimism/op-challenger/challenger"
	config "github.com/ethereum-optimism/optimism/op-challenger/config"
	metrics "github.com/ethereum-optimism/optimism/op-challenger/metrics"
)

// Oracle listens to the L2OutputOracle for newly proposed outputs.
func Oracle(logger log.Logger, cfg *config.Config) error {
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	m := metrics.NewMetrics("default")

	service, err := challenger.NewChallenger(*cfg, logger, m)
	if err != nil {
		logger.Error("Unable to create the Challenger", "error", err)
		return err
	}

	logger.Info("Listening for OutputProposed events from the L2OutputOracle contract", "l2oo", cfg.L2OOAddress.String())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subscription, err := service.NewOracleSubscription()
	if err != nil {
		logger.Error("Unable to create the subscription", "error", err)
		return err
	}

	err = subscription.Subscribe()
	if err != nil {
		logger.Error("Unable to subscribe to the L2OutputOracle contract", "error", err)
		return err
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

	m.RecordUp()

	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, []os.Signal{
		os.Interrupt,
		os.Kill,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	}...)

	for {
		select {
		case log := <-subscription.Logs():
			logger.Info("Received log", "log", log)
		case <-interruptChannel:
			logger.Info("Received interrupt signal, exiting...")
		}
	}
}
