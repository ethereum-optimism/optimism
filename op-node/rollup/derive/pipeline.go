package derive

import (
	"context"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/log"
)

type L1Fetcher interface {
	L1HeadBlockRef(ctx context.Context) (eth.L1BlockRef, error)
	L1BlockRefByNumberFetcher
	L1BlockRefByHashFetcher
	L1ReceiptsFetcher
	L1TransactionFetcher
}

type StageProgress interface {
	Progress() Progress
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
	resetting int

	// Index of the stage that is currently being processed.
	active int

	// stages in execution order. A stage Step that:
	stages []Stage

	eng EngineQueueStage
}

// NewDerivationPipeline creates a derivation pipeline, which should be reset before use.
func NewDerivationPipeline(log log.Logger, cfg *rollup.Config, l1Fetcher L1Fetcher, engine Engine) *DerivationPipeline {
	eng := NewEngineQueue(log, cfg, engine)
	attributesQueue := NewAttributesQueue(log, cfg, l1Fetcher, eng)
	batchQueue := NewBatchQueue(log, cfg, l1Fetcher, attributesQueue)
	chInReader := NewChannelInReader(log, batchQueue)
	bank := NewChannelBank(log, cfg, chInReader)
	dataSrc := NewCalldataSource(log, cfg, l1Fetcher)
	l1Src := NewL1Retrieval(log, dataSrc, bank)
	l1Traversal := NewL1Traversal(log, l1Fetcher, l1Src)
	stages := []Stage{eng, attributesQueue, batchQueue, chInReader, bank, l1Src, l1Traversal}

	return &DerivationPipeline{
		log:       log,
		cfg:       cfg,
		l1Fetcher: l1Fetcher,
		resetting: 0,
		active:    0,
		stages:    stages,
		eng:       eng,
	}
}

func (dp *DerivationPipeline) Reset() {
	dp.resetting = 0
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
	return io.EOF
}
