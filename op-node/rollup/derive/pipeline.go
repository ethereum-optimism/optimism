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

type L1BlockRefByNumberFetcher interface {
	L1BlockRefByNumber(context.Context, uint64) (eth.L1BlockRef, error)
}

type L1Fetcher interface {
	L1BlockRefByNumberFetcher
	L1BlockRefByHashFetcher
	L1ReceiptsFetcher
	L1TransactionFetcher
}

type DataAvailabilitySource interface {
	Fetch(ctx context.Context, id eth.BlockID) (eth.L1BlockRef, []eth.Data, error)
}

type ChannelBankStage interface {
	CurrentL1() eth.L1BlockRef
	NextL1(ref eth.L1BlockRef) error
	IngestData(data []byte) error

	Read() (chID ChannelID, data []byte)
	Reset(origin eth.L1BlockRef)
}

type ChannelInReaderStage interface {
	CurrentL1Origin() eth.L1BlockRef
	AddOrigin(origin eth.L1BlockRef) error
	WriteChannel(data []byte)
	NextChannel()
	ReadBatch(dest *BatchData) error
	Reset(origin eth.L1BlockRef)
}

type BatchQueueStage interface {
	LastL1Origin() eth.L1BlockRef
	AddOrigin(origin eth.L1BlockRef) error
	AddBatch(batch *BatchData) error
	DeriveL2Inputs(ctx context.Context, lastL2Timestamp uint64) ([]*eth.PayloadAttributes, error)
	Reset(l1Origin eth.L1BlockRef)
}

type EngineQueueStage interface {
	Finalized() eth.L2BlockRef
	UnsafeL2Head() eth.L2BlockRef
	SafeL2Head() eth.L2BlockRef
	LastL2Time() uint64

	Finalize(l1Origin eth.BlockID)
	AddSafeAttributes(attributes *eth.PayloadAttributes)
	AddUnsafePayload(payload *eth.ExecutionPayload)

	Step(ctx context.Context) error
	Reset(safeHead eth.L2BlockRef)
}

// DerivationPipeline is updated with new L1 data, and the Step() function can be iterated on to keep the L2 Engine in sync.
type DerivationPipeline struct {
	log         log.Logger
	cfg         *rollup.Config
	l1InfoSrc   L1Fetcher              // Where we traverse the L1 chain with to the next L1 input
	dataAvail   DataAvailabilitySource // Where we extract L1 data from
	bank        ChannelBankStage       // Where we buffer L1 data to read channel data from
	chInReader  ChannelInReaderStage   // Where we buffer channel data to read batches from
	batchQueue  BatchQueueStage        // Where we buffer all derived L2 batches
	engineQueue EngineQueueStage       // Where we buffer payload attributes, and apply/consolidate them with the L2 engine
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

func (dp *DerivationPipeline) Reset(ctx context.Context, l2SafeHead eth.L2BlockRef) error {
	l1SafeHead, err := dp.l1InfoSrc.L1BlockRefByHash(ctx, l2SafeHead.L1Origin.Hash)
	if err != nil {
		return fmt.Errorf("failed to find L1 reference corresponding to L1 origin %s of L2 block %s: %v", l2SafeHead.L1Origin, l2SafeHead.ID(), err)
	}

	bankStart, err := FindChannelBankStart(ctx, l1SafeHead, dp.l1InfoSrc)
	if err != nil {
		return fmt.Errorf("failed to find channel bank start: %v", err)
	}

	// the bank will catch up first, before the remaining part of the pipeline will start getting data.
	dp.bank.Reset(bankStart)
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

	nextL1Origin, err := dp.l1InfoSrc.L1BlockRefByNumber(ctx, current.Number+1)
	if errors.Is(err, ethereum.NotFound) {
		dp.log.Debug("can't find next L1 block info", "number", current.Number+1)
		return io.EOF
	} else if err != nil {
		dp.log.Warn("failed to find L1 block info by number", "number", current.Number+1)
		return nil
	}

	if nextL1Origin.ParentHash != current.Hash {
		// reorg, time to reset the pipeline
		return fmt.Errorf("reorg on L1, found %s with parent %s but expected parent to be %s",
			nextL1Origin.ID(), nextL1Origin.ParentID(), current.ID())
	}
	_, datas, err := dp.dataAvail.Fetch(ctx, nextL1Origin.ID())
	if err != nil {
		dp.log.Debug("can't fetch L1 data", "origin", nextL1Origin)
		return nil
	}
	if err := dp.bank.NextL1(nextL1Origin); err != nil {
		return err
	}
	for i, data := range datas {
		if err := dp.bank.IngestData(data); err != nil {
			dp.log.Warn("invalid data from availability source", "origin", nextL1Origin, "index", i, "length", len(data))
		}
	}
	return nil
}

func (dp *DerivationPipeline) readChannel() error {
	// If the bank is behind the channel reader, then we are replaying old data to prepare the bank.
	// Read if we can, and drop if it gives anything
	if dp.chInReader.CurrentL1Origin().Number > dp.bank.CurrentL1().Number {
		id, _ := dp.bank.Read()
		if id == (ChannelID{}) {
			return io.EOF
		}
		return nil
	}

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
