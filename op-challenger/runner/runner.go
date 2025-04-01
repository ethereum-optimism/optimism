package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	trace "github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
)

var (
	ErrUnexpectedStatusCode = errors.New("unexpected status code")
)

type Metricer interface {
	contractMetrics.ContractMetricer
	metrics.VmMetricer
	opmetrics.RPCClientMetricer

	RecordFailure(vmType string)
	RecordPanic(vmType string)
	RecordInvalid(vmType string)
	RecordSuccess(vmType string)
}

type RunConfig struct {
	TraceType        types.TraceType
	Name             string
	Prestate         common.Hash
	PrestateFilename string
}

type Runner struct {
	log        log.Logger
	cfg        *config.Config
	runConfigs []RunConfig
	m          Metricer

	running    atomic.Bool
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	metricsSrv *httputil.HTTPServer
}

func NewRunner(logger log.Logger, cfg *config.Config, runConfigs []RunConfig) *Runner {
	return &Runner{
		log:        logger,
		cfg:        cfg,
		runConfigs: runConfigs,
		m:          NewMetrics(runConfigs),
	}
}

func (r *Runner) Start(ctx context.Context) error {
	if !r.running.CompareAndSwap(false, true) {
		return errors.New("already started")
	}
	ctx, cancel := context.WithCancel(ctx)
	r.ctx = ctx
	r.cancel = cancel
	if err := r.initMetricsServer(&r.cfg.MetricsConfig); err != nil {
		return fmt.Errorf("failed to start metrics: %w", err)
	}

	var rollupClient *sources.RollupClient
	if r.cfg.RollupRpc != "" {
		r.log.Info("Dialling rollup client", "url", r.cfg.RollupRpc)
		cl, err := dial.DialRollupClientWithTimeout(ctx, 1*time.Minute, r.log, r.cfg.RollupRpc)
		if err != nil {
			return fmt.Errorf("failed to dial rollup client: %w", err)
		}
		rollupClient = cl
	}
	var supervisorClient *sources.SupervisorClient
	if r.cfg.SupervisorRPC != "" {
		r.log.Info("Dialling supervisor client", "url", r.cfg.SupervisorRPC)
		rpcCl, err := dial.DialRPCClientWithTimeout(ctx, 1*time.Minute, r.log, r.cfg.SupervisorRPC)
		if err != nil {
			return fmt.Errorf("failed to dial rollup client: %w", err)
		}
		supervisorClient = sources.NewSupervisorClient(client.NewBaseRPCClient(rpcCl), r.m)
	}

	l1Client, err := dial.DialRPCClientWithTimeout(ctx, 1*time.Minute, r.log, r.cfg.L1EthRpc)
	if err != nil {
		return fmt.Errorf("failed to dial l1 client: %w", err)
	}
	caller := batching.NewMultiCaller(l1Client, batching.DefaultBatchSize)

	for _, runConfig := range r.runConfigs {
		r.wg.Add(1)
		go r.loop(ctx, runConfig, rollupClient, supervisorClient, caller)
	}

	r.log.Info("Runners started", "num", len(r.runConfigs))
	return nil
}

func (r *Runner) loop(ctx context.Context, runConfig RunConfig, rollupClient *sources.RollupClient, supervisorClient *sources.SupervisorClient, caller *batching.MultiCaller) {
	defer r.wg.Done()
	t := time.NewTicker(1 * time.Minute)
	defer t.Stop()
	for {
		r.runAndRecordOnce(ctx, runConfig, rollupClient, supervisorClient, caller)
		select {
		case <-t.C:
		case <-ctx.Done():
			return
		}
	}
}

func (r *Runner) runAndRecordOnce(ctx context.Context, runConfig RunConfig, rollupClient *sources.RollupClient, supervisorClient *sources.SupervisorClient, caller *batching.MultiCaller) {
	recordError := func(err error, traceType string, m Metricer, log log.Logger) {
		if errors.Is(err, ErrUnexpectedStatusCode) {
			log.Error("Incorrect status code", "type", runConfig.Name, "err", err)
			m.RecordInvalid(traceType)
		} else if errors.Is(err, trace.ErrVMPanic) {
			log.Error("VM panicked", "type", runConfig.Name)
			m.RecordPanic(traceType)
		} else if err != nil {
			log.Error("Failed to run", "type", runConfig.Name, "err", err)
			m.RecordFailure(traceType)
		} else {
			log.Info("Successfully verified output root", "type", runConfig.Name)
			m.RecordSuccess(traceType)
		}
	}

	var prestateSource prestateFetcher
	if strings.HasPrefix(runConfig.PrestateFilename, "file:") {
		path := runConfig.PrestateFilename[len("file:"):]
		r.log.Info("Using local file prestate", "type", runConfig.TraceType, "path", path)
		prestateSource = &LocalPrestateFetcher{path: path}
	} else if runConfig.PrestateFilename != "" {
		r.log.Info("Using named prestate", "type", runConfig.TraceType, "filename", runConfig.PrestateFilename)
		prestateSource = &NamedPrestateFetcher{filename: runConfig.PrestateFilename}
	} else if runConfig.Prestate == (common.Hash{}) {
		r.log.Info("Using on chain prestate", "type", runConfig.TraceType)
		prestateSource = &OnChainPrestateFetcher{
			m:                  r.m,
			gameFactoryAddress: r.cfg.GameFactoryAddress,
			gameType:           runConfig.TraceType.GameType(),
			caller:             caller,
		}
	} else {
		r.log.Info("Using specific prestate", "type", runConfig.TraceType, "hash", runConfig.Prestate)
		prestateSource = &HashPrestateFetcher{prestateHash: runConfig.Prestate}
	}

	localInputs, err := createGameInputs(ctx, r.log, rollupClient, supervisorClient, runConfig.Name, runConfig.TraceType)
	if err != nil {
		recordError(err, runConfig.Name, r.m, r.log)
		return
	}

	inputsLogger := r.log.New("l1", localInputs.L1Head, "l2", localInputs.L2Head, "l2Block", localInputs.L2SequenceNumber, "claim", localInputs.L2Claim)
	// Sanitize the directory name.
	safeName := regexp.MustCompile("[^a-zA-Z0-9_-]").ReplaceAllString(runConfig.Name, "")
	dir, err := r.prepDatadir(safeName)
	if err != nil {
		recordError(err, runConfig.Name, r.m, r.log)
		return
	}
	err = r.runOnce(ctx, inputsLogger.With("type", runConfig.Name), runConfig.Name, runConfig.TraceType, prestateSource, localInputs, dir)
	recordError(err, runConfig.Name, r.m, r.log)
}

func (r *Runner) runOnce(ctx context.Context, logger log.Logger, name string, traceType types.TraceType, prestateSource prestateFetcher, localInputs utils.LocalGameInputs, dir string) error {
	provider, err := createTraceProvider(ctx, logger, metrics.NewTypedVmMetrics(r.m, name), r.cfg, prestateSource, traceType, localInputs, dir)
	if err != nil {
		return fmt.Errorf("failed to create trace provider: %w", err)
	}
	hash, err := provider.Get(ctx, types.RootPosition)
	if err != nil {
		return fmt.Errorf("failed to execute trace provider: %w", err)
	}
	if hash[0] != mipsevm.VMStatusValid {
		return fmt.Errorf("%w: %v", ErrUnexpectedStatusCode, hash)
	}
	return nil
}

func (r *Runner) prepDatadir(name string) (string, error) {
	dir := filepath.Join(r.cfg.Datadir, name)
	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("failed to remove old dir: %w", err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create data dir (%v): %w", dir, err)
	}
	return dir, nil
}

func (r *Runner) Stop(ctx context.Context) error {
	r.log.Info("Stopping")
	if !r.running.CompareAndSwap(true, false) {
		return errors.New("not started")
	}
	r.cancel()
	r.wg.Wait()

	if r.metricsSrv != nil {
		return r.metricsSrv.Stop(ctx)
	}
	return nil
}

func (r *Runner) Stopped() bool {
	return !r.running.Load()
}

func (r *Runner) initMetricsServer(cfg *opmetrics.CLIConfig) error {
	if !cfg.Enabled {
		return nil
	}
	r.log.Debug("Starting metrics server", "addr", cfg.ListenAddr, "port", cfg.ListenPort)
	m, ok := r.m.(opmetrics.RegistryMetricer)
	if !ok {
		return fmt.Errorf("metrics were enabled, but metricer %T does not expose registry for metrics-server", r.m)
	}
	metricsSrv, err := opmetrics.StartServer(m.Registry(), cfg.ListenAddr, cfg.ListenPort)
	if err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}
	r.log.Info("started metrics server", "addr", metricsSrv.Addr())
	r.metricsSrv = metricsSrv
	return nil
}

var _ cliapp.Lifecycle = (*Runner)(nil)
