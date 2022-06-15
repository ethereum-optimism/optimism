package derive

import (
	"context"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/log"
	"io"
)

// TODO: replace pipeline field types with interfaces, to test the pipeline with mocked stages.

// DerivationPipeline is updated with new L1 data, and the Step() function can be iterated on to keep the L2 Engine in sync.
type DerivationPipeline struct {
	log         log.Logger
	bank        *ChannelBank     // Where we buffer L1 data to read channel data from
	chInReader  *ChannelInReader // Where we buffer channel data to read batches from
	batchQueue  *BatchQueue      // Where we buffer all derived L2 batches
	engineQueue *EngineQueue     // Where we buffer payload attributes, and apply/consolidate them with the L2 engine
}

func NewDerivationPipeline() *DerivationPipeline {
	// TODO
	return nil
}

func (dp *DerivationPipeline) Reset(ctx context.Context, l2SafeHead eth.L2BlockRef) error {
	// TODO: determine l1SafeHead
	var l1SafeHead eth.L1BlockRef
	bank, err := NewChannelBank(ctx, dp.log, l1SafeHead, nil, nil) // TODO
	if err != nil {
		return fmt.Errorf("failed to init channel bank: %w", err)
	}
	dp.bank = bank
	dp.chInReader.ResetOrigin(l1SafeHead)
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
// An EOF is returned if there pipeline is blocked by retrieving new data from L1.
// If ctx errors no error is returned, but the step may exit early in a state that can still be continued.
// Any other error is critical and the derivation pipeline should be reset.
// An error is expected when the underlying source closes.
func (dp *DerivationPipeline) Step(ctx context.Context) error {
	for {
		// try to apply previous buffered information to the engine
		if err := dp.engineQueue.Step(ctx); err == nil {
			continue
		} else if err != io.EOF {
			return fmt.Errorf("critical failure while applying payload attributes to engine: %w", err)
		}
		// try to derive new payload attributes from buffered batch(es)
		if err := dp.readAttributes(ctx); err == nil {
			continue
		} else if err != io.EOF {
			return fmt.Errorf("critical failure while reading payload attributes: %w", err)
		}
		// read a batch from buffered tagged data.
		if err := dp.readBatch(); err == nil {
			continue
		} else if err != io.EOF {
			return fmt.Errorf("critical failure while reading batch: %w", err)
		}
		return dp.readChannel()
	}
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
	dp.chInReader.ResetChannel(data)
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
		dp.log.Warn("failed to read batch from channel reader, resetting it", "err", err)
		dp.chInReader.Reset()
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
