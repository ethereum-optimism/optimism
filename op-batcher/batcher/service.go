package batcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	_ "net/http/pprof"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-batcher/rpc"
	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	plasma "github.com/ethereum-optimism/optimism/op-plasma"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

var ErrAlreadyStopped = errors.New("already stopped")

type BatcherConfig struct {
	NetworkTimeout         time.Duration
	PollInterval           time.Duration
	MaxPendingTransactions uint64

	// UseBlobs is true if the batcher should use blobs instead of calldata for posting blobs
	UseBlobs bool

	// UsePlasma is true if the rollup config has a DA challenge address so the batcher
	// will post inputs to the Plasma DA server and post commitments to blobs or calldata.
	UsePlasma bool

	WaitNodeSync        bool
	CheckRecentTxsDepth int
}

// BatcherService represents a full batch-submitter instance and its resources,
// and conforms to the op-service CLI Lifecycle interface.
type BatcherService struct {
	Log              log.Logger
	Metrics          metrics.Metricer
	L1Client         *ethclient.Client
	EndpointProvider dial.L2EndpointProvider
	TxManager        txmgr.TxManager
	PlasmaDA         *plasma.DAClient

	BatcherConfig

	RollupConfig *rollup.Config

	// Channel builder parameters
	ChannelConfig ChannelConfig

	driver *BatchSubmitter

	Version string

	pprofService *oppprof.Service
	metricsSrv   *httputil.HTTPServer
	rpcServer    *oprpc.Server

	balanceMetricer io.Closer
	stopped         atomic.Bool

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
	bs.CheckRecentTxsDepth = cfg.CheckRecentTxsDepth
	bs.WaitNodeSync = cfg.WaitNodeSync
	if err := bs.initRPCClients(ctx, cfg); err != nil {
		return err
	}
	if err := bs.initRollupConfig(ctx); err != nil {
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
		return fmt.Errorf("failed to init profiling: %w", err)
	}
	// init before driver
	if err := bs.initPlasmaDA(cfg); err != nil {
		return fmt.Errorf("failed to init plasma DA: %w", err)
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

	var endpointProvider dial.L2EndpointProvider
	if strings.Contains(cfg.RollupRpc, ",") && strings.Contains(cfg.L2EthRpc, ",") {
		rollupUrls := strings.Split(cfg.RollupRpc, ",")
		ethUrls := strings.Split(cfg.L2EthRpc, ",")
		endpointProvider, err = dial.NewActiveL2EndpointProvider(ctx, ethUrls, rollupUrls, cfg.ActiveSequencerCheckDuration, dial.DefaultDialTimeout, bs.Log)
	} else {
		endpointProvider, err = dial.NewStaticL2EndpointProvider(ctx, bs.Log, cfg.L2EthRpc, cfg.RollupRpc)
	}
	if err != nil {
		return fmt.Errorf("failed to build L2 endpoint provider: %w", err)
	}
	bs.EndpointProvider = endpointProvider

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

func (bs *BatcherService) initRollupConfig(ctx context.Context) error {
	rollupNode, err := bs.EndpointProvider.RollupClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve rollup client: %w", err)
	}
	rollupConfig, err := rollupNode.RollupConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve rollup config: %w", err)
	}
	bs.RollupConfig = rollupConfig
	if err := bs.RollupConfig.Check(); err != nil {
		return fmt.Errorf("invalid rollup config: %w", err)
	}
	bs.RollupConfig.LogDescription(bs.Log, chaincfg.L2ChainIDToNetworkDisplayName)
	return nil
}

func (bs *BatcherService) initChannelConfig(cfg *CLIConfig) error {
	cc := ChannelConfig{
		SeqWindowSize:      bs.RollupConfig.SeqWindowSize,
		ChannelTimeout:     bs.RollupConfig.ChannelTimeout,
		MaxChannelDuration: cfg.MaxChannelDuration,
		MaxFrameSize:       cfg.MaxL1TxSize - 1, // account for version byte prefix; reset for blobs
		TargetNumFrames:    cfg.TargetNumFrames,
		SubSafetyMargin:    cfg.SubSafetyMargin,
		BatchType:          cfg.BatchType,
	}

	switch cfg.DataAvailabilityType {
	case flags.BlobsType:
		if !cfg.TestUseMaxTxSizeForBlobs {
			// account for version byte prefix
			cc.MaxFrameSize = eth.MaxBlobDataSize - 1
		}
		cc.MultiFrameTxs = true
		bs.UseBlobs = true
	case flags.CalldataType:
		bs.UseBlobs = false
	default:
		return fmt.Errorf("unknown data availability type: %v", cfg.DataAvailabilityType)
	}

	if bs.UsePlasma && cc.MaxFrameSize > plasma.MaxInputSize {
		return fmt.Errorf("max frame size %d exceeds plasma max input size %d", cc.MaxFrameSize, plasma.MaxInputSize)
	}

	cc.InitCompressorConfig(cfg.ApproxComprRatio, cfg.Compressor, cfg.CompressionAlgo)

	if bs.UseBlobs && !bs.RollupConfig.IsEcotone(uint64(time.Now().Unix())) {
		bs.Log.Error("Cannot use Blob data before Ecotone!") // log only, the batcher may not be actively running.
	}
	if !bs.UseBlobs && bs.RollupConfig.IsEcotone(uint64(time.Now().Unix())) {
		bs.Log.Warn("Ecotone upgrade is active, but batcher is not configured to use Blobs!")
	}

	// Checking for brotli compression only post Fjord
	if bs.ChannelConfig.CompressorConfig.CompressionAlgo.IsBrotli() && !bs.RollupConfig.IsFjord(uint64(time.Now().Unix())) {
		return fmt.Errorf("cannot use brotli compression before Fjord")
	}

	if err := cc.Check(); err != nil {
		return fmt.Errorf("invalid channel configuration: %w", err)
	}
	bs.Log.Info("Initialized channel-config",
		"use_blobs", bs.UseBlobs,
		"use_plasma", bs.UsePlasma,
		"max_frame_size", cc.MaxFrameSize,
		"target_num_frames", cc.TargetNumFrames,
		"compressor", cc.CompressorConfig.Kind,
		"max_channel_duration", cc.MaxChannelDuration,
		"channel_timeout", cc.ChannelTimeout,
		"batch_type", cc.BatchType,
		"sub_safety_margin", cc.SubSafetyMargin)
	bs.ChannelConfig = cc
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
	bs.pprofService = oppprof.New(
		cfg.PprofConfig.ListenEnabled,
		cfg.PprofConfig.ListenAddr,
		cfg.PprofConfig.ListenPort,
		cfg.PprofConfig.ProfileType,
		cfg.PprofConfig.ProfileDir,
		cfg.PprofConfig.ProfileFilename,
	)

	if err := bs.pprofService.Start(); err != nil {
		return fmt.Errorf("failed to start pprof service: %w", err)
	}

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
		Log:              bs.Log,
		Metr:             bs.Metrics,
		RollupConfig:     bs.RollupConfig,
		Config:           bs.BatcherConfig,
		Txmgr:            bs.TxManager,
		L1Client:         bs.L1Client,
		EndpointProvider: bs.EndpointProvider,
		ChannelConfig:    bs.ChannelConfig,
		PlasmaDA:         bs.PlasmaDA,
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

func (bs *BatcherService) initPlasmaDA(cfg *CLIConfig) error {
	config := cfg.PlasmaDA
	if err := config.Check(); err != nil {
		return err
	}
	bs.PlasmaDA = config.NewDAClient()
	bs.UsePlasma = config.Enabled
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
		return ErrAlreadyStopped
	}
	bs.Log.Info("Stopping batcher")

	// close the TxManager first, so that new work is denied, in-flight work is cancelled as early as possible
	// (transactions which are expected to be confirmed are still waited for)
	if bs.TxManager != nil {
		bs.TxManager.Close()
	}

	var result error
	if bs.driver != nil {
		if err := bs.driver.StopBatchSubmittingIfRunning(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop batch submitting: %w", err))
		}
	}

	if bs.rpcServer != nil {
		// TODO(7685): the op-service RPC server is not built on top of op-service httputil Server, and has poor shutdown
		if err := bs.rpcServer.Stop(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop RPC server: %w", err))
		}
	}
	if bs.pprofService != nil {
		if err := bs.pprofService.Stop(ctx); err != nil {
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
	if bs.EndpointProvider != nil {
		bs.EndpointProvider.Close()
	}

	if result == nil {
		bs.stopped.Store(true)
		bs.Log.Info("Batch Submitter stopped")
	}
	return result
}

var _ cliapp.Lifecycle = (*BatcherService)(nil)

// Driver returns the handler on the batch-submitter driver element,
// to start/stop/restart the batch-submission work, for use in testing.
func (bs *BatcherService) Driver() rpc.BatcherDriver {
	return bs.driver
}
