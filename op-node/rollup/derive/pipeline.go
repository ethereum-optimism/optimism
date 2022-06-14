package derive

import (
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/log"
	"io"
)

// TODO: replace pipeline field types with interfaces, to test the pipeline with mocked stages.

// DerivationPipeline is updated with new L1 data, and the Step() function can be iterated on to keep the L2 Engine in sync.
type DerivationPipeline struct {
	log         log.Logger
	bank        *ChannelBank
	chInReader  *ChannelInReader // Where we buffer tagged data to read batches from
	batchQueue  *BatchQueue      // Where we buffer all derived L2 batches
	engineQueue *EngineQueue     // Where we buffer payload attributes, and apply/consolidate them with the L2 engine
}

func (dp *DerivationPipeline) Reset(l2SafeHead eth.L2BlockRef) error {
	// TODO: clear/reset all contents of the pipeline
	return nil
}

func (dp *DerivationPipeline) CurrentL1() eth.L1BlockRef {
	return dp.bank.CurrentL1()
}

func (dp *DerivationPipeline) SafeL2Head() eth.L2BlockRef {
	return dp.engineQueue.SafeL2Head()
}

// Step tries to progress the buffer.
// An EOF is returned if there pipeline is blocked by retrieving new data from L1.
// Any other error is critical and the derivation pipeline should be reset.
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
		if err := dp.readBatch(); err == nil {
			continue
		} else if err != io.EOF {
			return fmt.Errorf("critical failure while reading batch: %w", err)
		}
		return dp.readChannel()
	}
}

func (dp *DerivationPipeline) readChannel() error {
	// TODO: implement below spec
	// 1. try to read channel data from the ChannelBank
	// 2. if no data was returned, try read the L1 origin and move forward the batch reader
	// 3. if the L1 origin did not change, then return io.EOF
	// 4. if data was returned, reset the channel reader with it
	return nil
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
