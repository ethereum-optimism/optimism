package derive2

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

type PipelineV2 struct {
	log       log.Logger
	rollupCfg *rollup.Config
	l1Fetcher derive.L1Fetcher
	l2        derive.L2Source

	metrics derive.Metrics

	attrib      *derive.AttributesQueue
	l1Traversal *L1Traversal

	// lastLocalSafe is the last confirmed local-safe block.
	// This is always non-zero, unless a reset was started.
	// This is a fully confirmed block; the attributes-stage
	// may contain attributes that follow *just after* this block, or otherwise nil attributes.
	lastLocalSafe eth.L2BlockRef

	// Index of the stage that is currently being reset.
	// >= len(stages) if no additional resetting is required
	resetting int
	stages    []derive.ResettableStage

	// if resetL2Safe is zero, then the stages cannot be reset yet

	// resetL2Safe is the parent L2 block to start building on top of
	resetL2Safe eth.L2BlockRef
	// resetL1Source is the source L1 block that the first relevant
	// channel data may be retrieved from to create the next block
	resetL1Source eth.L1BlockRef
	// resetSysConfig is system-config info that may be used during the reset, based on the resetL2Safe
	resetSysConfig eth.SystemConfig

	id ID

	// context for the lifetime of the pipeline, to anchor all requests onto
	ctx context.Context

	emitter event.Emitter
}

var _ event.AttachEmitter = (*PipelineV2)(nil)
var _ event.Deriver = (*PipelineV2)(nil)

// NewDerivationPipeline creates a DerivationPipeline, to turn L1 data into L2 block-inputs.
func NewDerivationPipeline(rootCtx context.Context, log log.Logger, rollupCfg *rollup.Config,
	depSet derive.DependencySet, l1Fetcher derive.L1Fetcher, l1Blobs derive.L1BlobsFetcher,
	altDA derive.AltDAInputFetcher, l2Source derive.L2Source, metrics derive.Metrics,
) *PipelineV2 {
	id := NextID()

	spec := rollup.NewChainSpec(rollupCfg)
	// Stages are strung together into a pipeline,
	// results are pulled from the stage closed to the L2 engine, which pulls from the previous stage, and so on.
	l1Traversal := NewL1Traversal(log, rollupCfg, l1Fetcher)
	dataSrc := derive.NewDataSourceFactory(log, rollupCfg, l1Fetcher, l1Blobs, altDA) // auxiliary stage for L1Retrieval
	l1Src := derive.NewL1Retrieval(log, dataSrc, l1Traversal)
	frameQueue := derive.NewFrameQueue(log, rollupCfg, l1Src)
	channelMux := derive.NewChannelAssembler(log, spec, frameQueue, metrics)
	chInReader := derive.NewChannelInReader(rollupCfg, log, channelMux, metrics)
	batchMux := derive.NewBatchStage(log, rollupCfg, chInReader, l2Source)
	attrBuilder := derive.NewFetchingAttributesBuilder(rollupCfg, depSet, l1Fetcher, l2Source)
	attributesQueue := derive.NewAttributesQueue(log, rollupCfg, attrBuilder, batchMux)

	// Reset from ResetEngine then up from L1 Traversal. The stages do not talk to each other during
	// the ResetEngine, but after the ResetEngine, this is the order in which the stages could talk to each other.
	// Note: The ResetEngine is the only reset that can fail.
	stages := []derive.ResettableStage{l1Traversal, l1Src, altDA, frameQueue, channelMux, chInReader, batchMux, attributesQueue}

	return &PipelineV2{
		log:         log,
		rollupCfg:   rollupCfg,
		l1Fetcher:   l1Fetcher,
		resetting:   0,
		stages:      stages,
		metrics:     metrics,
		l1Traversal: l1Traversal,
		attrib:      attributesQueue,
		l2:          l2Source,
		id:          id,
		ctx:         WithID(rootCtx, id),
	}
}

func (dp *PipelineV2) ID() ID {
	return dp.id
}

func (dp *PipelineV2) Reset() {
	dp.resetting = 0
	dp.resetSysConfig = eth.SystemConfig{}
	dp.resetL2Safe = eth.L2BlockRef{}
}

// Step tries to progress the buffer.
// An EOF is returned if the pipeline is blocked by waiting for new L1 data.
// If ctx errors no error is returned, but the step may exit early in a state that can still be continued.
// Any other error is critical and the derivation pipeline should be reset.
// An error is expected when the underlying source closes.
// When Step returns nil, it should be called again, to continue the derivation process.
func (dp *PipelineV2) onStep(ctx context.Context, ev StepRequestEvent) {
	dp.metrics.SetDerivationIdle(false)

	// if any stages need to be reset, do that first.
	if dp.resetting < len(dp.stages) {
		// Different from legacy pipeline implementation:
		// Wait for the outside to reset us accurately and set the dp.resetL2Safe.
		// The step request may also just be stale.
		if dp.resetL2Safe == (eth.L2BlockRef{}) {
			dp.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
				Err: errors.New("not determined where to reset to yet"),
			})
			return
		}

		dp.log.Info("Rewinding derivation-pipeline L1 traversal to handle reset")

		dp.metrics.RecordPipelineReset()

		sysCfg, err := dp.l2.SystemConfigByL2Hash(dp.ctx, dp.resetL2Safe.Hash)
		if err != nil {
			dp.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
				Err: fmt.Errorf("failed to fetch L1 config of L2 block %s: %w", dp.resetL2Safe.ID(), err),
			})
			return
		}
		dp.resetSysConfig = sysCfg

		if err := dp.stages[dp.resetting].Reset(dp.ctx, dp.resetL1Source, dp.resetSysConfig); err == io.EOF {
			dp.log.Debug("reset of stage completed", "stage", dp.resetting)
			dp.resetting += 1
			return // TODO step request
		} else if err != nil {
			if errors.Is(err, derive.ErrReset) {
				dp.emitter.Emit(ctx, rollup.ResetEvent{
					Err: fmt.Errorf("reset during resetting: %w", err),
				})
				return
			} else {
				dp.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
					Err: fmt.Errorf("stage %d failed resetting: %w", dp.resetting, err),
				})
			}
		}
		// resetting will need to complete (iteratively) before continuing with derivation
		return
	}

	// Sanity-check we are post-Holocene.
	// Pre-Holocene derivation rules are no longer supported
	if !dp.rollupCfg.IsHolocene(dp.lastLocalSafe.Time) {
		dp.emitter.Emit(ctx, rollup.ResetEvent{
			Err: fmt.Errorf("last L2 block is pre-Holocene, cannot derive blocks: %s", dp.lastLocalSafe),
		})
		return
	}

	goIdle := true
	if attrib, err := dp.attrib.NextAttributes(dp.ctx, dp.lastLocalSafe); err == nil {
		dp.emitter.Emit(ctx, derive.DerivedAttributesEvent{Attributes: attrib})
		dp.metrics.RecordL1Ref("l1_derived", attrib.DerivedFrom)
		goIdle = false
	} else if err == io.EOF {
		// If every stage has returned io.EOF, try to advance the L1 Origin
		dp.emitter.Emit(ctx, derive.ExhaustedL1Event{
			L1Ref:  dp.attrib.Origin(),
			LastL2: dp.lastLocalSafe,
		})
	} else if errors.Is(err, derive.EngineELSyncing) {
		dp.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: fmt.Errorf("engine is syncing %s: %w", dp.resetL2Safe.ID(), err),
		})
	} else if errors.Is(err, derive.ErrReset) {
		dp.emitter.Emit(ctx, rollup.ResetEvent{Err: err})
	} else if errors.Is(err, derive.ErrTemporary) { // TODO would be nice to differentiate between L1/L2
		dp.emitter.Emit(ctx, rollup.L1TemporaryErrorEvent{Err: err})
	} else if errors.Is(err, derive.ErrCritical) {
		dp.emitter.Emit(ctx, rollup.CriticalErrorEvent{Err: err})
	} else if errors.Is(err, derive.NotEnoughData) {
		// don't do a backoff for this error (bad error name)
		goIdle = false
	} else if err != nil {
		dp.log.Error("Derivation process error", "err", err)
		dp.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{Err: err})
	}

	if goIdle {
		dp.metrics.SetDerivationIdle(true)
	} else {
		dp.emitter.Emit(ctx, derive.DeriverMoreEvent{})
	}
}

func (d *PipelineV2) AttachEmitter(em event.Emitter) {
	d.emitter = em
}

type StepRequestEvent struct {
}

func (StepRequestEvent) String() string {
	return "step-request"
}

type ConfirmAttributesEvent struct {
	Confirmed eth.L2BlockRef
}

func (ConfirmAttributesEvent) String() string {
	return "confirm-attributes"
}

func (d *PipelineV2) OnEvent(ctx context.Context, ev event.Event) bool {
	if IDFromContext(ctx) != d.id { // If the event is not for us, ignore it
		return false
	}

	switch x := ev.(type) {
	case ConfirmAttributesEvent:
		if d.lastLocalSafe.Hash != x.Confirmed.ParentHash {
			d.log.Error("Confirmed payload does not follow last derived block", "confirmed", x.Confirmed, "last", d.lastLocalSafe)
			d.emitter.Emit(ctx, rollup.ResetEvent{
				Err: fmt.Errorf("confirmed payload %s does not follow %s", x.Confirmed, d.lastLocalSafe),
			})
			return true
		}
		d.lastLocalSafe = x.Confirmed
		d.emitter.Emit(ctx, derive.DeriverMoreEvent{})
	case StepRequestEvent:
		d.onStep(ctx, x)
	case derive.DepositsOnlyPayloadAttributesRequestEvent:
		d.log.Warn("Deriving deposits-only attributes", "origin", d.attrib.Origin())
		attrib, err := d.attrib.DepositsOnlyAttributes(x.Parent, x.DerivedFrom)
		if err != nil {
			d.emitter.Emit(ctx, rollup.CriticalErrorEvent{
				Err: fmt.Errorf("deriving deposits-only attributes: %w", err),
			})
			return true
		}
		d.emitter.Emit(ctx, derive.DerivedAttributesEvent{Attributes: attrib})
	case derive.ProvideL1Traversal:
		d.l1Traversal.ProvideNext(x.NextL1)
	default:
		return false
	}
	return true
}
