package buidl

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

type OutChannel struct {
	ID ChannelID

	// exclusive
	Start eth.BlockID
	// inclusive
	End eth.BlockID

	// Frame ID of the next frame to emit. Increment after emitting
	Frame uint64

	// How much we've pulled from the reader so far
	Offset uint64

	// Nil when closed
	Reader io.Reader

	// scratch for temporary buffering
	scratch bytes.Buffer
}

func makeUVarint(x uint64) []byte {
	var tmp [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(tmp[:], x)
	return tmp[:n]
}

func (oc *OutChannel) Output(maxSize uint64) ([]byte, error) {
	if oc.Reader == nil {
		return nil, fmt.Errorf("channel is closed")
	}

	var out []byte
	out = append(out, makeUVarint(uint64(oc.ID))...)
	out = append(out, makeUVarint(oc.Frame)...)
	// +1 for single byte of frame content, +1 for lastFrame bool
	if uint64(len(out))+2 > maxSize {
		return nil, fmt.Errorf("no more space: %d > %d", len(out), maxSize)
	}

	remaining := maxSize - uint64(len(out))
	maxFrameLen := remaining - 1 // -1 for the bool at the end
	// estimate how many bytes we lose with encoding the length of the frame
	// by encoding the max length (larger uvarints take more space)
	maxFrameLen -= uint64(len(makeUVarint(maxFrameLen)))

	oc.scratch.Reset()
	_, err := io.CopyN(&oc.scratch, oc.Reader, int64(maxFrameLen))
	frameLen := uint64(len(oc.scratch.Bytes()))
	oc.Offset += frameLen
	lastFrame := err == io.EOF
	if err != nil && !lastFrame {
		return nil, fmt.Errorf("failed to read data for frame: %w", err)
	}
	out = append(out, makeUVarint(frameLen)...)
	out = append(out, oc.scratch.Bytes()...)
	if lastFrame {
		out = append(out, 1)

	} else {
		out = append(out, 0)
	}
	oc.Frame += 1
	return out, nil
}

type Outgoing struct {
	openChannels map[ChannelID]*OutChannel
}

// TODO: based on previous data we may be able to reconstruct a half-finished channel, to continue it on a fresh (restarted or different instance) rollup node.

func (og *Outgoing) Output(ctx context.Context, lastID ChannelID, lastComplete eth.BlockID, id ChannelID, maxSize uint64) ([]byte, error) {
	if og.openChannels == nil {
		og.openChannels = make(map[ChannelID]*OutChannel)
	}

	// TODO: prune timed out and closed channels

	var out []byte
	out = append(out, ChannelVersion0)

	// Open new channels while we have space left to output to
	for {
		if ctx.Err() != nil {
			return out, nil
		}

		// check if we can fit in one more frame
		if uint64(len(out))+minimumFrameSize > maxSize {
			return out, nil
		}

		outCh, ok := og.openChannels[lastID]
		if !ok {
			// TODO find L2 end block for channel. Block until it's found, or ctx ends.
			end := eth.BlockID{}
			// TODO construct reader for encoding the data
			var r io.Reader
			outCh = &OutChannel{
				ID:     id,
				Start:  lastComplete,
				End:    end,
				Frame:  0,
				Offset: 0,
				Reader: r,
			}
			og.openChannels[id] = outCh
		}

		frame, err := outCh.Output(maxSize - uint64(len(out)))
		if err != nil {
			// remove the faulty channel (it may be closed, not canonical chain anymore, or corrupted somehow)
			delete(og.openChannels, outCh.ID)
			// TODO log error
			continue
		}
		out = append(out, frame...)
	}
}
