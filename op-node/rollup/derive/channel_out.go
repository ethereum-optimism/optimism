package derive

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

type ChannelOut struct {
	id ChannelID

	// the L2 blocks that were encoded in this channel
	blocks []eth.BlockID

	// Frame ID of the next frame to emit. Increment after emitting
	frame uint64

	// How much we've pulled from the reader so far
	offset uint64

	// Nil when closed
	reader io.Reader

	// scratch for temporary buffering
	scratch bytes.Buffer
}

func makeUVarint(x uint64) []byte {
	var tmp [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(tmp[:], x)
	return tmp[:n]
}

func (co *ChannelOut) Closed() bool {
	return co.reader == nil
}

func (co *ChannelOut) Output(maxSize uint64) ([]byte, error) {
	if co.reader == nil {
		return nil, fmt.Errorf("channel is closed")
	}

	var out []byte
	out = append(out, co.id.Data[:]...)
	out = append(out, makeUVarint(co.id.Time)...)
	out = append(out, makeUVarint(co.frame)...)
	// +1 for single byte of frame content, +1 for lastFrame bool
	if uint64(len(out))+2 > maxSize {
		return nil, fmt.Errorf("no more space: %d > %d", len(out), maxSize)
	}

	remaining := maxSize - uint64(len(out))
	maxFrameLen := remaining - 1 // -1 for the bool at the end
	// estimate how many bytes we lose with encoding the length of the frame
	// by encoding the max length (larger uvarints take more space)
	maxFrameLen -= uint64(len(makeUVarint(maxFrameLen)))

	co.scratch.Reset()
	_, err := io.CopyN(&co.scratch, co.reader, int64(maxFrameLen))
	frameLen := uint64(len(co.scratch.Bytes()))
	co.offset += frameLen
	lastFrame := err == io.EOF
	if err != nil && !lastFrame {
		return nil, fmt.Errorf("failed to read data for frame: %w", err)
	}
	out = append(out, makeUVarint(frameLen)...)
	out = append(out, co.scratch.Bytes()...)
	if lastFrame {
		out = append(out, 1)
		co.reader = nil // close the channel
	} else {
		out = append(out, 0)
	}
	co.frame += 1
	return out, nil
}
