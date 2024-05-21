package derive

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type Metrics interface {
	RecordL1Ref(name string, ref eth.L1BlockRef)
	RecordL2Ref(name string, ref eth.L2BlockRef)
	RecordUnsafePayloadsBuffer(length uint64, memSize uint64, next eth.BlockID)
	RecordChannelInputBytes(inputCompressedBytes int)
	RecordHeadChannelOpened()
	RecordChannelTimedOut()
	RecordFrame()
	RecordDerivedBatches(batchType string)
}

type L1Fetcher interface {
	L1BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L1BlockRef, error)
	L1BlockRefByNumberFetcher
	L1BlockRefByHashFetcher
	L1ReceiptsFetcher
	L1TransactionFetcher
}

type ResettableStage interface {
	// Reset resets a pull stage. `base` refers to the L1 Block Reference to reset to, with corresponding configuration.
	Reset(ctx context.Context, base eth.L1BlockRef, baseCfg eth.SystemConfig) error
}

type EngineQueueStage interface {
	LowestQueuedUnsafeBlock() eth.L2BlockRef
	FinalizedL1() eth.L1BlockRef
	Origin() eth.L1BlockRef
	SystemConfig() eth.SystemConfig

	Finalize(l1Origin eth.L1BlockRef)
	AddUnsafePayload(payload *eth.ExecutionPayloadEnvelope)
	Step(context.Context) error
}

// DerivationPipeline is updated with new L1 data, and the Step() function can be iterated on to keep the L2 Engine in sync.
type DerivationPipeline struct {
	log       log.Logger
	rollupCfg *rollup.Config
	l1Fetcher L1Fetcher
	plasma    PlasmaInputFetcher

	// Index of the stage that is currently being reset.
	// >= len(stages) if no additional resetting is required
	resetting int
	stages    []ResettableStage

	// Special stages to keep track of
	traversal *L1Traversal
	eng       EngineQueueStage

	metrics Metrics
}

// NewDerivationPipeline creates a derivation pipeline, which should be reset before use.

func NewDerivationPipeline(log log.Logger, rollupCfg *rollup.Config, l1Fetcher L1Fetcher, l1Blobs L1BlobsFetcher, plasma PlasmaInputFetcher, l2Source L2Source, engine LocalEngineControl, metrics Metrics, syncCfg *sync.Config, safeHeadListener SafeHeadListener) *DerivationPipeline {

	// Pull stages
	l1Traversal := NewL1Traversal(log, rollupCfg, l1Fetcher)
	dataSrc := NewDataSourceFactory(log, rollupCfg, l1Fetcher, l1Blobs, plasma) // auxiliary stage for L1Retrieval
	l1Src := NewL1Retrieval(log, dataSrc, l1Traversal)
	frameQueue := NewFrameQueue(log, l1Src)
	bank := NewChannelBank(log, rollupCfg, frameQueue, l1Fetcher, metrics)
	chInReader := NewChannelInReader(rollupCfg, log, bank, metrics)
	batchQueue := NewBatchQueue(log, rollupCfg, chInReader, l2Source)
	attrBuilder := NewFetchingAttributesBuilder(rollupCfg, l1Fetcher, l2Source)
	attributesQueue := NewAttributesQueue(log, rollupCfg, attrBuilder, batchQueue)

	// Step stages
	eng := NewEngineQueue(log, rollupCfg, l2Source, engine, metrics, attributesQueue, l1Fetcher, syncCfg, safeHeadListener)

	// Plasma takes control of the engine finalization signal only when usePlasma is enabled.
	plasma.OnFinalizedHeadSignal(func(ref eth.L1BlockRef) {
		eng.Finalize(ref)
	})

	// Reset from engine queue then up from L1 Traversal. The stages do not talk to each other during
	// the reset, but after the engine queue, this is the order in which the stages could talk to each other.
	// Note: The engine queue stage is the only reset that can fail.
	stages := []ResettableStage{eng, l1Traversal, l1Src, plasma, frameQueue, bank, chInReader, batchQueue, attributesQueue}

	return &DerivationPipeline{
		log:       log,
		rollupCfg: rollupCfg,
		l1Fetcher: l1Fetcher,
		plasma:    plasma,
		resetting: 0,
		stages:    stages,
		eng:       eng,
		metrics:   metrics,
		traversal: l1Traversal,
	}
}

// EngineReady returns true if the engine is ready to be used.
// When it's being reset its state is inconsistent, and should not be used externally.
func (dp *DerivationPipeline) EngineReady() bool {
	return dp.resetting > 0
}

func (dp *DerivationPipeline) Reset() {
	dp.resetting = 0
}

// Origin is the L1 block of the inner-most stage of the derivation pipeline,
// i.e. the L1 chain up to and including this point included and/or produced all the safe L2 blocks.
func (dp *DerivationPipeline) Origin() eth.L1BlockRef {
	return dp.eng.Origin()
}

func (dp *DerivationPipeline) Finalize(l1Origin eth.L1BlockRef) {
	// In plasma mode, the finalization signal is proxied through the plasma manager.
	// Finality signal will come from the DA contract or L1 finality whichever is last.
	if dp.rollupCfg.PlasmaEnabled() {
		dp.plasma.Finalize(l1Origin)
	} else {
		dp.eng.Finalize(l1Origin)
	}
}

// FinalizedL1 is the L1 finalization of the inner-most stage of the derivation pipeline,
// i.e. the L1 chain up to and including this point included and/or produced all the finalized L2 blocks.
func (dp *DerivationPipeline) FinalizedL1() eth.L1BlockRef {
	return dp.eng.FinalizedL1()
}

// AddUnsafePayload schedules an execution payload to be processed, ahead of deriving it from L1
func (dp *DerivationPipeline) AddUnsafePayload(payload *eth.ExecutionPayloadEnvelope) {
	dp.eng.AddUnsafePayload(payload)
}

// LowestQueuedUnsafeBlock returns the lowest queued unsafe block. If the gap is filled from the unsafe head
// to this block, the EngineQueue will be able to apply the queued payloads.
func (dp *DerivationPipeline) LowestQueuedUnsafeBlock() eth.L2BlockRef {
	return dp.eng.LowestQueuedUnsafeBlock()
}

// Step tries to progress the buffer.
// An EOF is returned if the pipeline is blocked by waiting for new L1 data.
// If ctx errors no error is returned, but the step may exit early in a state that can still be continued.
// Any other error is critical and the derivation pipeline should be reset.
// An error is expected when the underlying source closes.
// When Step returns nil, it should be called again, to continue the derivation process.
func (dp *DerivationPipeline) Step(ctx context.Context) error {
	defer dp.metrics.RecordL1Ref("l1_derived", dp.Origin())

	// if any stages need to be reset, do that first.
	if dp.resetting < len(dp.stages) {
		if err := dp.stages[dp.resetting].Reset(ctx, dp.eng.Origin(), dp.eng.SystemConfig()); err == io.EOF {
			dp.log.Debug("reset of stage completed", "stage", dp.resetting, "origin", dp.eng.Origin())
			dp.resetting += 1
			return nil
		} else if err != nil {
			return fmt.Errorf("stage %d failed resetting: %w", dp.resetting, err)
		} else {
			return nil
		}
	}

	// Now step the engine queue. It will pull earlier data as needed.
	if err := dp.eng.Step(ctx); err == io.EOF {
		// If every stage has returned io.EOF, try to advance the L1 Origin
		return dp.traversal.AdvanceL1Block(ctx)
	} else if errors.Is(err, EngineELSyncing) {
		return err
	} else if err != nil {
		return fmt.Errorf("engine stage failed: %w", err)
	} else {
		return nil
	}
}
