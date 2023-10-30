package game

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault"
	"github.com/ethereum-optimism/optimism/op-challenger/game/loader"
	"github.com/ethereum-optimism/optimism/op-challenger/game/registry"
	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-challenger/version"
	opClient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/log"
)

type Service struct {
	logger  log.Logger
	metrics metrics.Metricer
	monitor *gameMonitor
	sched   *scheduler.Scheduler

	pprofSrv   *httputil.HTTPServer
	metricsSrv *httputil.HTTPServer
}

func (s *Service) Stop(ctx context.Context) error {
	var result error
	if s.sched != nil {
		result = errors.Join(result, s.sched.Close())
	}
	if s.pprofSrv != nil {
		result = errors.Join(result, s.pprofSrv.Stop(ctx))
	}
	if s.metricsSrv != nil {
		result = errors.Join(result, s.metricsSrv.Stop(ctx))
	}
	return result
}

// NewService creates a new Service.
func NewService(ctx context.Context, logger log.Logger, cfg *config.Config) (*Service, error) {
	cl := clock.SystemClock
	m := metrics.NewMetrics()
	txMgr, err := txmgr.NewSimpleTxManager("challenger", logger, &m.TxMetrics, cfg.TxMgrConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create the transaction manager: %w", err)
	}

	l1Client, err := dial.DialEthClientWithTimeout(ctx, dial.DefaultDialTimeout, logger, cfg.L1EthRpc)
	if err != nil {
		return nil, fmt.Errorf("failed to dial L1: %w", err)
	}

	s := &Service{
		logger:  logger,
		metrics: m,
	}

	pprofConfig := cfg.PprofConfig
	if pprofConfig.Enabled {
		logger.Debug("starting pprof", "addr", pprofConfig.ListenAddr, "port", pprofConfig.ListenPort)
		pprofSrv, err := oppprof.StartServer(pprofConfig.ListenAddr, pprofConfig.ListenPort)
		if err != nil {
			return nil, errors.Join(fmt.Errorf("failed to start pprof server: %w", err), s.Stop(ctx))
		}
		s.pprofSrv = pprofSrv
		logger.Info("started pprof server", "addr", pprofSrv.Addr())
	}

	metricsCfg := cfg.MetricsConfig
	if metricsCfg.Enabled {
		logger.Debug("starting metrics server", "addr", metricsCfg.ListenAddr, "port", metricsCfg.ListenPort)
		metricsSrv, err := m.Start(metricsCfg.ListenAddr, metricsCfg.ListenPort)
		if err != nil {
			return nil, errors.Join(fmt.Errorf("failed to start metrics server: %w", err), s.Stop(ctx))
		}
		logger.Info("started metrics server", "addr", metricsSrv.Addr())
		s.metricsSrv = metricsSrv
		m.StartBalanceMetrics(ctx, logger, l1Client, txMgr.From())
	}

	factoryContract, err := bindings.NewDisputeGameFactory(cfg.GameFactoryAddress, l1Client)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to bind the fault dispute game factory contract: %w", err), s.Stop(ctx))
	}
	loader := loader.NewGameLoader(factoryContract)

	gameTypeRegistry := registry.NewGameTypeRegistry()
	fault.RegisterGameTypes(gameTypeRegistry, ctx, logger, m, cfg, txMgr, l1Client)

	disk := newDiskManager(cfg.Datadir)
	s.sched = scheduler.NewScheduler(
		logger,
		m,
		disk,
		cfg.MaxConcurrency,
		gameTypeRegistry.CreatePlayer)

	pollClient, err := opClient.NewRPCWithClient(ctx, logger, cfg.L1EthRpc, opClient.NewBaseRPCClient(l1Client.Client()), cfg.PollInterval)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to create RPC client: %w", err), s.Stop(ctx))
	}
	s.monitor = newGameMonitor(logger, cl, loader, s.sched, cfg.GameWindow, l1Client.BlockNumber, cfg.GameAllowlist, pollClient)

	m.RecordInfo(version.SimpleWithMeta)
	m.RecordUp()

	return s, nil
}

// MonitorGame monitors the fault dispute game and attempts to progress it.
func (s *Service) MonitorGame(ctx context.Context) error {
	s.sched.Start(ctx)
	err := s.monitor.MonitorGames(ctx)
	// The other ctx is the close-trigger.
	// We need to refactor Service more to allow for graceful/force-shutdown granularity.
	err = errors.Join(err, s.Stop(context.Background()))
	return err
}
