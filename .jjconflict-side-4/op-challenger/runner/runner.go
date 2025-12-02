package runner

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
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
	ErrVMTimeout            = errors.New("VM execution timed out")
)

type Metricer interface {
	contractMetrics.ContractMetricer
	metrics.VmMetricer
	opmetrics.RPCMetricer

	RecordSetupFailure(vmType string)
	RecordVmFailure(vmType string, reason string)
	RecordSuccess(vmType string)
}

type RunConfig struct {
	GameType         gameTypes.GameType
	Name             string
	Prestate         common.Hash
	PrestateFilename string
}

type TraceProviderCreator func(
	ctx context.Context,
	logger log.Logger,
	m trace.Metricer,
	cfg *config.Config,
	prestateSource prestateFetcher,
	gameType gameTypes.GameType,
	localInputs utils.LocalGameInputs,
	dir string,
) (types.TraceProvider, error)

type Runner struct {
	log                  log.Logger
	cfg                  *config.Config
	runConfigs           []RunConfig
	m                    Metricer
	vmTimeout            time.Duration
	traceProviderCreator TraceProviderCreator

	running    atomic.Bool
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	metricsSrv *httputil.HTTPServer
}

func NewRunner(logger log.Logger, cfg *config.Config, runConfigs []RunConfig, vmTimeout time.Duration) *Runner {
	return &Runner{
		log:                  logger,
		cfg:                  cfg,
		runConfigs:           runConfigs,
		m:                    NewMetrics(runConfigs),
		vmTimeout:            vmTimeout,
		traceProviderCreator: createTraceProvider,
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
		cl, err := dial.DialRollupClientWithTimeout(ctx, r.log, r.cfg.RollupRpc)
		if err != nil {
			return fmt.Errorf("failed to dial rollup client: %w", err)
		}
		rollupClient = cl
	}
	var supervisorClient *sources.SupervisorClient
	if r.cfg.SuperRPC != "" {
		r.log.Info("Dialling supervisor client", "url", r.cfg.SuperRPC)
		cl, err := dial.DialSupervisorClientWithTimeout(ctx, r.log, r.cfg.SuperRPC)
		if err != nil {
			return fmt.Errorf("failed to dial supervisor: %w", err)
		}
		supervisorClient = cl
	}

	l1Client, err := dial.DialRPCClientWithTimeout(ctx, r.log, r.cfg.L1EthRpc)
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
		baseLog := r.log.New("run_id", generateRunID())
		r.runAndRecordOnce(ctx, baseLog, runConfig, rollupClient, supervisorClient, caller)
		select {
		case <-t.C:
		case <-ctx.Done():
			return
		}
	}
}

func (r *Runner) runAndRecordOnce(ctx context.Context, rlog log.Logger, runConfig RunConfig, rollupClient *sources.RollupClient, supervisorClient *sources.SupervisorClient, caller *batching.MultiCaller) {
	recordError := func(err error, configName string, m Metricer, log log.Logger) {
		if errors.Is(err, ErrUnexpectedStatusCode) {
			log.Error("Incorrect status code", "type", runConfig.Name, "err", err)
			m.RecordVmFailure(configName, ReasonIncorrectStatus)
		} else if errors.Is(err, trace.ErrVMPanic) {
			log.Error("VM panicked", "type", runConfig.Name)
			m.RecordVmFailure(configName, ReasonPanic)
		} else if errors.Is(err, ErrVMTimeout) {
			log.Error("VM execution timed out", "type", runConfig.Name, "timeout", r.vmTimeout)
			m.RecordVmFailure(configName, ReasonTimeout)
		} else if err != nil {
			log.Error("Failed to run", "type", runConfig.Name, "err", err)
			m.RecordSetupFailure(configName)
		} else {
			log.Info("Successfully verified output root", "type", runConfig.Name)
			m.RecordSuccess(configName)
		}
	}

	var prestateSource prestateFetcher
	if strings.HasPrefix(runConfig.PrestateFilename, "file:") {
		path := runConfig.PrestateFilename[len("file:"):]
		rlog.Info("Using local file prestate", "type", runConfig.GameType, "path", path)
		prestateSource = &LocalPrestateFetcher{path: path}
	} else if runConfig.PrestateFilename != "" {
		rlog.Info("Using named prestate", "type", runConfig.GameType, "filename", runConfig.PrestateFilename)
		prestateSource = &NamedPrestateFetcher{filename: runConfig.PrestateFilename}
	} else if runConfig.Prestate == (common.Hash{}) {
		rlog.Info("Using on chain prestate", "type", runConfig.GameType)
		prestateSource = &OnChainPrestateFetcher{
			m:                  r.m,
			gameFactoryAddress: r.cfg.GameFactoryAddress,
			gameType:           runConfig.GameType,
			caller:             caller,
		}
	} else {
		rlog.Info("Using specific prestate", "type", runConfig.GameType, "hash", runConfig.Prestate)
		prestateSource = &HashPrestateFetcher{prestateHash: runConfig.Prestate}
	}

	localInputs, err := createGameInputs(ctx, rlog, rollupClient, supervisorClient, runConfig.Name, runConfig.GameType)
	if err != nil {
		recordError(err, runConfig.Name, r.m, rlog)
		return
	}

	inputsLogger := rlog.New("l1", localInputs.L1Head, "l2", localInputs.L2Head, "l2Block", localInputs.L2SequenceNumber, "claim", localInputs.L2Claim)
	// Sanitize the directory name.
	safeName := regexp.MustCompile("[^a-zA-Z0-9_-]").ReplaceAllString(runConfig.Name, "")
	dir, err := r.prepDatadir(safeName)
	if err != nil {
		recordError(err, runConfig.Name, r.m, rlog)
		return
	}
	err = r.runOnce(ctx, inputsLogger.With("type", runConfig.Name), runConfig.Name, runConfig.GameType, prestateSource, localInputs, dir)
	recordError(err, runConfig.Name, r.m, rlog)
}

func (r *Runner) runOnce(ctx context.Context, logger log.Logger, name string, gameType gameTypes.GameType, prestateSource prestateFetcher, localInputs utils.LocalGameInputs, dir string) error {
	if r.vmTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.vmTimeout)
		defer cancel()
	}
	provider, err := r.traceProviderCreator(ctx, logger, metrics.NewTypedVmMetrics(r.m, name), r.cfg, prestateSource, gameType, localInputs, dir)
	if err != nil {
		return fmt.Errorf("failed to create trace provider: %w", err)
	}
	hash, err := provider.Get(ctx, types.RootPosition)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("%w: %w", ErrVMTimeout, err)
		}
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

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

func generateRunID() string {
	var b [6]byte
	_, _ = io.ReadFull(rand.Reader, b[:])
	return b32.EncodeToString(b[:])
}

var _ cliapp.Lifecycle = (*Runner)(nil)
