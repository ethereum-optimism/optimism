package derive

import (
	"context"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

type Metrics interface {
	RecordL1Ref(name string, ref eth.L1BlockRef)
	RecordL2Ref(name string, ref eth.L2BlockRef)
	RecordUnsafePayloadsBuffer(length uint64, memSize uint64, next eth.BlockID)
	RecordChannelInputBytes(inputCompresedBytes int)
}

type L1Fetcher interface {
	L1BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L1BlockRef, error)
	L1BlockRefByNumberFetcher
	L1BlockRefByHashFetcher
	L1ReceiptsFetcher
	L1TransactionFetcher
}

// ResettableEngineControl wraps EngineControl with reset-functionality,
// which handles reorgs like the derivation pipeline:
// by determining the last valid block references to continue from.
type ResettableEngineControl interface {
	EngineControl
	Reset()
}

type ResetableStage interface {
	// Reset resets a pull stage. `base` refers to the L1 Block Reference to reset to, with corresponding configuration.
	Reset(ctx context.Context, base eth.L1BlockRef, baseCfg eth.SystemConfig) error
}

type EngineQueueStage interface {
	EngineControl

	FinalizedL1() eth.L1BlockRef
	Finalized() eth.L2BlockRef
	UnsafeL2Head() eth.L2BlockRef
	SafeL2Head() eth.L2BlockRef
	Origin() eth.L1BlockRef
	SystemConfig() eth.SystemConfig
	SetUnsafeHead(head eth.L2BlockRef)

	Finalize(l1Origin eth.L1BlockRef)
	AddUnsafePayload(payload *eth.ExecutionPayload)
	UnsafeL2SyncTarget() eth.L2BlockRef
	Step(context.Context) error
}

// DerivationPipeline is updated with new L1 data, and the Step() function can be iterated on to keep the L2 Engine in sync.
type DerivationPipeline struct {
	log       log.Logger
	cfg       *rollup.Config
	l1Fetcher L1Fetcher

	// Index of the stage that is currently being reset.
	// >= len(stages) if no additional resetting is required
	resetting int
	stages    []ResetableStage

	// Special stages to keep track of
	traversal *L1Traversal
	eng       EngineQueueStage

	metrics Metrics
}

// NewDerivationPipeline creates a derivation pipeline, which should be reset before use.
func NewDerivationPipeline(log log.Logger, cfg *rollup.Config, l1Fetcher L1Fetcher, engine Engine, metrics Metrics) *DerivationPipeline {

	// Pull stages
	l1Traversal := NewL1Traversal(log, cfg, l1Fetcher)
	dataSrc := NewDataSourceFactory(log, cfg, l1Fetcher) // auxiliary stage for L1Retrieval
	l1Src := NewL1Retrieval(log, dataSrc, l1Traversal)
	frameQueue := NewFrameQueue(log, l1Src)
	bank := NewChannelBank(log, cfg, frameQueue, l1Fetcher)
	chInReader := NewChannelInReader(log, bank, metrics)
	batchQueue := NewBatchQueue(log, cfg, chInReader)
	attrBuilder := NewFetchingAttributesBuilder(cfg, l1Fetcher, engine)
	attributesQueue := NewAttributesQueue(log, cfg, attrBuilder, batchQueue)

	// Step stages
	eng := NewEngineQueue(log, cfg, engine, metrics, attributesQueue, l1Fetcher)

	// Reset from engine queue then up from L1 Traversal. The stages do not talk to each other during
	// the reset, but after the engine queue, this is the order in which the stages could talk to each other.
	// Note: The engine queue stage is the only reset that can fail.
	stages := []ResetableStage{eng, l1Traversal, l1Src, frameQueue, bank, chInReader, batchQueue, attributesQueue}

	return &DerivationPipeline{
		log:       log,
		cfg:       cfg,
		l1Fetcher: l1Fetcher,
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
	dp.eng.Finalize(l1Origin)
}

// FinalizedL1 is the L1 finalization of the inner-most stage of the derivation pipeline,
// i.e. the L1 chain up to and including this point included and/or produced all the finalized L2 blocks.
func (dp *DerivationPipeline) FinalizedL1() eth.L1BlockRef {
	return dp.eng.FinalizedL1()
}

func (dp *DerivationPipeline) Finalized() eth.L2BlockRef {
	return dp.eng.Finalized()
}

func (dp *DerivationPipeline) SafeL2Head() eth.L2BlockRef {
	return dp.eng.SafeL2Head()
}

// UnsafeL2Head returns the head of the L2 chain that we are deriving for, this may be past what we derived from L1
func (dp *DerivationPipeline) UnsafeL2Head() eth.L2BlockRef {
	return dp.eng.UnsafeL2Head()
}

func (dp *DerivationPipeline) StartPayload(ctx context.Context, parent eth.L2BlockRef, attrs *eth.PayloadAttributes, updateSafe bool) (errType BlockInsertionErrType, err error) {
	return dp.eng.StartPayload(ctx, parent, attrs, updateSafe)
}

func (dp *DerivationPipeline) ConfirmPayload(ctx context.Context) (out *eth.ExecutionPayload, errTyp BlockInsertionErrType, err error) {
	return dp.eng.ConfirmPayload(ctx)
}

func (dp *DerivationPipeline) CancelPayload(ctx context.Context, force bool) error {
	return dp.eng.CancelPayload(ctx, force)
}

func (dp *DerivationPipeline) BuildingPayload() (onto eth.L2BlockRef, id eth.PayloadID, safe bool) {
	return dp.eng.BuildingPayload()
}

// AddUnsafePayload schedules an execution payload to be processed, ahead of deriving it from L1
func (dp *DerivationPipeline) AddUnsafePayload(payload *eth.ExecutionPayload) {
	dp.eng.AddUnsafePayload(payload)
}

// UnsafeL2SyncTarget retrieves the first queued-up L2 unsafe payload, or a zeroed reference if there is none.
func (dp *DerivationPipeline) UnsafeL2SyncTarget() eth.L2BlockRef {
	return dp.eng.UnsafeL2SyncTarget()
}

// Step tries to progress the buffer.
// An EOF is returned if there pipeline is blocked by waiting for new L1 data.
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
	} else if err != nil {
		return fmt.Errorf("engine stage failed: %w", err)
	} else {
		return nil
	}
}
