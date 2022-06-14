package derive

import (
	"fmt"
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

	inputs map[uint64][]byte
}

// IngestData buffers a frame in the channel
func (ch *ChannelIn) IngestData(frameNum uint64, isLast bool, frameData []byte) error {
	if frameNum < ch.progress {
		// already consumed a frame with equal number, this must be a duplicate
		return DuplicateErr
	}
	if frameNum > ch.endsAt {
		return fmt.Errorf("channel already ended ingesting inputs")
	}
	if ch.endsAt != ^uint64(0) && isLast {
		return fmt.Errorf("already received a closing frame")
	}
	// the frame is from the current or future, it will be read from the buffer later

	// create buffer if it didn't exist yet
	if ch.inputs == nil {
		ch.inputs = make(map[uint64][]byte)
	}
	if _, exists := ch.inputs[frameNum]; exists {
		// already seen a frame for this channel with this frame number
		return DuplicateErr
	}
	// buffer the frame
	ch.inputs[frameNum] = frameData
	if isLast {
		ch.endsAt = frameNum
	}
	ch.size += uint64(len(frameData)) + frameOverhead
	return nil
}

// Read next tagged piece of data. This may return nil if there is none.
func (ch *ChannelIn) Read() []byte {
	data, ok := ch.inputs[ch.progress]
	if !ok {
		return nil
	}
	ch.size -= uint64(len(data)) + frameOverhead
	delete(ch.inputs, ch.progress)
	ch.progress += 1
	return data
}

// Closed returns if this channel has been fully read yet.
func (ch *ChannelIn) Closed() bool {
	return ch.progress > ch.endsAt
}
