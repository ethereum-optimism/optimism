package archiver

import (
	"context"
	"errors"
	"fmt"
	_ "net/http/pprof"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	// "github.com/ethereum-optimism/optimism/op-blob-archiver/metrics"
	"github.com/ethereum-optimism/optimism/op-blob-archiver/rpc"
	"github.com/ethereum-optimism/optimism/op-blob-archiver/storage"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type ArchiverConfig struct {
	PollInterval time.Duration
}

// ArchiverService represents a full blob archiver instance and its resources,
// and conforms to the op-service CLI Lifecycle interface.
type ArchiverService struct {
	Log            log.Logger
	Metrics        metrics.Metricer
	L1Client       *ethclient.Client
	Storage        storage.Storage
	Fetcher        derive.L1TransactionFetcher
	L1BeaconClient *sources.L1BeaconClient
	RollupNode     *sources.RollupClient

	ArchiverConfig

	RollupConfig *rollup.Config

	// Channel builder parameters
	// Channel ChannelConfig

	// driver *BatchSubmitter
	poller *Poller

	Version string

	// pprofSrv   *httputil.HTTPServer
	metricsSrv *httputil.HTTPServer
	rpcServer  *oprpc.Server

	// stopped atomic.Bool
}

// ArchiverServiceFromCLIConfig creates a new ArchiverService from a CLIConfig.
// The service components are fully started.
func ArchiverServiceFromCLIConfig(ctx context.Context, version string, cfg *CLIConfig, log log.Logger) (*ArchiverService, error) {
	var as ArchiverService
	if err := as.initFromCLIConfig(ctx, version, cfg, log); err != nil {
		return nil, errors.Join(err, as.Stop(ctx)) // try to clean up our failed initialization attempt
	}
	return &as, nil
}

func (as *ArchiverService) initFromCLIConfig(ctx context.Context, version string, cfg *CLIConfig, log log.Logger) error {
	as.Version = version
	as.Log = log

	as.initMetrics(cfg)

	as.PollInterval = cfg.PollInterval

	if err := as.initRPCClients(ctx, cfg); err != nil {
		return err
	}
	if err := as.initRollupCfg(ctx); err != nil {
		return fmt.Errorf("failed to load rollup config: %w", err)
	}

	as.poller = NewPoller(
		as.Log,
		as.Storage,
		as.L1BeaconClient,

		// as.Metrics,
		as.L1Client,
		as.RollupConfig,
		as.ArchiverConfig,
		as.RollupNode,
	)

	// as.Metrics.RecordInfo(as.Version)
	// as.Metrics.RecordUp()
	return nil
}

func (as *ArchiverService) initRPCClients(ctx context.Context, cfg *CLIConfig) error {
	l1Client, err := dial.DialEthClientWithTimeout(ctx, dial.DefaultDialTimeout, as.Log, cfg.L1EthRpc)
	if err != nil {
		return fmt.Errorf("failed to dial L1 RPC: %w", err)
	}
	as.L1Client = l1Client

	rollupClient, err := dial.DialRollupClientWithTimeout(ctx, dial.DefaultDialTimeout, as.Log, cfg.RollupRpc)
	if err != nil {
		return fmt.Errorf("failed to dial L2 rollup-client RPC: %w", err)
	}
	as.RollupNode = rollupClient
	return nil
}

func (as *ArchiverService) initMetrics(cfg *CLIConfig) {
	if cfg.MetricsConfig.Enabled {
		procName := "default"
		as.Metrics = metrics.NewMetrics(procName)
	} else {
		as.Metrics = metrics.NoopMetrics
	}
}

func (as *ArchiverService) initRollupCfg(ctx context.Context) error {
	rollupCfg, err := as.RollupNode.RollupConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve rollup config: %w", err)
	}
	as.RollupConfig = rollupCfg
	if err := as.RollupConfig.Check(); err != nil {
		return fmt.Errorf("invalid rollup config: %w", err)
	}
	return nil
}

// func (as *ArchiverService) initPProf(cfg *CLIConfig) error {
// 	if !cfg.PprofConfig.Enabled {
// 		return nil
// 	}
// 	log.Debug("starting pprof server", "addr", net.JoinHostPort(cfg.PprofConfig.ListenAddr, strconv.Itoa(cfg.PprofConfig.ListenPort)))
// 	srv, err := oppprof.StartServer(cfg.PprofConfig.ListenAddr, cfg.PprofConfig.ListenPort)
// 	if err != nil {
// 		return err
// 	}
// 	as.pprofSrv = srv
// 	log.Info("started pprof server", "addr", srv.Addr())
// 	return nil
// }

func (as *ArchiverService) initMetricsServer(cfg *CLIConfig) error {
	if !cfg.MetricsConfig.Enabled {
		as.Log.Info("metrics disabled")
		return nil
	}
	m, ok := as.Metrics.(opmetrics.RegistryMetricer)
	if !ok {
		return fmt.Errorf("metrics were enabled, but metricer %T does not expose registry for metrics-server", as.Metrics)
	}
	as.Log.Debug("starting metrics server", "addr", cfg.MetricsConfig.ListenAddr, "port", cfg.MetricsConfig.ListenPort)
	metricsSrv, err := opmetrics.StartServer(m.Registry(), cfg.MetricsConfig.ListenAddr, cfg.MetricsConfig.ListenPort)
	if err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}
	as.Log.Info("started metrics server", "addr", metricsSrv.Addr())
	as.metricsSrv = metricsSrv
	return nil
}

// Start runs once upon start of the batcher lifecycle,
// and starts batch-submission work if the batcher is configured to start submit data on startup.
func (as *ArchiverService) Start(_ context.Context) error {
	as.driver.Log.Info("Starting batcher", "notSubmittingOnStart", as.NotSubmittingOnStart)

	if !as.NotSubmittingOnStart {
		return as.driver.StartBatchSubmitting()
	}
	return nil
}

// Stopped returns if the service as a whole is stopped.
func (as *ArchiverService) Stopped() bool {
	return as.stopped.Load()
}

// Kill is a convenience method to forcefully, non-gracefully, stop the ArchiverService.
func (as *ArchiverService) Kill() error {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return as.Stop(ctx)
}

// Stop fully stops the batch-submitter and all its resources gracefully. After stopping, it cannot be restarted.
// See driver.StopBatchSubmitting to temporarily stop the batch submitter.
// If the provided ctx is cancelled, the stopping is forced, i.e. the batching work is killed non-gracefully.
func (as *ArchiverService) Stop(ctx context.Context) error {
	if as.stopped.Load() {
		return errors.New("already stopped")
	}
	as.Log.Info("Stopping batcher")

	var result error
	if as.driver != nil {
		if err := as.driver.StopBatchSubmittingIfRunning(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop batch submitting: %w", err))
		}
	}

	if as.rpcServer != nil {
		// TODO(7685): the op-service RPC server is not built on top of op-service httputil Server, and has poor shutdown
		if err := as.rpcServer.Stop(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop RPC server: %w", err))
		}
	}
	if as.pprofSrv != nil {
		if err := as.pprofSrv.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop PProf server: %w", err))
		}
	}
	if as.balanceMetricer != nil {
		if err := as.balanceMetricer.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close balance metricer: %w", err))
		}
	}
	if as.metricsSrv != nil {
		if err := as.metricsSrv.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop metrics server: %w", err))
		}
	}

	if as.L1Client != nil {
		as.L1Client.Close()
	}

	if as.RollupNode != nil {
		as.RollupNode.Close()
	}

	if result == nil {
		as.stopped.Store(true)
		as.Log.Info("Batch Submitter stopped")
	}
	return result
}

var _ cliapp.Lifecycle = (*ArchiverService)(nil)

// Driver returns the handler on the batch-submitter driver element,
// to start/stop/restart the batch-submission work, for use in testing.
func (as *ArchiverService) Driver() rpc.BatcherDriver {
	return as.driver
}
