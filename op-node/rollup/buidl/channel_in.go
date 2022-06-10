package buidl

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

type ChannelIn struct {
	// id of the channel
	id ChannelID

	// estimated memory size, used to drop the channel if we have too much data
	size uint64

	progress uint64

	firstSeen uint64

	// final frame number (inclusive). Max value if we haven't seen the end yet.
	endsAt uint64

	inputs map[uint64]*TaggedData
}

// IngestData buffers a frame in the channel, and potentially forwards it, along with any previously buffered frames
func (ch *ChannelIn) IngestData(ref eth.L1BlockRef, frameNum uint64, isLast bool, frameData []byte) error {
	if frameNum < ch.progress {
		// already consumed a frame with equal number, this must be a duplicate
		return DuplicateErr
	}
	if frameNum > ch.endsAt {
		return fmt.Errorf("channel already ended ingesting inputs")
	}
	// the frame is from the current or future, it will be read from the buffer later

	// create buffer if it didn't exist yet
	if ch.inputs == nil {
		ch.inputs = make(map[uint64]*TaggedData)
	}
	if _, exists := ch.inputs[frameNum]; exists {
		// already seen a frame for this channel with this frame number
		return DuplicateErr
	}
	// buffer the frame
	ch.inputs[frameNum] = &TaggedData{
		L1Origin:  ref,
		ChannelID: ch.id,
		Data:      frameData,
	}
	if isLast {
		ch.endsAt = frameNum
	}
	ch.size += uint64(len(frameData)) + frameOverhead
	return nil
}

// Read next tagged piece of data. This may return nil if there is none.
func (ch *ChannelIn) Read() *TaggedData {
	taggedData, ok := ch.inputs[ch.progress]
	if !ok {
		return nil
	}
	ch.size -= uint64(len(taggedData.Data)) + frameOverhead
	delete(ch.inputs, ch.progress)
	ch.progress += 1
	return taggedData
}

// Closed returns if this channel has been fully read yet.
func (ch *ChannelIn) Closed() bool {
	return ch.progress > ch.endsAt
}
