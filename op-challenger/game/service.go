package game

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/loader"
	"github.com/ethereum-optimism/optimism/op-challenger/game/registry"
	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-challenger/version"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

type Service struct {
	logger  log.Logger
	metrics metrics.Metricer
	monitor *gameMonitor
	sched   *scheduler.Scheduler

	faultGamesCloser fault.CloseFunc

	txMgr *txmgr.SimpleTxManager

	loader *loader.GameLoader

	rollupClient *sources.RollupClient

	l1Client   *ethclient.Client
	pollClient client.RPC

	pprofSrv   *httputil.HTTPServer
	metricsSrv *httputil.HTTPServer

	balanceMetricer io.Closer

	stopped atomic.Bool
}

// NewService creates a new Service.
func NewService(ctx context.Context, logger log.Logger, cfg *config.Config) (*Service, error) {
	s := &Service{
		logger:  logger,
		metrics: metrics.NewMetrics(),
	}

	if err := s.initFromConfig(ctx, cfg); err != nil {
		// upon initialization error we can try to close any of the service components that may have started already.
		return nil, errors.Join(fmt.Errorf("failed to init challenger game service: %w", err), s.Stop(ctx))
	}

	return s, nil
}

func (s *Service) initFromConfig(ctx context.Context, cfg *config.Config) error {
	if err := s.initTxManager(cfg); err != nil {
		return err
	}
	if err := s.initL1Client(ctx, cfg); err != nil {
		return err
	}
	if err := s.initRollupClient(ctx, cfg); err != nil {
		return err
	}
	if err := s.initPollClient(ctx, cfg); err != nil {
		return err
	}
	if err := s.initPProfServer(&cfg.PprofConfig); err != nil {
		return err
	}
	if err := s.initMetricsServer(&cfg.MetricsConfig); err != nil {
		return err
	}
	if err := s.initGameLoader(cfg); err != nil {
		return err
	}
	if err := s.initScheduler(ctx, cfg); err != nil {
		return err
	}

	s.initMonitor(cfg)

	s.metrics.RecordInfo(version.SimpleWithMeta)
	s.metrics.RecordUp()
	return nil
}

func (s *Service) initTxManager(cfg *config.Config) error {
	txMgr, err := txmgr.NewSimpleTxManager("challenger", s.logger, s.metrics, cfg.TxMgrConfig)
	if err != nil {
		return fmt.Errorf("failed to create the transaction manager: %w", err)
	}
	s.txMgr = txMgr
	return nil
}

func (s *Service) initL1Client(ctx context.Context, cfg *config.Config) error {
	l1Client, err := dial.DialEthClientWithTimeout(ctx, dial.DefaultDialTimeout, s.logger, cfg.L1EthRpc)
	if err != nil {
		return fmt.Errorf("failed to dial L1: %w", err)
	}
	s.l1Client = l1Client
	return nil
}

func (s *Service) initPollClient(ctx context.Context, cfg *config.Config) error {
	pollClient, err := client.NewRPCWithClient(ctx, s.logger, cfg.L1EthRpc, client.NewBaseRPCClient(s.l1Client.Client()), cfg.PollInterval)
	if err != nil {
		return fmt.Errorf("failed to create RPC client: %w", err)
	}
	s.pollClient = pollClient
	return nil
}

func (s *Service) initPProfServer(cfg *oppprof.CLIConfig) error {
	if !cfg.Enabled {
		return nil
	}
	s.logger.Debug("starting pprof", "addr", cfg.ListenAddr, "port", cfg.ListenPort)
	pprofSrv, err := oppprof.StartServer(cfg.ListenAddr, cfg.ListenPort)
	if err != nil {
		return fmt.Errorf("failed to start pprof server: %w", err)
	}
	s.pprofSrv = pprofSrv
	s.logger.Info("started pprof server", "addr", pprofSrv.Addr())
	return nil
}

func (s *Service) initMetricsServer(cfg *opmetrics.CLIConfig) error {
	if !cfg.Enabled {
		return nil
	}
	s.logger.Debug("starting metrics server", "addr", cfg.ListenAddr, "port", cfg.ListenPort)
	m, ok := s.metrics.(opmetrics.RegistryMetricer)
	if !ok {
		return fmt.Errorf("metrics were enabled, but metricer %T does not expose registry for metrics-server", s.metrics)
	}
	metricsSrv, err := opmetrics.StartServer(m.Registry(), cfg.ListenAddr, cfg.ListenPort)
	if err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}
	s.logger.Info("started metrics server", "addr", metricsSrv.Addr())
	s.metricsSrv = metricsSrv
	s.balanceMetricer = s.metrics.StartBalanceMetrics(s.logger, s.l1Client, s.txMgr.From())
	return nil
}

func (s *Service) initGameLoader(cfg *config.Config) error {
	factoryContract, err := contracts.NewDisputeGameFactoryContract(cfg.GameFactoryAddress,
		batching.NewMultiCaller(s.l1Client.Client(), batching.DefaultBatchSize))
	if err != nil {
		return fmt.Errorf("failed to bind the fault dispute game factory contract: %w", err)
	}
	s.loader = loader.NewGameLoader(factoryContract)
	return nil
}

func (s *Service) initRollupClient(ctx context.Context, cfg *config.Config) error {
	if cfg.RollupRpc == "" {
		return nil
	}
	rollupClient, err := dial.DialRollupClientWithTimeout(ctx, dial.DefaultDialTimeout, s.logger, cfg.RollupRpc)
	if err != nil {
		return err
	}
	s.rollupClient = rollupClient
	return nil
}

func (s *Service) initScheduler(ctx context.Context, cfg *config.Config) error {
	gameTypeRegistry := registry.NewGameTypeRegistry()
	caller := batching.NewMultiCaller(s.l1Client.Client(), batching.DefaultBatchSize)
	closer, err := fault.RegisterGameTypes(gameTypeRegistry, ctx, s.logger, s.metrics, cfg, s.rollupClient, s.txMgr, caller)
	if err != nil {
		return err
	}
	s.faultGamesCloser = closer

	disk := newDiskManager(cfg.Datadir)
	s.sched = scheduler.NewScheduler(s.logger, s.metrics, disk, cfg.MaxConcurrency, gameTypeRegistry.CreatePlayer)
	return nil
}

func (s *Service) initMonitor(cfg *config.Config) {
	cl := clock.SystemClock
	s.monitor = newGameMonitor(s.logger, cl, s.loader, s.sched, cfg.GameWindow, s.l1Client.BlockNumber, cfg.GameAllowlist, s.pollClient)
}

func (s *Service) Start(ctx context.Context) error {
	s.logger.Info("starting scheduler")
	s.sched.Start(ctx)
	s.logger.Info("starting monitoring")
	s.monitor.StartMonitoring()
	s.logger.Info("challenger game service start completed")
	return nil
}

func (s *Service) Stopped() bool {
	return s.stopped.Load()
}

func (s *Service) Stop(ctx context.Context) error {
	s.logger.Info("stopping challenger game service")

	var result error
	if s.sched != nil {
		if err := s.sched.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close scheduler: %w", err))
		}
	}
	if s.monitor != nil {
		s.monitor.StopMonitoring()
	}
	if s.faultGamesCloser != nil {
		s.faultGamesCloser()
	}
	if s.pprofSrv != nil {
		if err := s.pprofSrv.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close pprof server: %w", err))
		}
	}
	if s.balanceMetricer != nil {
		if err := s.balanceMetricer.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close balance metricer: %w", err))
		}
	}

	if s.txMgr != nil {
		s.txMgr.Close()
	}

	if s.rollupClient != nil {
		s.rollupClient.Close()
	}
	if s.pollClient != nil {
		s.pollClient.Close()
	}
	if s.l1Client != nil {
		s.l1Client.Close()
	}
	if s.metricsSrv != nil {
		if err := s.metricsSrv.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close metrics server: %w", err))
		}
	}
	s.stopped.Store(true)
	s.logger.Info("stopped challenger game service", "err", result)
	return result
}
