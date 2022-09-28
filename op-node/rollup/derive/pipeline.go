package derive

import (
	"context"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/log"
)

type Metrics interface {
	RecordL1Ref(name string, ref eth.L1BlockRef)
	RecordL2Ref(name string, ref eth.L2BlockRef)
	RecordUnsafePayloadsBuffer(length uint64, memSize uint64, next eth.BlockID)
}

type L1Fetcher interface {
	L1BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L1BlockRef, error)
	L1BlockRefByNumberFetcher
	L1BlockRefByHashFetcher
	L1ReceiptsFetcher
	L1TransactionFetcher
}

type StageProgress interface {
	Progress() Progress
}

type PullStage interface {
	// Reset resets a pull stage. `base` refers to the L1 Block Reference to reset to.
	// TODO: Return L1 Block reference
	Reset(ctx context.Context, base eth.L1BlockRef) error
}

type Stage interface {
	StageProgress

	// Step tries to progress the state.
	// The outer stage progress informs the step what to do.
	//
	// If the stage:
	// - returns EOF: the stage will be skipped
	// - returns another error: the stage will make the pipeline error.
	// - returns nil: the stage will be repeated next Step
	Step(ctx context.Context, outer Progress) error

	// ResetStep prepares the state for usage in regular steps.
	// Similar to Step(ctx) it returns:
	// - EOF if the next stage should be reset
	// - error if the reset should start all over again
	// - nil if the reset should continue resetting this stage.
	ResetStep(ctx context.Context, l1Fetcher L1Fetcher) error
}

type EngineQueueStage interface {
	Finalized() eth.L2BlockRef
	UnsafeL2Head() eth.L2BlockRef
	SafeL2Head() eth.L2BlockRef
	Progress() Progress
	SetUnsafeHead(head eth.L2BlockRef)

	Finalize(l1Origin eth.BlockID)
	AddSafeAttributes(attributes *eth.PayloadAttributes)
	AddUnsafePayload(payload *eth.ExecutionPayload)
}

// DerivationPipeline is updated with new L1 data, and the Step() function can be iterated on to keep the L2 Engine in sync.
type DerivationPipeline struct {
	log       log.Logger
	cfg       *rollup.Config
	l1Fetcher L1Fetcher

	// Index of the stage that is currently being reset.
	// >= len(stages) if no additional resetting is required
	resetting    int
	pullResetIdx int

	// Index of the stage that is currently being processed.
	active int

	// stages in execution order. A stage Step that:
	stages []Stage

	pullStages []PullStage
	traversal  *L1Traversal

	eng EngineQueueStage

	metrics Metrics
}

// NewDerivationPipeline creates a derivation pipeline, which should be reset before use.
func NewDerivationPipeline(log log.Logger, cfg *rollup.Config, l1Fetcher L1Fetcher, engine Engine, metrics Metrics) *DerivationPipeline {

	// Pull stages
	l1Traversal := NewL1Traversal(log, l1Fetcher)
	dataSrc := NewDataSourceFactory(log, cfg, l1Fetcher) // auxiliary stage for L1Retrieval
	l1Src := NewL1Retrieval(log, dataSrc, l1Traversal)
	bank := NewChannelBank(log, cfg, l1Src, l1Fetcher)
	chInReader := NewChannelInReader(log, bank)
	batchQueue := NewBatchQueue(log, cfg, chInReader)
	attributesQueue := NewAttributesQueue(log, cfg, l1Fetcher, batchQueue)

	// Push stages (that act like pull stages b/c we push from the innermost stages prior to the outermost stages)
	eng := NewEngineQueue(log, cfg, engine, metrics, attributesQueue)

	stages := []Stage{eng}
	pullStages := []PullStage{attributesQueue, batchQueue, chInReader, bank, l1Src, l1Traversal}

	return &DerivationPipeline{
		log:        log,
		cfg:        cfg,
		l1Fetcher:  l1Fetcher,
		resetting:  0,
		active:     0,
		stages:     stages,
		pullStages: pullStages,
		eng:        eng,
		metrics:    metrics,
		traversal:  l1Traversal,
	}
}

func (dp *DerivationPipeline) Reset() {
	dp.resetting = 0
	dp.pullResetIdx = 0
}

func (dp *DerivationPipeline) Progress() Progress {
	return dp.eng.Progress()
}

func (dp *DerivationPipeline) Finalize(l1Origin eth.BlockID) {
	dp.eng.Finalize(l1Origin)
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

func (dp *DerivationPipeline) SetUnsafeHead(head eth.L2BlockRef) {
	dp.eng.SetUnsafeHead(head)
}

// AddUnsafePayload schedules an execution payload to be processed, ahead of deriving it from L1
func (dp *DerivationPipeline) AddUnsafePayload(payload *eth.ExecutionPayload) {
	dp.eng.AddUnsafePayload(payload)
}

// Step tries to progress the buffer.
// An EOF is returned if there pipeline is blocked by waiting for new L1 data.
// If ctx errors no error is returned, but the step may exit early in a state that can still be continued.
// Any other error is critical and the derivation pipeline should be reset.
// An error is expected when the underlying source closes.
// When Step returns nil, it should be called again, to continue the derivation process.
func (dp *DerivationPipeline) Step(ctx context.Context) error {
	defer dp.metrics.RecordL1Ref("l1_derived", dp.Progress().Origin)

	// if any stages need to be reset, do that first.
	if dp.resetting < len(dp.stages) {
		if err := dp.stages[dp.resetting].ResetStep(ctx, dp.l1Fetcher); err == io.EOF {
			dp.log.Debug("reset of stage completed", "stage", dp.resetting, "origin", dp.stages[dp.resetting].Progress().Origin)
			dp.resetting += 1
			return nil
		} else if err != nil {
			return fmt.Errorf("stage %d failed resetting: %w", dp.resetting, err)
		} else {
			return nil
		}
	}
	// Then reset the pull based stages
	if dp.pullResetIdx < len(dp.pullStages) {
		// Use the last stage's progress as the one to pull from
		inner := dp.stages[len(dp.stages)-1].Progress()

		// Do the reset
		if err := dp.pullStages[dp.pullResetIdx].Reset(ctx, inner.Origin); err == io.EOF {
			// dp.log.Debug("reset of stage completed", "stage", dp.pullResetIdx, "origin", dp.pullStages[dp.pullResetIdx].Progress().Origin)
			dp.pullResetIdx += 1
			return nil
		} else if err != nil {
			return fmt.Errorf("stage %d failed resetting: %w", dp.pullResetIdx, err)
		} else {
			return nil
		}
	}

	// Lastly advance the stages
	for i, stage := range dp.stages {
		var outer Progress
		if i+1 < len(dp.stages) {
			outer = dp.stages[i+1].Progress()
		}
		if err := stage.Step(ctx, outer); err == io.EOF {
			continue
		} else if err != nil {
			return fmt.Errorf("stage %d failed: %w", i, err)
		} else {
			return nil
		}
	}
	// If every stage has returned io.EOF, try to advance the L1 Origin
	return dp.traversal.AdvanceL1Block(ctx)
}
