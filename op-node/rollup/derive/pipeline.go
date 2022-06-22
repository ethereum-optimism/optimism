package derive

import (
	"context"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/log"
)

type L1Fetcher interface {
	L1BlockRefByNumberFetcher
	L1BlockRefByHashFetcher
	L1ReceiptsFetcher
	L1TransactionFetcher
}

type Stage interface {
	// Step tries to progress the state. If the stage:
	// - returns EOF: the stage will be skipped
	// - returns another error: the stage will make the pipeline error.
	// - returns nil: the stage will be repeated next Step
	Step(ctx context.Context) error

	// ResetStep prepares the state for usage in regular steps.
	// Similar to Step(ctx) it returns:
	// - EOF if the next stage should be reset
	// - error if the reset should start all over again
	// - nil if the reset should continue resetting this stage.
	ResetStep(ctx context.Context, l1Fetcher L1Fetcher) error
}

type Origin interface {
	// CurrentOrigin returns which L1 origin the stage has currently synced up to.
	CurrentOrigin() eth.L1BlockRef
}

type OriginStage interface {
	Origin
	// OpenOrigin is called before any data included in the origin is written to the stage
	OpenOrigin(origin eth.L1BlockRef) error
	// CloseOrigin is called if no more data included in the current origin will be written to the stage
	CloseOrigin()
	// IsOriginOpen returns true when CloseOrigin has not yet been called for the current origin.
	IsOriginOpen() bool
}

type EngineQueueStage interface {
	Finalized() eth.L2BlockRef
	UnsafeL2Head() eth.L2BlockRef
	SafeL2Head() eth.L2BlockRef
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

	// start references the closest-to-L1 stage, where we consume a new L1 origin soonest from
	start Origin

	// stages in execution order. A stage Step that:
	stages []Stage

	eng EngineQueueStage
}

// NewDerivationPipeline creates a derivation pipeline, which should be reset before use.
func NewDerivationPipeline(log log.Logger, cfg *rollup.Config, l1Fetcher L1Fetcher, engine Engine) *DerivationPipeline {
	eng := NewEngineQueue(log, cfg, engine)
	batchQueue := NewBatchQueue(log, cfg, l1Fetcher, eng)
	chInReader := NewChannelInReader(log, batchQueue)
	bank := NewChannelBank(log, cfg, chInReader)
	dataSrc := NewCalldataSource(log, cfg, l1Fetcher)
	l1Src := NewL1Source(log, l1Fetcher, dataSrc, bank)
	stages := []Stage{eng, batchQueue, chInReader, bank, l1Src}

	return &DerivationPipeline{
		log:       log,
		cfg:       cfg,
		l1Fetcher: l1Fetcher,
		resetting: 0,
		start:     l1Src,
		stages:    stages,
		eng:       eng,
	}
}

func (dp *DerivationPipeline) Reset() {
	dp.resetting = 0
}

func (dp *DerivationPipeline) CurrentL1() eth.L1BlockRef {
	return dp.start.CurrentOrigin()
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
			dp.resetting += 1
			return nil
		} else if err != nil {
			return err
		} else {
			return nil
		}
	}

	for _, stage := range dp.stages {
		if err := stage.Step(ctx); err == io.EOF {
			continue
		} else if err != nil {
			return err
		} else {
			return nil
		}
	}
	return io.EOF
}
