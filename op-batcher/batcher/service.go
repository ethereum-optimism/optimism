package batcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	_ "net/http/pprof"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-batcher/rpc"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

type BatcherConfig struct {
	NetworkTimeout         time.Duration
	PollInterval           time.Duration
	MaxPendingTransactions uint64
}

// BatcherService represents a full batch-submitter instance and its resources,
// and conforms to the op-service CLI Lifecycle interface.
type BatcherService struct {
	Log        log.Logger
	Metrics    metrics.Metricer
	L1Client   *ethclient.Client
	L2Client   *ethclient.Client
	RollupNode *sources.RollupClient
	TxManager  txmgr.TxManager

	BatcherConfig

	RollupConfig *rollup.Config

	// Channel builder parameters
	Channel ChannelConfig

	driver *BatchSubmitter

	Version string

	pprofSrv   *httputil.HTTPServer
	metricsSrv *httputil.HTTPServer
	rpcServer  *oprpc.Server

	balanceMetricer io.Closer

	stopped atomic.Bool

	NotSubmittingOnStart bool
}

// BatcherServiceFromCLIConfig creates a new BatcherService from a CLIConfig.
// The service components are fully started, except for the driver,
// which will not be submitting batches (if it was configured to) until the Start part of the lifecycle.
func BatcherServiceFromCLIConfig(ctx context.Context, version string, cfg *CLIConfig, log log.Logger) (*BatcherService, error) {
	var bs BatcherService
	if err := bs.initFromCLIConfig(ctx, version, cfg, log); err != nil {
		return nil, errors.Join(err, bs.Stop(ctx)) // try to clean up our failed initialization attempt
	}
	return &bs, nil
}

func (bs *BatcherService) initFromCLIConfig(ctx context.Context, version string, cfg *CLIConfig, log log.Logger) error {
	bs.Version = version
	bs.Log = log
	bs.NotSubmittingOnStart = cfg.Stopped

	bs.initMetrics(cfg)

	bs.PollInterval = cfg.PollInterval
	bs.MaxPendingTransactions = cfg.MaxPendingTransactions
	bs.NetworkTimeout = cfg.TxMgrConfig.NetworkTimeout

	if err := bs.initRPCClients(ctx, cfg); err != nil {
		return err
	}
	if err := bs.initRollupCfg(ctx); err != nil {
		return fmt.Errorf("failed to load rollup config: %w", err)
	}
	if err := bs.initChannelConfig(cfg); err != nil {
		return fmt.Errorf("failed to init channel config: %w", err)
	}
	if err := bs.initTxManager(cfg); err != nil {
		return fmt.Errorf("failed to init Tx manager: %w", err)
	}
	bs.initBalanceMonitor(cfg)
	if err := bs.initMetricsServer(cfg); err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}
	if err := bs.initPProf(cfg); err != nil {
		return fmt.Errorf("failed to start pprof server: %w", err)
	}
	bs.initDriver()
	if err := bs.initRPCServer(cfg); err != nil {
		return fmt.Errorf("failed to start RPC server: %w", err)
	}

	bs.Metrics.RecordInfo(bs.Version)
	bs.Metrics.RecordUp()
	return nil
}

func (bs *BatcherService) initRPCClients(ctx context.Context, cfg *CLIConfig) error {
	l1Client, err := dial.DialEthClientWithTimeout(ctx, dial.DefaultDialTimeout, bs.Log, cfg.L1EthRpc)
	if err != nil {
		return fmt.Errorf("failed to dial L1 RPC: %w", err)
	}
	bs.L1Client = l1Client

	l2Client, err := dial.DialEthClientWithTimeout(ctx, dial.DefaultDialTimeout, bs.Log, cfg.L2EthRpc)
	if err != nil {
		return fmt.Errorf("failed to dial L2 engine RPC: %w", err)
	}
	bs.L2Client = l2Client

	rollupClient, err := dial.DialRollupClientWithTimeout(ctx, dial.DefaultDialTimeout, bs.Log, cfg.RollupRpc)
	if err != nil {
		return fmt.Errorf("failed to dial L2 rollup-client RPC: %w", err)
	}
	bs.RollupNode = rollupClient
	return nil
}

func (bs *BatcherService) initMetrics(cfg *CLIConfig) {
	if cfg.MetricsConfig.Enabled {
		procName := "default"
		bs.Metrics = metrics.NewMetrics(procName)
	} else {
		bs.Metrics = metrics.NoopMetrics
	}
}

// initBalanceMonitor depends on Metrics, L1Client and TxManager to start background-monitoring of the batcher balance.
func (bs *BatcherService) initBalanceMonitor(cfg *CLIConfig) {
	if cfg.MetricsConfig.Enabled {
		bs.balanceMetricer = bs.Metrics.StartBalanceMetrics(bs.Log, bs.L1Client, bs.TxManager.From())
	}
}

func (bs *BatcherService) initRollupCfg(ctx context.Context) error {
	rollupCfg, err := bs.RollupNode.RollupConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve rollup config: %w", err)
	}
	bs.RollupConfig = rollupCfg
	if err := bs.RollupConfig.Check(); err != nil {
		return fmt.Errorf("invalid rollup config: %w", err)
	}
	return nil
}

func (bs *BatcherService) initChannelConfig(cfg *CLIConfig) error {
	bs.Channel = ChannelConfig{
		SeqWindowSize:      bs.RollupConfig.SeqWindowSize,
		ChannelTimeout:     bs.RollupConfig.ChannelTimeout,
		MaxChannelDuration: cfg.MaxChannelDuration,
		SubSafetyMargin:    cfg.SubSafetyMargin,
		MaxFrameSize:       cfg.MaxL1TxSize - 1, // subtract 1 byte for version
		CompressorConfig:   cfg.CompressorConfig.Config(),
	}
	if err := bs.Channel.Check(); err != nil {
		return fmt.Errorf("invalid channel configuration: %w", err)
	}
	return nil
}

func (bs *BatcherService) initTxManager(cfg *CLIConfig) error {
	txManager, err := txmgr.NewSimpleTxManager("batcher", bs.Log, bs.Metrics, cfg.TxMgrConfig)
	if err != nil {
		return err
	}
	bs.TxManager = txManager
	return nil
}

func (bs *BatcherService) initPProf(cfg *CLIConfig) error {
	if !cfg.PprofConfig.Enabled {
		return nil
	}
	log.Debug("starting pprof server", "addr", net.JoinHostPort(cfg.PprofConfig.ListenAddr, strconv.Itoa(cfg.PprofConfig.ListenPort)))
	srv, err := oppprof.StartServer(cfg.PprofConfig.ListenAddr, cfg.PprofConfig.ListenPort)
	if err != nil {
		return err
	}
	bs.pprofSrv = srv
	log.Info("started pprof server", "addr", srv.Addr())
	return nil
}

func (bs *BatcherService) initMetricsServer(cfg *CLIConfig) error {
	if !cfg.MetricsConfig.Enabled {
		bs.Log.Info("metrics disabled")
		return nil
	}
	m, ok := bs.Metrics.(opmetrics.RegistryMetricer)
	if !ok {
		return fmt.Errorf("metrics were enabled, but metricer %T does not expose registry for metrics-server", bs.Metrics)
	}
	bs.Log.Debug("starting metrics server", "addr", cfg.MetricsConfig.ListenAddr, "port", cfg.MetricsConfig.ListenPort)
	metricsSrv, err := opmetrics.StartServer(m.Registry(), cfg.MetricsConfig.ListenAddr, cfg.MetricsConfig.ListenPort)
	if err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}
	bs.Log.Info("started metrics server", "addr", metricsSrv.Addr())
	bs.metricsSrv = metricsSrv
	return nil
}

func (bs *BatcherService) initDriver() {
	bs.driver = NewBatchSubmitter(DriverSetup{
		Log:          bs.Log,
		Metr:         bs.Metrics,
		RollupCfg:    bs.RollupConfig,
		Cfg:          bs.BatcherConfig,
		Txmgr:        bs.TxManager,
		L1Client:     bs.L1Client,
		L2Client:     bs.L2Client,
		RollupClient: bs.RollupNode,
		Channel:      bs.Channel,
	})
}

func (bs *BatcherService) initRPCServer(cfg *CLIConfig) error {
	server := oprpc.NewServer(
		cfg.RPC.ListenAddr,
		cfg.RPC.ListenPort,
		bs.Version,
		oprpc.WithLogger(bs.Log),
	)
	if cfg.RPC.EnableAdmin {
		adminAPI := rpc.NewAdminAPI(bs.driver, bs.Metrics, bs.Log)
		server.AddAPI(rpc.GetAdminAPI(adminAPI))
		bs.Log.Info("Admin RPC enabled")
	}
	bs.Log.Info("Starting JSON-RPC server")
	if err := server.Start(); err != nil {
		return fmt.Errorf("unable to start RPC server: %w", err)
	}
	bs.rpcServer = server
	return nil
}

// Start runs once upon start of the batcher lifecycle,
// and starts batch-submission work if the batcher is configured to start submit data on startup.
func (bs *BatcherService) Start(_ context.Context) error {
	bs.driver.Log.Info("Starting batcher", "notSubmittingOnStart", bs.NotSubmittingOnStart)

	if !bs.NotSubmittingOnStart {
		return bs.driver.StartBatchSubmitting()
	}
	return nil
}

// Stopped returns if the service as a whole is stopped.
func (bs *BatcherService) Stopped() bool {
	return bs.stopped.Load()
}

// Kill is a convenience method to forcefully, non-gracefully, stop the BatcherService.
func (bs *BatcherService) Kill() error {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return bs.Stop(ctx)
}

// Stop fully stops the batch-submitter and all its resources gracefully. After stopping, it cannot be restarted.
// See driver.StopBatchSubmitting to temporarily stop the batch submitter.
// If the provided ctx is cancelled, the stopping is forced, i.e. the batching work is killed non-gracefully.
func (bs *BatcherService) Stop(ctx context.Context) error {
	if bs.stopped.Load() {
		return errors.New("already stopped")
	}
	bs.Log.Info("Stopping batcher")

	var result error
	if err := bs.driver.StopBatchSubmittingIfRunning(ctx); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to stop batch submitting: %w", err))
	}

	if bs.rpcServer != nil {
		// TODO(7685): the op-service RPC server is not built on top of op-service httputil Server, and has poor shutdown
		if err := bs.rpcServer.Stop(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop RPC server: %w", err))
		}
	}
	if bs.pprofSrv != nil {
		if err := bs.pprofSrv.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop PProf server: %w", err))
		}
	}
	if bs.balanceMetricer != nil {
		if err := bs.balanceMetricer.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close balance metricer: %w", err))
		}
	}
	if bs.metricsSrv != nil {
		if err := bs.metricsSrv.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop metrics server: %w", err))
		}
	}

	if bs.L1Client != nil {
		bs.L1Client.Close()
	}
	if bs.L2Client != nil {
		bs.L2Client.Close()
	}
	if bs.RollupNode != nil {
		bs.RollupNode.Close()
	}

	if result == nil {
		bs.stopped.Store(true)
		bs.driver.Log.Info("Batch Submitter stopped")
	}
	return result
}

var _ cliapp.Lifecycle = (*BatcherService)(nil)

// Driver returns the handler on the batch-submitter driver element,
// to start/stop/restart the batch-submission work, for use in testing.
func (bs *BatcherService) Driver() rpc.BatcherDriver {
	return bs.driver
}
