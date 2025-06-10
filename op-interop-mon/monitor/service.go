package monitor

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/op-interop-mon/metrics"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"

	"github.com/ethereum/go-ethereum/log"
)

var ErrAlreadyStopped = errors.New("already stopped")

type InteropMonitorConfig struct {
	PollInterval time.Duration
}

type InteropMonitorService struct {
	Log     log.Logger
	Metrics metrics.Metricer

	InteropMonitorConfig

	clients    map[eth.ChainID]*sources.EthClient
	maintainer *Maintainer
	finders    []Finder
	updaters   []Updater

	Version string

	pprofService *oppprof.Service
	metricsSrv   *httputil.HTTPServer
	rpcServer    *oprpc.Server

	stopped atomic.Bool
}

func InteropMonitorServiceFromCLIConfig(ctx context.Context, version string, cfg *CLIConfig, log log.Logger) (*InteropMonitorService, error) {
	var ms InteropMonitorService
	if err := ms.initFromCLIConfig(ctx, version, cfg, log); err != nil {
		return nil, errors.Join(err, ms.Start(ctx))
	}
	return &ms, nil
}

func (ms *InteropMonitorService) initFromCLIConfig(ctx context.Context, version string, cfg *CLIConfig, log log.Logger) error {
	ms.Version = version
	ms.Log = log

	ms.initMetrics(cfg)

	ms.PollInterval = cfg.PollInterval

	ms.maintainer = NewMaintainer(ms.Log, ms.Metrics)

	ms.clients = make(map[eth.ChainID]*sources.EthClient)
	for _, l2Rpc := range cfg.L2Rpcs {
		if err := ms.dialAndRegister(ctx, l2Rpc); err != nil {
			return fmt.Errorf("failed to dial and register: %w", err)
		}
	}

	if err := ms.initMetricsServer(cfg); err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}
	if err := ms.initPProf(cfg); err != nil {
		return fmt.Errorf("failed to init pprof server: %w", err)
	}
	if err := ms.initRPCServer(cfg); err != nil {
		return fmt.Errorf("failed to start rpc server: %w", err)
	}

	ms.Metrics.RecordInfo(ms.Version)
	ms.Metrics.RecordUp()
	fmt.Println("initialized from cli config")
	return nil
}

func (ms *InteropMonitorService) dialAndRegister(ctx context.Context, l2Rpc string) error {
	fmt.Println("dialing and registering", l2Rpc)
	client, err := client.NewRPC(ctx, ms.Log, l2Rpc)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	ethClient, err := sources.NewEthClient(client, ms.Log, nil, sources.DefaultEthClientConfig(1000))
	if err != nil {
		return fmt.Errorf("failed to create eth client: %w", err)
	}
	fmt.Println("created eth client")
	chainIDBig, err := ethClient.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %w", err)
	}
	chainID := eth.ChainIDFromBig(chainIDBig)
	ms.clients[chainID] = ethClient
	fmt.Println("added eth client to map")

	finder := NewFinder(chainID, ethClient, BlockReceiptsToJobs, ms.maintainer.EnqueueNew, ms.Log)
	updater := NewUpdater(chainID, ethClient, ms.Log)
	ms.finders = append(ms.finders, finder)
	ms.updaters = append(ms.updaters, updater)
	ms.maintainer.AddClient(chainID, ethClient)
	ms.maintainer.AddFinder(chainID, finder)
	ms.maintainer.AddUpdater(chainID, updater)
	fmt.Println("added finder and updater to maintainer")
	return nil
}

func (ms *InteropMonitorService) initMetrics(cfg *CLIConfig) {
	if cfg.MetricsConfig.Enabled {
		procName := "default"
		ms.Metrics = metrics.NewMetrics(procName)
	} else {
		ms.Metrics = metrics.NoopMetrics
	}
}

func (ms *InteropMonitorService) initPProf(cfg *CLIConfig) error {
	ms.pprofService = oppprof.New(
		cfg.PprofConfig.ListenEnabled,
		cfg.PprofConfig.ListenAddr,
		cfg.PprofConfig.ListenPort,
		cfg.PprofConfig.ProfileType,
		cfg.PprofConfig.ProfileDir,
		cfg.PprofConfig.ProfileFilename,
	)

	if err := ms.pprofService.Start(); err != nil {
		return fmt.Errorf("failed to start pprof service: %w", err)
	}

	return nil
}

func (ms *InteropMonitorService) initMetricsServer(cfg *CLIConfig) error {
	if !cfg.MetricsConfig.Enabled {
		ms.Log.Info("metrics disabled")
		return nil
	}
	m, ok := ms.Metrics.(opmetrics.RegistryMetricer)
	if !ok {
		return fmt.Errorf("metrics were enabled, but metricer %T does not expose registry for metrics-server", ms.Metrics)
	}
	ms.Log.Debug("starting metrics server", "addr", cfg.MetricsConfig.ListenAddr, "port", cfg.MetricsConfig.ListenPort)
	metricsSrv, err := opmetrics.StartServer(m.Registry(), cfg.MetricsConfig.ListenAddr, cfg.MetricsConfig.ListenPort)
	if err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}
	ms.Log.Info("started metrics server", "addr", metricsSrv.Addr())
	ms.metricsSrv = metricsSrv
	return nil
}

func (ms *InteropMonitorService) initRPCServer(cfg *CLIConfig) error {
	server := oprpc.NewServer(
		cfg.RPCConfig.ListenAddr,
		cfg.RPCConfig.ListenPort,
		ms.Version,
		oprpc.WithLogger(ms.Log),
		oprpc.WithRPCRecorder(ms.Metrics.NewRecorder("main")),
	)
	if cfg.RPCConfig.EnableAdmin {
		ms.Log.Info("admin rpc enabled, but no admin APIs are available")
	}
	ms.Log.Info("starting json-rpc server")
	if err := server.Start(); err != nil {
		return fmt.Errorf("unable to start rpc server: %w", err)
	}
	ms.rpcServer = server
	return nil
}

func (ms *InteropMonitorService) Start(ctx context.Context) error {
	err := ms.maintainer.Start()
	if err != nil {
		return fmt.Errorf("failed to start maintainer: %w", err)
	}
	for _, updater := range ms.updaters {
		if err := updater.Start(ctx); err != nil {
			return fmt.Errorf("failed to start updater: %w", err)
		}
	}
	for _, finder := range ms.finders {
		if err := finder.Start(ctx); err != nil {
			return fmt.Errorf("failed to start finder: %w", err)
		}
	}
	return nil
}

func (ms *InteropMonitorService) Stopped() bool {
	return ms.stopped.Load()
}

func (ms *InteropMonitorService) Kill() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return ms.Stop(ctx)
}

func (ms *InteropMonitorService) Stop(ctx context.Context) error {
	if ms.Stopped() {
		return ErrAlreadyStopped
	}
	var result error

	ms.Log.Info("stopping finders")
	for _, finder := range ms.finders {
		if err := finder.Stop(); err != nil {
			ms.Log.Error("failed to stop finder", "error", err)
			result = errors.Join(result, fmt.Errorf("failed to stop finder: %w", err))
		}
	}

	ms.Log.Info("stopping updaters")
	for _, updater := range ms.updaters {
		if err := updater.Stop(); err != nil {
			ms.Log.Error("failed to stop updater", "error", err)
		}
	}

	ms.Log.Info("stopping maintainer")
	if err := ms.maintainer.Stop(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to stop maintainer: %w", err))
		ms.Log.Error("failed to stop maintainer", "error", err)
	}

	ms.Log.Info("stopping rpc server")
	if ms.rpcServer != nil {
		if err := ms.rpcServer.Stop(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop rpc server: %w", err))
		}
	}

	ms.Log.Info("stopping pprof server")
	if ms.pprofService != nil {
		if err := ms.pprofService.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop pprof server: %w", err))
		}
	}

	ms.Log.Info("stopping metrics server")
	if ms.metricsSrv != nil {
		if err := ms.metricsSrv.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop metrics server: %w", err))
		}
	}

	if result == nil {
		ms.stopped.Store(true)
		ms.Log.Info("stopped all services")
	}

	return result
}

var _ cliapp.Lifecycle = (*InteropMonitorService)(nil)

func (ms *InteropMonitorService) Maintainer() *Maintainer {
	return ms.maintainer
}
