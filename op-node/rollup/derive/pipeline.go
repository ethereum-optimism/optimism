package derive

import (
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/log"
	"io"
)

// DerivationPipeline is updated with new L1 data, and the Step() function can be iterated on to keep the L2 Engine in sync.
type DerivationPipeline struct {
	log         log.Logger
	batchReader *ChannelInReader // Where we buffer tagged data to read batches from, this is blocking.
	batchQueue  *BatchQueue      // Where we buffer all derived L2 batches
	engineQueue *EngineQueue     // Where we buffer payload attributes, and apply/consolidate them with the L2 engine
}

func (dp *DerivationPipeline) Reset(l2Head eth.L2BlockRef) error {
	// TODO: clear all contents of the pipeline, and prepare for deriving data on top of the given L2 head.
	return nil
}

// Step tries to progress the buffer.
// An error is critical and the derivation pipeline should be reset.
// An error is expected when the underlying source closes.
func (dp *DerivationPipeline) Step() error {
	for {
		// try to apply previous buffered information to the engine
		if err := dp.engineQueue.Step(); err == nil {
			continue
		} else if err != io.EOF {
			return fmt.Errorf("critical failure while applying payload attributes to engine: %w", err)
		}
		// try to derive new payload attributes from buffered batch(es)
		if err := dp.readAttributes(); err == nil {
			continue
		} else if err != io.EOF {
			return fmt.Errorf("critical failure while reading payload attributes: %w", err)
		}
		// read a batch from buffered tagged data.
		// This step may be blocking until additional L1 data is available.
		return dp.readBatch()
	}
}

func (dp *DerivationPipeline) readBatch() error {
	// TODO: implement below spec
	// 1. try to read a batch from the ChannelInReader
	// 2. if no batch was returned, return io.EOF
	// 3. if a batch was returned:
	//   3.1 get the CurrentOrigin from the reader, and update the BatchQueue with all origins since then

	return nil
}

func (dp *DerivationPipeline) readAttributes() error {
	// TODO: implement below spec
	// 1. try to derive payload attributes from the BatchQueue
	// 2. if none were returned, return io.EOF
	// 3. if an error was returned (e.g. failed to fetch L1 receipts), log it and return nil
	// 4. if any were returned, append them to the engineQueue, and return nil

	//dp.batchQueue.DeriveL2Inputs()

	return nil
}
