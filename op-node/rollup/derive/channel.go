package derive

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/rlp"
)

// A Channel is a set of batches that are split into at least one, but possibly multiple frames.
// Frames are allowed to be ingested out of order.
// Each frame is ingested one by one. Once a frame with `closed` is added to the channel, the
// channel may mark itself as ready for reading once all intervening frames have been added
type Channel struct {
	// id of the channel
	id        ChannelID
	openBlock eth.L1BlockRef

	// estimated memory size, used to drop the channel if we have too much data
	size uint64

	// true if we have buffered the last frame
	closed bool

	// highestFrameNumber is the highest frame number yet seen.
	highestFrameNumber uint16

	// endFrameNumber is the frame number of the frame where `isLast` is true
	// No other frame number must be larger than this.
	endFrameNumber uint16

	// Store a map of frame number -> frame for constant time ordering
	inputs map[uint64]Frame

	highestL1InclusionBlock eth.L1BlockRef
}

func NewChannel(id ChannelID, openBlock eth.L1BlockRef) *Channel {
	return &Channel{
		id:        id,
		inputs:    make(map[uint64]Frame),
		openBlock: openBlock,
	}
}

// AddFrame adds a frame to the channel.
// If the frame is not valid for the channel it returns an error.
// Otherwise the frame is buffered.
func (ch *Channel) AddFrame(frame Frame, l1InclusionBlock eth.L1BlockRef) error {
	if frame.ID != ch.id {
		return fmt.Errorf("frame id does not match channel id. Expected %v, got %v", ch.id, frame.ID)
	}
	// These checks are specified and cannot be changed without a hard fork.
	if frame.IsLast && ch.closed {
		return fmt.Errorf("cannot add ending frame to a closed channel. id %v", ch.id)
	}
	if _, ok := ch.inputs[uint64(frame.FrameNumber)]; ok {
		return DuplicateErr
	}
	if ch.closed && frame.FrameNumber >= ch.endFrameNumber {
		return fmt.Errorf("frame number (%d) is greater than or equal to end frame number (%d) of a closed channel", frame.FrameNumber, ch.endFrameNumber)
	}

	// Guaranteed to succeed. Now update internal state
	if frame.IsLast {
		ch.endFrameNumber = frame.FrameNumber
		ch.closed = true
	}
	// Prune frames with a number higher than the closing frame number when we receive a closing frame
	if frame.IsLast && ch.endFrameNumber < ch.highestFrameNumber {
		// Do a linear scan over saved inputs instead of ranging over ID numbers
		for id, prunedFrame := range ch.inputs {
			if id >= uint64(ch.endFrameNumber) {
				delete(ch.inputs, id)
			}
			ch.size -= frameSize(prunedFrame)
		}
		ch.highestFrameNumber = ch.endFrameNumber
	}
	// Update highest seen frame number after pruning
	if frame.FrameNumber > ch.highestFrameNumber {
		ch.highestFrameNumber = frame.FrameNumber
	}

	if ch.highestL1InclusionBlock.Number < l1InclusionBlock.Number {
		ch.highestL1InclusionBlock = l1InclusionBlock
	}
	ch.inputs[uint64(frame.FrameNumber)] = frame
	ch.size += frameSize(frame)

	return nil
}

// OpenBlockNumber returns the block number of L1 block that contained
// the first frame for this channel.
func (ch *Channel) OpenBlockNumber() uint64 {
	return ch.openBlock.Number
}

// Size returns the current size of the channel including frame overhead.
// Reading from the channel does not reduce the size as reading is done
// on uncompressed data while this size is over compressed data.
func (ch *Channel) Size() uint64 {
	return ch.size
}

// IsReady returns true iff the channel is ready to be read.
func (ch *Channel) IsReady() bool {
	// Must see the last frame before the channel is ready to be read
	if !ch.closed {
		return false
	}
	// Must have the possibility of contiguous frames
	if len(ch.inputs) != int(ch.endFrameNumber)+1 {
		return false
	}
	// Check for contiguous frames
	for i := uint64(0); i <= uint64(ch.endFrameNumber); i++ {
		_, ok := ch.inputs[i]
		if !ok {
			return false
		}
	}
	return true
}

// Reader returns an io.Reader over the channel data.
// This panics if it is called while `IsReady` is not true.
// This function is able to be called multiple times.
func (ch *Channel) Reader() io.Reader {
	var readers []io.Reader
	for i := uint64(0); i <= uint64(ch.endFrameNumber); i++ {
		frame, ok := ch.inputs[i]
		if !ok {
			panic("dev error in channel.Reader. Must be called after the channel is ready.")
		}
		readers = append(readers, bytes.NewReader(frame.Data))
	}
	return io.MultiReader(readers...)
}

// BatchReader provides a function that iteratively consumes batches from the reader.
// The L1Inclusion block is also provided at creation time.
// Warning: the batch reader can read every batch-type.
// The caller of the batch-reader should filter the results.
func BatchReader(r io.Reader) (func() (*BatchData, error), error) {
	// Setup decompressor stage + RLP reader
	zr, err := zlib.NewReader(r)
	if err != nil {
		return nil, err
	}
	rlpReader := rlp.NewStream(zr, MaxRLPBytesPerChannel)
	// Read each batch iteratively
	return func() (*BatchData, error) {
		var batchData BatchData
		if err = rlpReader.Decode(&batchData); err != nil {
			return nil, err
		}
		return &batchData, nil
	}, nil
}
