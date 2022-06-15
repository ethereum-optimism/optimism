package derive

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"
	"io"
)

// TODO: replace pipeline field types with interfaces, to test the pipeline with mocked stages.

type DataAvailabilitySource interface {
	Fetch(ctx context.Context, id eth.BlockID) (eth.L1BlockRef, []eth.Data, error)
}

type L1InfoFetcher interface {
	InfoByNumber(ctx context.Context, number uint64) (L1Info, error)
}

type L1Fetcher interface {
	L1InfoFetcher
	L1ReceiptsFetcher
	L1TransactionFetcher
}

// DerivationPipeline is updated with new L1 data, and the Step() function can be iterated on to keep the L2 Engine in sync.
type DerivationPipeline struct {
	log         log.Logger
	cfg         *rollup.Config
	l1InfoSrc   L1InfoFetcher          // Where we traverse the L1 chain with to the next L1 input
	dataAvail   DataAvailabilitySource // Where we extract L1 data from
	bank        *ChannelBank           // Where we buffer L1 data to read channel data from
	chInReader  *ChannelInReader       // Where we buffer channel data to read batches from
	batchQueue  *BatchQueue            // Where we buffer all derived L2 batches
	engineQueue *EngineQueue           // Where we buffer payload attributes, and apply/consolidate them with the L2 engine
}

// NewDerivationPipeline creates a derivation pipeline, which should be reset before use.
func NewDerivationPipeline(log log.Logger, cfg *rollup.Config, l1Src L1Fetcher, engine Engine) *DerivationPipeline {
	return &DerivationPipeline{
		log:         log,
		cfg:         cfg,
		l1InfoSrc:   l1Src,
		dataAvail:   NewCalldataSource(log, cfg, l1Src), // this may change to pull EIP-4844 data (or pull both)
		bank:        NewChannelBank(log),
		chInReader:  NewChannelInReader(),
		batchQueue:  NewBatchQueue(log, cfg, l1Src),
		engineQueue: NewEngineQueue(log, cfg, engine),
	}
}

func (dp *DerivationPipeline) Reset(ctx context.Context, l1SafeHead eth.L1BlockRef) error {
	// TODO: determine l2SafeHead
	var l2SafeHead eth.L2BlockRef

	// TODO: requires replay of data to get into consistent state again
	dp.bank.Reset(l1SafeHead)

	dp.chInReader.Reset(l1SafeHead)
	dp.batchQueue.Reset(l1SafeHead)
	dp.engineQueue.Reset(l2SafeHead)
	return nil
}

func (dp *DerivationPipeline) CurrentL1() eth.L1BlockRef {
	return dp.bank.CurrentL1()
}

func (dp *DerivationPipeline) Finalize(l1Origin eth.BlockID) {
	dp.engineQueue.Finalize(l1Origin)
}

func (dp *DerivationPipeline) Finalized() eth.L2BlockRef {
	return dp.engineQueue.Finalized()
}

func (dp *DerivationPipeline) SafeL2Head() eth.L2BlockRef {
	return dp.engineQueue.SafeL2Head()
}

// UnsafeL2Head returns the head of the L2 chain that we are deriving for, this may be past what we derived from L1
func (dp *DerivationPipeline) UnsafeL2Head() eth.L2BlockRef {
	return dp.engineQueue.UnsafeL2Head()
}

// AddUnsafePayload schedules an execution payload to be processed, ahead of deriving it from L1
func (dp *DerivationPipeline) AddUnsafePayload(payload *eth.ExecutionPayload) {
	dp.engineQueue.AddUnsafePayload(payload)
}

// Step tries to progress the buffer.
// An EOF is returned if there pipeline is blocked by waiting for new L1 data.
// If ctx errors no error is returned, but the step may exit early in a state that can still be continued.
// Any other error is critical and the derivation pipeline should be reset.
// An error is expected when the underlying source closes.
// When Step returns nil, it should be called again, to continue the derivation process.
func (dp *DerivationPipeline) Step(ctx context.Context) error {
	// try to apply previous buffered information to the engine
	if err := dp.engineQueue.Step(ctx); err == nil {
		return nil
	} else if err != io.EOF {
		return fmt.Errorf("critical failure while applying payload attributes to engine: %w", err)
	}
	// try to derive new payload attributes from buffered batch(es)
	if err := dp.readAttributes(ctx); err == nil {
		return nil
	} else if err != io.EOF {
		return fmt.Errorf("critical failure while reading payload attributes: %w", err)
	}
	// read a batch from buffered tagged data.
	if err := dp.readBatch(); err == nil {
		return nil
	} else if err != io.EOF {
		return fmt.Errorf("critical failure while reading batch: %w", err)
	}
	if err := dp.readChannel(); err == nil {
		return nil
	} else if err != io.EOF {
		return fmt.Errorf("critical failure while reading channel: %w", err)
	}
	return dp.readL1(ctx)
}

func (dp *DerivationPipeline) readL1(ctx context.Context) error {
	current := dp.bank.CurrentL1()

	// TODO: we need to add confirmation depth in the source here, and return ethereum.NotFound when the data is not ready to be read.

	l1Info, err := dp.l1InfoSrc.InfoByNumber(ctx, current.Number+1)
	if errors.Is(err, ethereum.NotFound) {
		dp.log.Debug("can't find next L1 block info", "number", current.Number+1)
		return io.EOF
	} else if err != nil {
		dp.log.Warn("failed to find L1 block info by number", "number", current.Number+1)
		return nil
	}

	if l1Info.ParentHash() != current.Hash {
		// reorg, time to reset the pipeline
		return fmt.Errorf("reorg on L1, found %s with parent %s but expected parent to be %s",
			l1Info.Hash(), l1Info.ParentHash(), current.Hash)
	}
	id := l1Info.ID()
	_, datas, err := dp.dataAvail.Fetch(ctx, id)
	if err != nil {
		dp.log.Debug("can't fetch L1 data", "origin", id)
		return nil
	}
	if err := dp.bank.NextL1(l1Info.BlockRef()); err != nil {
		return err
	}
	for i, data := range datas {
		if err := dp.bank.IngestData(data); err != nil {
			dp.log.Warn("invalid data from availability source", "origin", id, "index", i, "length", len(data))
		}
	}
	return nil
}

func (dp *DerivationPipeline) readChannel() error {
	// move forward the ch reader if the bank has new L1 data
	if dp.chInReader.CurrentL1Origin() != dp.bank.CurrentL1() {
		return dp.chInReader.AddOrigin(dp.bank.CurrentL1())
	}
	// otherwise, read the next channel data from the bank
	id, data := dp.bank.Read()
	if id == (ChannelID{}) { // need new L1 data in the bank before we can read more channel data
		return io.EOF
	}
	dp.chInReader.WriteChannel(data)
	return nil
}

func (dp *DerivationPipeline) readBatch() error {
	// move forward the batch queue if the ch reader has new L1 data
	if dp.batchQueue.LastL1Origin() != dp.chInReader.CurrentL1Origin() {
		return dp.batchQueue.AddOrigin(dp.chInReader.CurrentL1Origin())
	}
	var batch BatchData
	if err := dp.chInReader.ReadBatch(&batch); err == io.EOF {
		return io.EOF
	} else if err != nil {
		dp.log.Warn("failed to read batch from channel reader, skipping to next channel now", "err", err)
		dp.chInReader.NextChannel()
		return nil
	}
	return dp.batchQueue.AddBatch(&batch)
}

func (dp *DerivationPipeline) readAttributes(ctx context.Context) error {
	attrs, err := dp.batchQueue.DeriveL2Inputs(ctx, dp.engineQueue.LastL2Time())
	if err != nil {
		return err
	}
	for _, attr := range attrs {
		dp.engineQueue.AddSafeAttributes(attr)
	}
	return nil
}
