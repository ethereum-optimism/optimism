package derive

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/rlp"
)

// TODO: Full state machine for channel
// Open, Closed, Ready (todo - when to construct RLP reader), Batches Read

// A Channel is a set of batches that are split into at least one, but possibly multiple frames.
// Frames are allowed to be ingested out of order.
// Each frame is ingested one by one. Once a frame with `closed` is added to the channel, the
// channel may mark itself as ready for reading once all intervening frames have been added
type Channel struct {
	// id of the channel
	id ChannelID

	// estimated memory size, used to drop the channel if we have too much data
	size uint64

	// true if we have buffered the last frame
	closed bool

	// TODO: implement this check
	// highestFrameNumber is the highest frame number yet seen.
	// This must be equal to `endFrameNumber`
	// highestFrameNumber uint16

	// endFrameNumber is the frame number of the frame where `isLast` is true
	// No other frame number must be larger than this.
	endFrameNumber uint16

	// Store a map of frame number -> frame data for constant time ordering
	inputs map[uint64][]byte

	highestL1InclusionBlock eth.L1BlockRef

	// Output stream. Constructed at `ReadBatch` if not nil
	readRLP *rlp.Stream
}

func NewChannel(id ChannelID) *Channel {
	return &Channel{
		id:     id,
		inputs: make(map[uint64][]byte),
	}
}

// AddFrame adds a frame to the channel.
// If the frame is not valid for the channel it returns an error.
// Otherwise the frame is buffered.
func (ch *Channel) AddFrame(frame Frame, l1InclusionBlock eth.L1BlockRef) error {
	if frame.ID != ch.id {
		return fmt.Errorf("frame id does not match channel id. Expected %v, got %v", ch.id, frame.ID)
	}
	if frame.IsLast && ch.closed {
		return fmt.Errorf("cannot add ending frame to a closed channel. id %v", ch.id)
	}
	if _, ok := ch.inputs[uint64(frame.FrameNumber)]; ok {
		return DuplicateErr
	}
	// TODO: highest seen frame vs endFrameNumber

	// Guaranteed to succeed. Now update internal state
	if frame.IsLast {
		ch.endFrameNumber = frame.FrameNumber
	}
	if ch.highestL1InclusionBlock.Number < l1InclusionBlock.Number {
		ch.highestL1InclusionBlock = l1InclusionBlock
	}
	ch.inputs[uint64(frame.FrameNumber)] = frame.Data
	ch.size += uint64(len(frame.Data)) + frameOverhead

	// todo use `IsReady` + state to create final output reader

	return nil
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
	if len(ch.inputs) != int(ch.endFrameNumber) {
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

// Read full channel content (it may be incomplete if the channel is not Closed)
// This should only be called after `IsReady` returns true.
func (ch *Channel) Read() (out []byte) {
	for i := uint64(0); i <= uint64(ch.endFrameNumber); i++ {
		data, ok := ch.inputs[i]
		if !ok {
			// TODO: dev error here
			return
		}
		out = append(out, data...)
	}
	return out
}

// ReadBatch returns a decoded rollup batch, or an error:
//   - io.EOF when the channel is empty
//   - any other error (e.g. invalid compression or batch data)
// The L1 Inclusion block is the highest L1 inclusion block inside the channel.
func (ch *Channel) ReadBatch() (BatchWithL1InclusionBlock, error) {
	if ch.readRLP == nil {
		var readers []io.Reader
		for i := uint64(0); i <= uint64(ch.endFrameNumber); i++ {
			data, ok := ch.inputs[i]
			if !ok {
				return BatchWithL1InclusionBlock{}, errors.New("dev error")
			}
			readers = append(readers, bytes.NewBuffer(data))
		}
		zr, err := zlib.NewReader(io.MultiReader(readers...))
		if err != nil {
			return BatchWithL1InclusionBlock{}, err
		}
		ch.readRLP = rlp.NewStream(zr, 10_000_000)
	}

	ret := BatchWithL1InclusionBlock{
		L1InclusionBlock: ch.highestL1InclusionBlock,
	}
	err := ch.readRLP.Decode(&ret.Batch)
	return ret, err
}
