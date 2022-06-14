package derive

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/log"
)

// ChannelBank buffers channel frames, and emits TaggedData to an onProgress channel.
type ChannelBank struct {
	log log.Logger

	// channels by ID
	channels map[ChannelID]*ChannelIn
	// channels in FIFO order
	channelQueue []ChannelID

	// Current L1 origin that we have seen. Used to filter channels and continue reading.
	currentL1Origin eth.L1BlockRef
}

// Read the raw data of the first channel, if it's timed-out or closed
// Read returns a zeroed channel ID and nil data if there is nothing new to Read.
// The caller should tag the data with CurrentL1() to track the last L1 block the channel data depends on.
func (ib *ChannelBank) Read() (chID ChannelID, data []byte) {
	if len(ib.channelQueue) == 0 {
		return ChannelID{}, nil
	}
	first := ib.channelQueue[0]
	ch := ib.channels[first]
	timedOut := ch.firstSeen+ChannelTimeout < ib.currentL1Origin.Time
	if timedOut {
		ib.log.Info("channel timed out", "channel", ch, "frames", len(ch.inputs), "first_seen", ch.firstSeen)
	}
	if ch.closed {
		ib.log.Debug("channel closed", "channel", ch, "first_seen", ch.firstSeen)
	}
	if !timedOut && !ch.closed {
		return ChannelID{}, nil
	}
	delete(ib.channels, first)
	ib.channelQueue = ib.channelQueue[1:]
	return first, ch.Read()
}

func (ib *ChannelBank) CurrentL1() eth.L1BlockRef {
	return ib.currentL1Origin
}

// NextL1 updates the channel bank to tag new data with the next L1 reference
func (ib *ChannelBank) NextL1(ref eth.L1BlockRef) error {
	if ref.ParentHash != ib.currentL1Origin.Hash {
		return fmt.Errorf("reorg detected, cannot start consuming this L1 block without using a new channel bank: new.parent: %s, expected: %s", ref.ParentID(), ib.currentL1Origin.ParentID())
	}
	ib.currentL1Origin = ref
	return nil
}

// IngestData adds new L1 data to the channel bank.
// Read() should be called repeatedly first, until everything has been read, before adding new data.
// Then NextL1(ref) should be called to move forward to the next L1 input
func (ib *ChannelBank) IngestData(data []byte) error {
	if len(data) < 1 {
		return fmt.Errorf("data must be at least have a version byte")
	}

	if data[0] != DerivationVersion0 {
		return fmt.Errorf("unrecognized derivation version: %d", data)
	}

	// check total size
	totalSize := uint64(0)
	for _, ch := range ib.channels {
		totalSize += ch.size
	}
	// prune until it is reasonable again. The high-priority channel failed to be read, so we start pruning there.
	for totalSize > MaxChannelBankSize {
		id := ib.channelQueue[0]
		ch := ib.channels[id]
		ib.channelQueue = ib.channelQueue[1:]
		delete(ib.channels, id)
		totalSize -= ch.size
	}

	offset := 1
	if len(data[offset:]) < minimumFrameSize {
		return fmt.Errorf("data must be at least have one frame")
	}

	// Iterate over all frames. They may have different channel IDs to indicate that they stream consumer should reset.
	for {
		if len(data) < offset+ChannelIDSize {
			return nil
		}
		var chID ChannelID
		copy(chID[:], data[offset:])
		offset += ChannelIDSize
		// stop reading and ignore remaining data if we encounter a zeroed ID
		if chID == (ChannelID{}) {
			return nil
		}

		frameNumber, n := binary.Uvarint(data[offset:])
		if n <= 0 {
			return fmt.Errorf("failed to read frame number")
		}
		offset += n

		frameLength, n := binary.Uvarint(data[offset:])
		if n <= 0 {
			return fmt.Errorf("failed to read frame length")
		}
		offset += n

		if remaining := uint64(len(data) - offset); remaining < frameLength {
			return fmt.Errorf("not enough data left for frame: %d < %d", remaining, frameLength)
		}
		frameData := data[offset : uint64(offset)+frameLength]
		offset += int(frameLength)

		if offset >= len(data) {
			return fmt.Errorf("failed to read frame end byte, no data left, offset past length %d", len(data))
		}
		isLastNum := data[offset]
		if isLastNum > 1 {
			return fmt.Errorf("invalid isLast bool value: %d", data[offset])
		}
		isLast := isLastNum == 1
		offset += 1

		currentCh, ok := ib.channels[chID]
		if ok { // check if the channel is not timed out
			if currentCh.firstSeen+ChannelTimeout < ib.currentL1Origin.Time {
				ib.log.Info("channel is timed out, ignore frame", "channel", chID, "first_seen", currentCh.firstSeen, "frame", frameNumber)
				continue
			}
		} else { // create new channel if it doesn't exist yet
			currentCh = &ChannelIn{id: chID, firstSeen: ib.currentL1Origin.Time}
			ib.channels[chID] = currentCh
			ib.channelQueue = append(ib.channelQueue, chID)
		}

		if err := currentCh.IngestData(frameNumber, isLast, frameData); err != nil {
			ib.log.Debug("failed to ingest frame into channel", "frame_number", frameNumber, "channel", chID, "err", err)
			continue
		}
	}
}

// NewChannelBank prepares a new channel bank,
// ready to read data from starting at a point where we can be sure no channel data is missing.
// Upon a reorg, or startup, a channel bank should be constructed with the last consumed L1 block as l1Start.
// It will traverse back with the provided lookupParent to find the continuation point,
// and then replay everything with pullData to get a channel bank ready for reading from l1Start.
func NewChannelBank(ctx context.Context, log log.Logger, l1Start eth.L1BlockRef,
	lookupParent func(ctx context.Context, id eth.BlockID) (eth.L1BlockRef, error),
	pullData func(id eth.BlockID, txIndex uint64) ([]byte, error)) (*ChannelBank, error) {
	block := l1Start
	var blocks []eth.L1BlockRef
	for block.Time+ChannelTimeout > l1Start.Time && block.Number > 0 {
		parent, err := lookupParent(ctx, block.ParentID())
		if err != nil {
			return nil, fmt.Errorf("failed to find channel bank block, failed to retrieve L1 reference: %w", err)
		}
		block = parent
		blocks = append(blocks, parent)
	}
	bank := &ChannelBank{
		log:             log,
		channels:        make(map[ChannelID]*ChannelIn),
		currentL1Origin: blocks[len(blocks)-1],
	}

	// now replay all the parent blocks
	for i := len(blocks) - 1; i >= 0; i-- {
		ref := blocks[i]
		if i != len(blocks)-1 {
			if err := bank.NextL1(ref); err != nil {
				return nil, fmt.Errorf("failed to continue replay at block %s: %v", ref, err)
			}
		}
		for j := uint64(0); ; j++ {
			txData, err := pullData(ref.ID(), j)
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("failed to replay data of %s: %w", blocks[i], err)
			}
			if err := bank.IngestData(txData); err != nil {
				log.Debug("encountered bad tx during replay", "replay_index", i, "block", ref.ID(), "tx_index", j, "err", err)
				continue
			}
		}
		// we drain before ingesting more, since writes affect reads this is mandatory
		for {
			if chID, _ := bank.Read(); chID == (ChannelID{}) {
				break
			}
		}
	}
	if err := bank.NextL1(l1Start); err != nil {
		return nil, fmt.Errorf("failed to move bank origin to final %s: %v", l1Start, err)
	}
	return bank, nil
}
