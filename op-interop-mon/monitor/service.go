package monitor

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-interop-mon/metrics"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	"github.com/ethereum-optimism/optimism/op-service/locks"
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

	finders   map[eth.ChainID]Finder
	updaters  map[eth.ChainID]Updater
	collector *MetricCollector
	finalized *locks.RWMap[eth.ChainID, eth.NumberAndHash]

	// messageExpiryWindow (seconds) is sourced from the dependency set and passed to updaters.
	messageExpiryWindow uint64

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

// InteropMonitorServiceFromClients creates a new InteropMonitorService with pre-initialized clients
func InteropMonitorServiceFromClients(
	ctx context.Context,
	version string,
	cfg *CLIConfig,
	clients map[eth.ChainID]*sources.EthClient,
	log log.Logger,
) (*InteropMonitorService, error) {
	var ms InteropMonitorService
	if err := ms.initFromClients(ctx, version, cfg, clients, log); err != nil {
		return nil, errors.Join(err, ms.Start(ctx))
	}
	return &ms, nil
}

func (ms *InteropMonitorService) initFromCLIConfig(ctx context.Context, version string, cfg *CLIConfig, log log.Logger) error {
	// Initialize all clients
	clients, err := ms.initClients(ctx, cfg.L2Rpcs)
	if err != nil {
		return fmt.Errorf("failed to init clients: %w", err)
	}

	// Load the dependency set: it provides the authoritative chain set and the
	// message expiry window used for executing-message validity checks.
	loader := &depset.JSONDependencySetLoader{Path: cfg.DependencySetPath}
	depSet, err := loader.LoadDependencySet()
	if err != nil {
		return fmt.Errorf("failed to load dependency set: %w", err)
	}
	ms.messageExpiryWindow = depSet.MessageExpiryWindow()

	// Reconcile configured RPCs against the dependency set.
	for chainID := range clients {
		if !depSet.HasChain(chainID) {
			log.Warn("configured L2 RPC chain is not in the dependency set", "chain_id", chainID)
		}
	}
	for _, chainID := range depSet.Chains() {
		if _, ok := clients[chainID]; !ok {
			return fmt.Errorf("dependency set chain %s has no configured L2 RPC; cannot validate its initiating messages", chainID)
		}
	}

	return ms.initFromClients(ctx, version, cfg, clients, log)
}

// initFromClients initializes the service with pre-created clients
func (ms *InteropMonitorService) initFromClients(
	ctx context.Context,
	version string,
	cfg *CLIConfig,
	clients map[eth.ChainID]*sources.EthClient,
	log log.Logger,
) error {
	ms.Version = version
	ms.Log = log

	ms.initMetrics(cfg)

	ms.PollInterval = cfg.PollInterval

	// Initialize the expiry map
	ms.finalized = locks.RWMapFromMap(make(map[eth.ChainID]eth.NumberAndHash))

	// Default the message expiry window when not sourced from a dependency set
	// (e.g. the pre-built-clients constructor used in tests). The CLI path sets
	// this from the dependency set before calling initFromClients.
	if ms.messageExpiryWindow == 0 {
		ms.messageExpiryWindow = depset.MessageExpiryTimeSecondsInterop
	}

	// Initialize all updaters
	ms.updaters = make(map[eth.ChainID]Updater)
	if err := ms.initUpdaters(clients); err != nil {
		return fmt.Errorf("failed to init updaters: %w", err)
	}

	// Initialize all finders
	ms.finders = make(map[eth.ChainID]Finder)
	if err := ms.initFinders(clients); err != nil {
		return fmt.Errorf("failed to init finders: %w", err)
	}

	if cfg.MetricsConfig.Enabled {
		// Initialize the metric collector, with access to all updaters
		ms.collector = NewMetricCollector(ms.Log, ms.Metrics, ms.updaters)
	}

	// Optional read-only interop-filter observer (cross-check + failsafe gauge).
	if cfg.InteropFilterEndpoint != "" && ms.collector != nil {
		minSafety := safety.Level(cfg.InteropFilterMinSafety)
		// The interop-filter only checks access lists at unsafe or cross-unsafe; any
		// other level errors on every call. Fail fast rather than flooding divergences.
		if minSafety != safety.CrossUnsafe && minSafety != safety.LocalUnsafe {
			return fmt.Errorf("interop-filter-min-safety %q unsupported; the filter only supports %q or %q", cfg.InteropFilterMinSafety, safety.CrossUnsafe, safety.LocalUnsafe)
		}
		filterClient, err := NewFilterClient(cfg.InteropFilterEndpoint, minSafety, ms.Log)
		if err != nil {
			return fmt.Errorf("failed to init interop-filter client: %w", err)
		}
		ms.collector.filterObserver = NewFilterObserver(filterClient, ms.Metrics, ms.Log)
	}

	// Optional read-only supernode observers (liveness, heads, cross-safety violations).
	if len(cfg.SupernodeEndpoints) > 0 && ms.collector != nil {
		// The observer confirms a job's executing block is still canonical (by hash)
		// on its chain before flagging a cross-safety violation.
		els := make(map[eth.ChainID]CanonicalBlockSource, len(clients))
		for chainID, c := range clients {
			els[chainID] = c
		}
		for _, endpoint := range cfg.SupernodeEndpoints {
			rpcClient, err := client.NewRPC(ctx, ms.Log, endpoint)
			if err != nil {
				return fmt.Errorf("failed to dial supernode %q: %w", endpoint, err)
			}
			supernodeClient := sources.NewSuperNodeClient(rpcClient)
			ms.collector.supernodeObservers = append(ms.collector.supernodeObservers,
				NewSupernodeObserver(endpoint, supernodeClient, els, ms.Metrics, ms.Log))
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

	return nil
}

// initClients initializes the clients for the given L2 RPCs
func (ms *InteropMonitorService) initClients(ctx context.Context, l2Rpcs []string) (map[eth.ChainID]*sources.EthClient, error) {
	clients := make(map[eth.ChainID]*sources.EthClient)
	for _, l2Rpc := range l2Rpcs {
		ethClient, err := ms.dial(ctx, l2Rpc)
		if err != nil {
			return nil, fmt.Errorf("failed to dial: %w", err)
		}
		chainIDBig, err := ethClient.ChainID(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get chain ID: %w", err)
		}
		chainID := eth.ChainIDFromBig(chainIDBig)
		clients[chainID] = ethClient
	}
	return clients, nil
}

// dial dials a new client and returns it
func (ms *InteropMonitorService) dial(ctx context.Context, l2Rpc string) (*sources.EthClient, error) {
	client, err := client.NewRPC(ctx, ms.Log, l2Rpc)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	ethClient, err := sources.NewEthClient(client, ms.Log, nil, sources.DefaultEthClientConfig(1000))
	if err != nil {
		return nil, fmt.Errorf("failed to create eth client: %w", err)
	}
	return ethClient, nil
}

// initUpdaters initializes the updaters for the given clients
func (ms *InteropMonitorService) initUpdaters(clients map[eth.ChainID]*sources.EthClient) error {
	for chainID, ethClient := range clients {
		updater := NewUpdater(chainID, ethClient, ms.finalized, ms.messageExpiryWindow, ms.Log)
		ms.updaters[chainID] = updater
	}
	return nil
}

// initFinders initializes the finders for the given clients
func (ms *InteropMonitorService) initFinders(clients map[eth.ChainID]*sources.EthClient) error {
	for chainID, ethClient := range clients {
		finder := NewFinder(chainID, ethClient, BlockReceiptsToJobs, ms.RouteNewJob, ms.SetExpiry, 1000, ms.Log)
		ms.finders[chainID] = finder
	}
	return nil
}

// RouteNewJob routes a new job to the appropriate updater by simply enqueuing to the initiating chain's updater
func (ms *InteropMonitorService) RouteNewJob(job *Job) {
	if updater, ok := ms.updaters[job.initiating.ChainID]; ok {
		updater.Enqueue(job)
	} else {
		ms.Log.Error("no updater found for chain ID", "chain_id", job.initiating.ChainID)
	}
}

// SetExpiry sets the expiry for a chain ID
func (ms *InteropMonitorService) SetExpiry(chainID eth.ChainID, expiry eth.BlockInfo) {
	ms.finalized.Set(chainID, expiry)
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
	if ms.collector != nil {
		err := ms.collector.Start(ctx)
		if err != nil {
			return fmt.Errorf("failed to start metric collector: %w", err)
		}
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

	ms.Log.Info("stopping metric collector")
	if err := ms.collector.Stop(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to stop metric collector: %w", err))
		ms.Log.Error("failed to stop metric collector", "error", err)
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
