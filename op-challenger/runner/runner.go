package runner

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/prestates"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrUnexpectedStatusCode = errors.New("unexpected status code")
)

type Metricer interface {
	vm.Metricer
	contractMetrics.ContractMetricer

	RecordFailure(vmType types.TraceType)
	RecordInvalid(vmType types.TraceType)
	RecordSuccess(vmType types.TraceType)
}

type Runner struct {
	log log.Logger
	cfg *config.Config
	m   Metricer

	running    atomic.Bool
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	metricsSrv *httputil.HTTPServer
}

func NewRunner(logger log.Logger, cfg *config.Config) *Runner {
	return &Runner{
		log: logger,
		cfg: cfg,
		m:   NewMetrics(),
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

	rollupClient, err := dial.DialRollupClientWithTimeout(ctx, 1*time.Minute, r.log, r.cfg.RollupRpc)
	if err != nil {
		return fmt.Errorf("failed to dial rollup client: %w", err)
	}

	l1Client, err := dial.DialRPCClientWithTimeout(ctx, 1*time.Minute, r.log, r.cfg.L1EthRpc)
	if err != nil {
		return fmt.Errorf("failed to dial l1 client: %w", err)
	}
	caller := batching.NewMultiCaller(l1Client, batching.DefaultBatchSize)

	for _, traceType := range r.cfg.TraceTypes {
		r.wg.Add(1)
		go r.loop(ctx, traceType, rollupClient, caller)
	}

	r.log.Info("Runners started")
	return nil
}

func (r *Runner) loop(ctx context.Context, traceType types.TraceType, client *sources.RollupClient, caller *batching.MultiCaller) {
	defer r.wg.Done()
	t := time.NewTicker(1 * time.Minute)
	defer t.Stop()
	for {
		if err := r.runOnce(ctx, traceType, client, caller); errors.Is(err, ErrUnexpectedStatusCode) {
			r.log.Error("Incorrect status code", "type", traceType, "err", err)
			r.m.RecordInvalid(traceType)
		} else if err != nil {
			r.log.Error("Failed to run", "type", traceType, "err", err)
			r.m.RecordFailure(traceType)
		} else {
			r.log.Info("Successfully verified output root", "type", traceType)
			r.m.RecordSuccess(traceType)
		}
		select {
		case <-t.C:
		case <-ctx.Done():
			return
		}
	}
}

func (r *Runner) runOnce(ctx context.Context, traceType types.TraceType, client *sources.RollupClient, caller *batching.MultiCaller) error {
	prestateHash, err := r.getPrestateHash(ctx, traceType, caller)
	if err != nil {
		return err
	}

	localInputs, err := r.createGameInputs(ctx, client)
	if err != nil {
		return err
	}
	dir, err := r.prepDatadir(traceType)
	if err != nil {
		return err
	}
	prestateSource := prestates.NewPrestateSource(
		r.cfg.CannonAbsolutePreStateBaseURL,
		r.cfg.CannonAbsolutePreState,
		filepath.Join(dir, "prestates"))
	logger := r.log.New("l1", localInputs.L1Head, "l2", localInputs.L2Head, "l2Block", localInputs.L2BlockNumber, "claim", localInputs.L2Claim, "type", traceType)
	provider, err := createTraceProvider(logger, r.m, r.cfg, prestateSource, prestateHash, traceType, localInputs, dir)
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

func (r *Runner) prepDatadir(traceType types.TraceType) (string, error) {
	dir := filepath.Join(r.cfg.Datadir, traceType.String())
	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("failed to remove old dir: %w", err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create data dir (%v): %w", dir, err)
	}
	return dir, nil
}

func (r *Runner) createGameInputs(ctx context.Context, client *sources.RollupClient) (utils.LocalGameInputs, error) {
	status, err := client.SyncStatus(ctx)
	if err != nil {
		return utils.LocalGameInputs{}, fmt.Errorf("failed to get rollup sync status: %w", err)
	}

	if status.SafeL2.Number == 0 {
		return utils.LocalGameInputs{}, errors.New("safe head is 0")
	}
	claimOutput, err := client.OutputAtBlock(ctx, status.SafeL2.Number)
	if err != nil {
		return utils.LocalGameInputs{}, fmt.Errorf("failed to get claim output: %w", err)
	}
	parentOutput, err := client.OutputAtBlock(ctx, status.SafeL2.Number-1)
	if err != nil {
		return utils.LocalGameInputs{}, fmt.Errorf("failed to get claim output: %w", err)
	}
	localInputs := utils.LocalGameInputs{
		L1Head:        status.HeadL1.Hash,
		L2Head:        parentOutput.BlockRef.Hash,
		L2OutputRoot:  common.Hash(parentOutput.OutputRoot),
		L2Claim:       common.Hash(claimOutput.OutputRoot),
		L2BlockNumber: new(big.Int).SetUint64(status.SafeL2.Number),
	}
	return localInputs, nil
}

func (r *Runner) getPrestateHash(ctx context.Context, traceType types.TraceType, caller *batching.MultiCaller) (common.Hash, error) {
	gameFactory := contracts.NewDisputeGameFactoryContract(r.m, r.cfg.GameFactoryAddress, caller)
	gameImplAddr, err := gameFactory.GetGameImpl(ctx, traceType.GameType())
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to load game impl: %w", err)
	}
	gameImpl, err := contracts.NewFaultDisputeGameContract(ctx, r.m, gameImplAddr, caller)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create fault dispute game contract bindings for %v: %w", gameImplAddr, err)
	}
	prestateHash, err := gameImpl.GetAbsolutePrestateHash(ctx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get absolute prestate hash for %v: %w", gameImplAddr, err)
	}
	return prestateHash, err
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
