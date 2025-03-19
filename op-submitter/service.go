package submitter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/op-proposer/metrics"
	"github.com/ethereum-optimism/optimism/op-proposer/submitter/rpc"
	"github.com/ethereum-optimism/optimism/op-proposer/submitter/source"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

var ErrAlreadyStopped = errors.New("already stopped")

type SubmitterConfig struct {
	PollInterval   time.Duration
	NetworkTimeout time.Duration

	WaitNodeSync bool
}

type SubmitterService struct {
	Log     log.Logger
	Metrics metrics.Metricer

	SubmitterConfig

	TxManager      txmgr.TxManager
	L1Client       *ethclient.Client
	ProposalSource source.ProposalSource

	driver *L2OutputSubmitter

	Version string

	pprofService *oppprof.Service
	metricsSrv   *httputil.HTTPServer
	rpcServer    *oprpc.Server

	balanceMetricer io.Closer

	stopped atomic.Bool
}

// SubmitterServiceFromCLIConfig creates a new SubmitterService from a CLIConfig.
// The service components are fully started, except for the driver,
// which will not be submitting state (if it was configured to) until the Start part of the lifecycle.
func SubmitterServiceFromCLIConfig(ctx context.Context, version string, cfg *CLIConfig, log log.Logger) (*SubmitterService, error) {
	var ps SubmitterService
	if err := ps.initFromCLIConfig(ctx, version, cfg, log); err != nil {
		return nil, errors.Join(err, ps.Stop(ctx)) // try to clean up our failed initialization attempt
	}
	return &ps, nil
}

func (ps *SubmitterService) initFromCLIConfig(ctx context.Context, version string, cfg *CLIConfig, log log.Logger) error {
	ps.Version = version
	ps.Log = log

	ps.initMetrics(cfg)

	ps.PollInterval = cfg.PollInterval
	ps.NetworkTimeout = cfg.TxMgrConfig.NetworkTimeout
	ps.WaitNodeSync = cfg.WaitNodeSync

	ps.initL2nrAddress(cfg)

	if err := ps.initRPCClients(ctx, cfg); err != nil {
		return err
	}
	if err := ps.initTxManager(cfg); err != nil {
		return fmt.Errorf("failed to init Tx manager: %w", err)
	}
	ps.initBalanceMonitor(cfg)
	if err := ps.initMetricsServer(cfg); err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}
	if err := ps.initPProf(cfg); err != nil {
		return fmt.Errorf("failed to init profiling: %w", err)
	}
	if err := ps.initDriver(); err != nil {
		return fmt.Errorf("failed to init Driver: %w", err)
	}
	if err := ps.initRPCServer(cfg); err != nil {
		return fmt.Errorf("failed to start RPC server: %w", err)
	}

	ps.Metrics.RecordInfo(ps.Version)
	ps.Metrics.RecordUp()
	return nil
}

func (ps *SubmitterService) initRPCClients(ctx context.Context, cfg *CLIConfig) error {
	l1Client, err := dial.DialEthClientWithTimeout(ctx, dial.DefaultDialTimeout, ps.Log, cfg.L1EthRpc)
	if err != nil {
		return fmt.Errorf("failed to dial L1 RPC: %w", err)
	}
	ps.L1Client = l1Client

	if cfg.RollupRpc != "" {
		var rollupProvider dial.RollupProvider
		if strings.Contains(cfg.RollupRpc, ",") {
			rollupUrls := strings.Split(cfg.RollupRpc, ",")
			rollupProvider, err = dial.NewActiveL2RollupProvider(ctx, rollupUrls, cfg.ActiveSequencerCheckDuration, dial.DefaultDialTimeout, ps.Log)
		} else {
			rollupProvider, err = dial.NewStaticL2RollupProvider(ctx, ps.Log, cfg.RollupRpc)
		}
		if err != nil {
			return fmt.Errorf("failed to build L2 endpoint provider: %w", err)
		}
		ps.ProposalSource = source.NewRollupProposalSource(rollupProvider)
	}
	if len(cfg.SupervisorRpcs) != 0 {
		var clients []source.SupervisorClient
		for _, url := range cfg.SupervisorRpcs {
			supervisorRpc, err := dial.DialRPCClientWithTimeout(ctx, dial.DefaultDialTimeout, ps.Log, url)
			if err != nil {
				return fmt.Errorf("failed to dial supervisor RPC client (%v): %w", url, err)
			}
			cl := sources.NewSupervisorClient(client.NewBaseRPCClient(supervisorRpc))
			clients = append(clients, cl)
		}
		ps.ProposalSource = source.NewSupervisorProposalSource(ps.Log, clients...)
	}
	return nil
}

func (ps *SubmitterService) initMetrics(cfg *CLIConfig) {
	if cfg.MetricsConfig.Enabled {
		procName := "default"
		ps.Metrics = metrics.NewMetrics(procName)
	} else {
		ps.Metrics = metrics.NoopMetrics
	}
}

// initBalanceMonitor depends on Metrics, L1Client and TxManager to start background-monitoring of the Submitter balance.
func (ps *SubmitterService) initBalanceMonitor(cfg *CLIConfig) {
	if cfg.MetricsConfig.Enabled {
		ps.balanceMetricer = ps.Metrics.StartBalanceMetrics(ps.Log, ps.L1Client, ps.TxManager.From())
	}
}

func (ps *SubmitterService) initTxManager(cfg *CLIConfig) error {
	txManager, err := txmgr.NewSimpleTxManager("proposer", ps.Log, ps.Metrics, cfg.TxMgrConfig)
	if err != nil {
		return err
	}
	ps.TxManager = txManager
	return nil
}

func (ps *SubmitterService) initPProf(cfg *CLIConfig) error {
	ps.pprofService = oppprof.New(
		cfg.PprofConfig.ListenEnabled,
		cfg.PprofConfig.ListenAddr,
		cfg.PprofConfig.ListenPort,
		cfg.PprofConfig.ProfileType,
		cfg.PprofConfig.ProfileDir,
		cfg.PprofConfig.ProfileFilename,
	)

	if err := ps.pprofService.Start(); err != nil {
		return fmt.Errorf("failed to start pprof service: %w", err)
	}

	return nil
}

func (ps *SubmitterService) initMetricsServer(cfg *CLIConfig) error {
	if !cfg.MetricsConfig.Enabled {
		ps.Log.Info("Metrics disabled")
		return nil
	}
	m, ok := ps.Metrics.(opmetrics.RegistryMetricer)
	if !ok {
		return fmt.Errorf("metrics were enabled, but metricer %T does not expose registry for metrics-server", ps.Metrics)
	}
	ps.Log.Debug("Starting metrics server", "addr", cfg.MetricsConfig.ListenAddr, "port", cfg.MetricsConfig.ListenPort)
	metricsSrv, err := opmetrics.StartServer(m.Registry(), cfg.MetricsConfig.ListenAddr, cfg.MetricsConfig.ListenPort)
	if err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}
	ps.Log.Info("Started metrics server", "addr", metricsSrv.Addr())
	ps.metricsSrv = metricsSrv
	return nil
}

func (ps *SubmitterService) initL2nrAddress(cfg *CLIConfig) {
	l2ooAddress, err := opservice.ParseAddress(cfg.L2NRAddress)
	if err != nil {
		// Return no error & set no L2OO related configuration fields.
		return
	}
	ps.L2NativeRollupAddr = &l2ooAddress
}

func (ps *SubmitterService) initDriver() error {
	driver, err := NewL2NativeRollupSubmitter(DriverSetup{
		Log:            ps.Log,
		Metr:           ps.Metrics,
		Cfg:            ps.SubmitterConfig,
		Txmgr:          ps.TxManager,
		L1Client:       ps.L1Client,
		Multicaller:    batching.NewMultiCaller(ps.L1Client.Client(), batching.DefaultBatchSize),
		ProposalSource: ps.ProposalSource,
	})
	if err != nil {
		return err
	}
	ps.driver = driver
	return nil
}

func (ps *SubmitterService) initRPCServer(cfg *CLIConfig) error {
	server := oprpc.NewServer(
		cfg.RPCConfig.ListenAddr,
		cfg.RPCConfig.ListenPort,
		ps.Version,
		oprpc.WithLogger(ps.Log),
	)
	if cfg.RPCConfig.EnableAdmin {
		adminAPI := rpc.NewAdminAPI(ps.driver, ps.Metrics, ps.Log)
		server.AddAPI(rpc.GetAdminAPI(adminAPI))
		server.AddAPI(ps.TxManager.API())
		ps.Log.Info("Admin RPC enabled")
	}
	ps.Log.Info("Starting JSON-RPC server")
	if err := server.Start(); err != nil {
		return fmt.Errorf("unable to start RPC server: %w", err)
	}
	ps.rpcServer = server
	return nil
}

// Start runs once upon start of the submitter lifecycle,
// and starts L2NativeRollup-submission work if the submitter is configured to start submit data on startup.
func (ps *SubmitterService) Start(_ context.Context) error {
	ps.Log.Info("Starting Submitter")
	return ps.driver.StartL2NativeRollupSubmitting()
}

func (ps *SubmitterService) Stopped() bool {
	return ps.stopped.Load()
}

// Kill is a convenience method to forcefully, non-gracefully, stop the SubmitterService.
func (ps *SubmitterService) Kill() error {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ps.Stop(ctx)
}

// Stop fully stops the L2NativeRollup-submitter and all its resources gracefully. After stopping, it cannot be restarted.
// See driver.StopL2NativeRollupSubmitting to temporarily stop the L2NativeRollup submitter.
func (ps *SubmitterService) Stop(ctx context.Context) error {
	if ps.stopped.Load() {
		return ErrAlreadyStopped
	}
	ps.Log.Info("Stopping Submitter")

	var result error
	if ps.driver != nil {
		if err := ps.driver.StopL2NativeRollupSubmittingIfRunning(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop L2NativeRollup submitting: %w", err))
		}
	}

	if ps.rpcServer != nil {
		// TODO(7685): the op-service RPC server is not built on top of op-service httputil Server, and has poor shutdown
		if err := ps.rpcServer.Stop(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop RPC server: %w", err))
		}
	}
	if ps.pprofService != nil {
		if err := ps.pprofService.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop PProf server: %w", err))
		}
	}
	if ps.balanceMetricer != nil {
		if err := ps.balanceMetricer.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close balance metricer: %w", err))
		}
	}

	if ps.TxManager != nil {
		ps.TxManager.Close()
	}

	if ps.metricsSrv != nil {
		if err := ps.metricsSrv.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop metrics server: %w", err))
		}
	}

	if ps.L1Client != nil {
		ps.L1Client.Close()
	}

	if ps.ProposalSource != nil {
		ps.ProposalSource.Close()
	}

	if result == nil {
		ps.stopped.Store(true)
		ps.Log.Info("L2NativeRollup Submitter stopped")
	}

	return result
}

var _ cliapp.Lifecycle = (*SubmitterService)(nil)

// Driver returns the handler on the L2NativeRollup-submitter driver element,
// to start/stop/restart the L2NativeRollup-submission work, for use in testing.
func (ps *SubmitterService) Driver() rpc.SubmitterDriver {
	return ps.driver
}
