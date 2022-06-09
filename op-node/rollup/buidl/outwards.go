package buidl

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"sort"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/log"
)

type OutChannel struct {
	id ChannelID

	// the L2 blocks that were encoded in this channel
	blocks []eth.BlockID

	// Frame ID of the next frame to emit. Increment after emitting
	frame uint64

	// How much we've pulled from the reader so far
	offset uint64

	// time of creation, to prune out old timed-out channels
	created uint64

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

func (oc *OutChannel) Closed() bool {
	return oc.reader == nil
}

func (oc *OutChannel) Output(maxSize uint64) ([]byte, error) {
	if oc.reader == nil {
		return nil, fmt.Errorf("channel is closed")
	}

	var out []byte
	out = append(out, oc.id[:]...)
	out = append(out, makeUVarint(oc.frame)...)
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
	_, err := io.CopyN(&oc.scratch, oc.reader, int64(maxFrameLen))
	frameLen := uint64(len(oc.scratch.Bytes()))
	oc.offset += frameLen
	lastFrame := err == io.EOF
	if err != nil && !lastFrame {
		return nil, fmt.Errorf("failed to read data for frame: %w", err)
	}
	out = append(out, makeUVarint(frameLen)...)
	out = append(out, oc.scratch.Bytes()...)
	if lastFrame {
		out = append(out, 1)
		oc.reader = nil // close the channel
	} else {
		out = append(out, 0)
	}
	oc.frame += 1
	return out, nil
}

type Outgoing struct {
	log log.Logger

	// pruned when timed out. We keep track of fully read channels to avoid resubmitting data.
	channels map[ChannelID]*OutChannel

	l1Head eth.L1BlockRef
}

// SetL1Head updates the L1 head, so the old channels can be pruned
func (og *Outgoing) SetL1Head(head eth.L1BlockRef) {
	og.l1Head = head
}

// TODO: based on previous data we may be able to reconstruct a partially-consumed channel, to continue it on a fresh (restarted or different instance) rollup node.

type OutputData struct {
	// Channels identifies all channels that were involved in this output, with their last frame ID.
	// Empty if no new data was produced.
	Channels map[ChannelID]uint64

	// Data to post to L1, encodes channel version byte and one or more frames
	Data []byte
}

// history is the collection of channels that have been submitted, and the frame ID of the last submission
func (og *Outgoing) Output(ctx context.Context, history map[ChannelID]uint64, maxSize uint64, maxBlocksPerFrame uint64) (*OutputData, error) {
	if og.channels == nil {
		og.channels = make(map[ChannelID]*OutChannel)
	}

	// prune timed out channels, before adding new ones
	for id, ch := range og.channels {
		if ch.created+ChannelTimeout < og.l1Head.Time {
			if ch.Closed() {
				og.log.Debug("cleaning up closed timed-out channel", "channel", ch)
			} else {
				og.log.Warn("channel timed out without completing", "channel", ch, "frame", ch.frame, "created", ch.created, "offset", ch.offset)
			}
			delete(og.channels, id)
		}
	}

	// We find the first 1000 unsafe blocks that we may want to put in the output
	unsafeBlocks := make(map[eth.BlockID]struct{})
	// TODO scan from safe-head to unsafe-head and fill unsafeBlocks

	out := &OutputData{Channels: make(map[ChannelID]uint64)}
	out.Data = append(out.Data, ChannelVersion0)

	// check full history, and add data for any channels we still consider to be open.
	for chID, frameNr := range history {
		if ctx.Err() != nil { // return what we have if we run out of time.
			return out, nil
		}

		// check if we can fit in one more frame
		if uint64(len(out.Data))+minimumFrameSize > maxSize {
			return out, nil
		}

		outCh, ok := og.channels[chID]
		if !ok {
			// we may already have pruned the timed-out channel.
			// If timed-out, fair game to resubmit contents if we still consider the contents unsafe.
			continue
		}
		if len(outCh.blocks) == 0 {
			delete(og.channels, chID)
			og.log.Warn("found open channel without any blocks, deleting it", "channel", chID)
			continue
		}

		nextFrame := frameNr + 1
		// The caller is behind the previous state of this channel, e.g. due to a reorg of L1.
		// There may be signed txs floating around that add frames to this channel.
		// Let's avoid this channel, and don't encode any of the blocks.
		// When the channel times out we can reinsert the blocks.
		if outCh.frame > nextFrame {
			for _, b := range outCh.blocks {
				delete(unsafeBlocks, b)
			}
			og.log.Warn("Cannot continue channel from older frame, thus not submitting blocks of this channel",
				"channel", chID, "expected_next_frame", outCh.frame, "history_next_frame", nextFrame)
			continue // TODO: if we want to reproduce new (potentially conflicting) frames we can, but we may not want to.
		}
		// The channel is not as far, we don't know of the frame that was previously submitted
		if outCh.frame < nextFrame {
			for _, b := range outCh.blocks {
				delete(unsafeBlocks, b)
			}
			og.log.Warn("Cannot continue channel from future frame, thus not submitting blocks of this channel",
				"channel", chID, "expected_next_frame", outCh.frame, "history_next_frame", nextFrame)
			continue
		}

		// If the channel is done, we avoid resubmitting any of the blocks
		if outCh.Closed() {
			for _, b := range outCh.blocks {
				delete(unsafeBlocks, b)
			}
			og.log.Debug("Already submitted full channel contents of channel, not submitting it again",
				"channel", chID)
			continue
		}

		frame, err := outCh.Output(maxSize - uint64(len(out.Data)))
		if err != nil {
			// remove the channel (it may be closed, not canonical chain anymore, or corrupted somehow)
			delete(og.channels, outCh.id)
			log.Error("failed to output frame for channel", "channel", outCh.id, "err", err)
			continue
		}
		out.Data = append(out.Data, frame...)
		out.Channels[outCh.id] = nextFrame
	}

	// There may be gaps in the remaining unsafe blocks to submit.
	// But we want to submit the lowest-number blocks first. So collect and sort them.
	unsafeBlocksSorted := make([]eth.BlockID, 0, len(unsafeBlocks))
	for b := range unsafeBlocks {
		unsafeBlocksSorted = append(unsafeBlocksSorted, b)
	}
	sort.Slice(unsafeBlocksSorted, func(i, j int) bool {
		return unsafeBlocksSorted[i].Number < unsafeBlocksSorted[j].Number
	})

	// Open new channels while we have space left to output to.
	for {
		if ctx.Err() != nil { // return what we have if we run out of time.
			return out, nil
		}

		// check if we can fit in one more frame
		if uint64(len(out.Data))+minimumFrameSize > maxSize {
			return out, nil
		}

		var id ChannelID
		if _, err := rand.Read(id[:]); err != nil {
			return nil, fmt.Errorf("failed to create new random ID: %v", err)
		}

		if _, ok := og.channels[id]; ok {
			log.Warn("generated a channel ID that already exists", "channel", id)
			continue
		}

		blocks := unsafeBlocksSorted
		// don't put too many L2 blocks into the same frame.
		if uint64(len(blocks)) > maxBlocksPerFrame {
			blocks = blocks[:maxBlocksPerFrame]
		}
		// TODO construct reader for encoding the data
		var r io.Reader
		outCh := &OutChannel{
			id:      id,
			blocks:  blocks,
			frame:   0,
			offset:  0,
			created: og.l1Head.Time,
			reader:  r,
		}
		og.channels[id] = outCh

		frame, err := outCh.Output(maxSize - uint64(len(out.Data)))
		if err != nil {
			log.Error("failed to output frame for new channel", "channel", id, "err", err)
			continue
		}
		out.Data = append(out.Data, frame...)
		out.Channels[id] = 0
	}
}
