package derive

import (
	"context"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"io"
	"math/big"
)

type Engine interface {
	GetPayload(ctx context.Context, payloadId eth.PayloadID) (*eth.ExecutionPayload, error)
	ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)
	NewPayload(ctx context.Context, payload *eth.ExecutionPayload) error
	PayloadByHash(context.Context, common.Hash) (*eth.ExecutionPayload, error)
	PayloadByNumber(context.Context, *big.Int) (*eth.ExecutionPayload, error)
}

// DerivationPipeline is updated with new L1 data, and the Step() function can be iterated on to keep the L2 Engine in sync.
type DerivationPipeline struct {
	log         log.Logger
	nextL1      func() eth.L1BlockRef    // Where we fetch new L1 data from
	bank        *ChannelBank             // Where we buffer all incoming L1 data
	batchReader *ChannelInReader         // Where we buffer tagged data to read batches from (TODO: this API is blocking, we can't handle that currently)
	batchQueue  *BatchQueue              // Where we buffer all derived L2 batches
	attributes  []*eth.PayloadAttributes // Where we buffer all derived payload attributes
	engine      Engine                   // Final destination: the execution engine (EVM + chain and state DB)
}

func (dp *DerivationPipeline) Reset(l2Head eth.L2BlockRef) error {
	// TODO: clear all contents of the pipeline, and prepare for deriving data on top of the given L2 head.
	return nil
}

// Step tries to progress the buffer.
// When no error is returned, the buffer is ready for the next Step() immediately.
//
// When io.EOF is returned, the buffered data is exhausted to a point where no new L2 payload
// can be derived without more L1 data first. Step() should not be called until new L1 data.
//
// Other errors are critical, and the caller should reset the derivation process.
func (dp *DerivationPipeline) Step() error {
	// try to apply previous buffered information to the engine
	if err := dp.applyToEngine(); err != io.EOF {
		return err
	}
	// try to derive new payload attributes from buffered batch(es)
	if err := dp.readAttributes(); err != io.EOF {
		return err
	}
	// read a batch from buffered tagged data
	if err := dp.readBatch(); err != io.EOF {
		return err
	}
	if err := dp.readNextL1Origin(); err != io.EOF {
		return err
	}
	return dp.readL1InputData()
}

func (dp *DerivationPipeline) readL1InputData() error {
	// 1. if all data has been fetched already, return io.EOF
	// 2. if not, fetch remaining data
	//    2.1 if fetching error, log it and return nil
	//    2.2 if new data, buffer it
	// 3. when data is complete, flush it to the channel bank
	return nil
}

func (dp *DerivationPipeline) readNextL1Origin() error {
	// 1. try to read the next canonical L1 origin
	//   1.1 return io.EOF if there is no new L1 origin yet
	// 2. check if the parent hash matches the current origin
	//   2.1 return an error if it does not match
	// 3. update the BatchQueue with the new origin
	// 4. update the ChannelBank with the new origin
	return nil
}

func (dp *DerivationPipeline) readBatch() error {
	// TODO: implement below spec
	// 1. try to read a batch from the ChannelInReader
	// 2. if no batch was returned, return io.EOF
	// 3. if a batch was returned:
	//   3.1 get the CurrentOrigin from the reader, and update the BatchQueue with all origins since the
	return nil
}

func (dp *DerivationPipeline) readAttributes() error {
	// TODO: implement below spec
	// 1. try to derive payload attributes from the BatchQueue
	// 2. if none were returned, return io.EOF
	// 3. if an error was returned (e.g. failed to fetch L1 receipts), log it and return nil
	// 4. if any were returned, append them to the queue of attributes to be consolidated/applied to the engine, and return nil
	return nil
}

func (dp *DerivationPipeline) applyToEngine() error {
	// TODO: implement below spec  (and put the buffer of payload attributes into a separate struct type from the L2 derivation, to test it in isolation)
	// 1. return io.EOF if there are no payload attributes buffered
	// 2. peek into first payload attributes
	// 3. check if the engine has synced past these attributes
	//     3.1 if yes, compare the engine attributes
	//        3.1.1 mark the attributes as safe (forkchoice update, without changing unsafe head) (with timeout)
	//             or log error and return nil if this fails
	//        3.1.2 pop the attributes from buffer
	//        3.1.3 log what we just consolidated
	//        3.1.4 pop from the buffer
	//        3.1.5 return nil
	//     3.2 if not, re-apply the engine attributes
	//        3.2.1 forkchoice update to make the safe block the head (with timeout)
	//             or log error and return nil if this fails
	//        3.2.2 try to apply the payload attributes to the engine (with timeout)
	//             if RPC error: log it, and return nil
	//        3.2.3 if invalid payload: log it (err level), pop it, and then return nil
	//        3.2.4 if valid payload: log it (info level), pop it, and then return nil
	return nil
}
