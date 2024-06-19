package driver

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/attributes"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	plasma "github.com/ethereum-optimism/optimism/op-plasma"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var ErrClaimNotValid = errors.New("invalid claim")

type Derivation interface {
	Step(ctx context.Context) error
}

type Pipeline interface {
	Step(ctx context.Context, pendingSafeHead eth.L2BlockRef) (outAttrib *derive.AttributesWithParent, outErr error)
	ConfirmEngineReset()
}

type Engine interface {
	SafeL2Head() eth.L2BlockRef
	PendingSafeL2Head() eth.L2BlockRef
	TryUpdateEngine(ctx context.Context) error
	engine.ResetEngineControl
}

type L2Source interface {
	engine.Engine
	L2OutputRoot(uint64) (eth.Bytes32, error)
}

type Deriver interface {
	SafeL2Head() eth.L2BlockRef
	SyncStep(ctx context.Context) error
}

type MinimalSyncDeriver struct {
	logger            log.Logger
	pipeline          Pipeline
	attributesHandler driver.AttributesHandler
	l1Source          derive.L1Fetcher
	l2Source          L2Source
	engine            Engine
	syncCfg           *sync.Config
	initialResetDone  bool
	cfg               *rollup.Config
}

func (d *MinimalSyncDeriver) SafeL2Head() eth.L2BlockRef {
	return d.engine.SafeL2Head()
}

func (d *MinimalSyncDeriver) SyncStep(ctx context.Context) error {
	if !d.initialResetDone {
		if err := d.engine.TryUpdateEngine(ctx); !errors.Is(err, engine.ErrNoFCUNeeded) {
			return err
		}
		if err := engine.ResetEngine(ctx, d.logger, d.cfg, d.engine, d.l1Source, d.l2Source, d.syncCfg, nil); err != nil {
			return err
		}
		d.pipeline.ConfirmEngineReset()
		d.initialResetDone = true
	}

	if err := d.engine.TryUpdateEngine(ctx); !errors.Is(err, engine.ErrNoFCUNeeded) {
		return err
	}
	if err := d.attributesHandler.Proceed(ctx); err != io.EOF {
		// EOF error means we can't process the next attributes. Then we should derive the next attributes.
		return err
	}

	attrib, err := d.pipeline.Step(ctx, d.engine.PendingSafeL2Head())
	if err != nil {
		return err
	}
	d.attributesHandler.SetAttributes(attrib)
	return nil
}

type Driver struct {
	logger log.Logger

	deriver Deriver

	l2OutputRoot   func(uint64) (eth.Bytes32, error)
	targetBlockNum uint64
}

func NewDriver(logger log.Logger, cfg *rollup.Config, l1Source derive.L1Fetcher, l1BlobsSource derive.L1BlobsFetcher, l2Source L2Source, targetBlockNum uint64) *Driver {
	engine := engine.NewEngineController(l2Source, logger, metrics.NoopMetrics, cfg, sync.CLSync)
	attributesHandler := attributes.NewAttributesHandler(logger, cfg, engine, l2Source)
	syncCfg := &sync.Config{SyncMode: sync.CLSync}
	pipeline := derive.NewDerivationPipeline(logger, cfg, l1Source, l1BlobsSource, plasma.Disabled, l2Source, metrics.NoopMetrics)
	return &Driver{
		logger: logger,
		deriver: &MinimalSyncDeriver{
			logger:            logger,
			pipeline:          pipeline,
			attributesHandler: attributesHandler,
			l1Source:          l1Source,
			l2Source:          l2Source,
			engine:            engine,
			syncCfg:           syncCfg,
			cfg:               cfg,
		},
		l2OutputRoot:   l2Source.L2OutputRoot,
		targetBlockNum: targetBlockNum,
	}
}

// Step runs the next step of the derivation pipeline.
// Returns nil if there are further steps to be performed
// Returns io.EOF if the derivation completed successfully
// Returns a non-EOF error if the derivation failed
func (d *Driver) Step(ctx context.Context) error {
	if err := d.deriver.SyncStep(ctx); errors.Is(err, io.EOF) {
		d.logger.Info("Derivation complete: reached L1 head", "head", d.deriver.SafeL2Head())
		return io.EOF
	} else if errors.Is(err, derive.NotEnoughData) {
		// NotEnoughData is not handled differently than a nil error.
		// This used to be returned by the EngineQueue when a block was derived, but also other stages.
		// Instead, every driver-loop iteration we check if the target block number has been reached.
		d.logger.Debug("Data is lacking")
	} else if errors.Is(err, derive.ErrTemporary) {
		// While most temporary errors are due to requests for external data failing which can't happen,
		// they may also be returned due to other events like channels timing out so need to be handled
		d.logger.Warn("Temporary error in derivation", "err", err)
		return nil
	} else if err != nil {
		return fmt.Errorf("pipeline err: %w", err)
	}
	head := d.deriver.SafeL2Head()
	if head.Number >= d.targetBlockNum {
		d.logger.Info("Derivation complete: reached L2 block", "head", head)
		return io.EOF
	}
	return nil
}

func (d *Driver) SafeHead() eth.L2BlockRef {
	return d.deriver.SafeL2Head()
}

func (d *Driver) ValidateClaim(l2ClaimBlockNum uint64, claimedOutputRoot eth.Bytes32) error {
	l2Head := d.SafeHead()
	outputRoot, err := d.l2OutputRoot(min(l2ClaimBlockNum, l2Head.Number))
	if err != nil {
		return fmt.Errorf("calculate L2 output root: %w", err)
	}
	d.logger.Info("Validating claim", "head", l2Head, "output", outputRoot, "claim", claimedOutputRoot)
	if claimedOutputRoot != outputRoot {
		return fmt.Errorf("%w: claim: %v actual: %v", ErrClaimNotValid, claimedOutputRoot, outputRoot)
	}
	return nil
}
