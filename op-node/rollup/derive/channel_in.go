package derive

import (
	"fmt"
)

type ChannelIn struct {
	// id of the channel
	id ChannelID

	// estimated memory size, used to drop the channel if we have too much data
	size uint64

	// true if we have buffered the last frame
	closed bool

	inputs map[uint64][]byte
}

// IngestData buffers a frame in the channel
func (ch *ChannelIn) IngestData(frameNum uint64, isLast bool, frameData []byte) error {
	if ch.closed {
		return fmt.Errorf("already received a closing frame")
	}
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
	ch.closed = isLast
	ch.size += uint64(len(frameData)) + frameOverhead
	return nil
}

// Read full channel content (it may be incomplete if the channel is not Closed)
func (ch *ChannelIn) Read() (out []byte) {
	for frameNr := uint64(0); ; frameNr++ {
		data, ok := ch.inputs[frameNr]
		if !ok {
			return
		}
		out = append(out, data...)
	}
}
