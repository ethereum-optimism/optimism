package derive

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type ChannelBankOutput interface {
	OriginStage
	WriteChannel(data []byte)
}

// ChannelBank buffers channel frames, and emits full channel data
type ChannelBank struct {
	log log.Logger
	cfg *rollup.Config

	channels     map[ChannelID]*ChannelIn // channels by ID
	channelQueue []ChannelID              // channels in FIFO order

	// Current L1 origin that we have seen. Used to filter channels and continue reading.
	currentOrigin eth.L1BlockRef
	originOpen    bool

	resetting bool

	next ChannelBankOutput
}

var _ OriginStage = (*ChannelBank)(nil)

// NewChannelBank creates a ChannelBank, which should be Reset(origin) before use.
func NewChannelBank(log log.Logger, cfg *rollup.Config, next ChannelBankOutput) *ChannelBank {
	return &ChannelBank{
		log:          log,
		cfg:          cfg,
		channels:     make(map[ChannelID]*ChannelIn),
		channelQueue: make([]ChannelID, 0, 10),
		next:         next,
	}
}

func (ib *ChannelBank) CurrentOrigin() eth.L1BlockRef {
	return ib.currentOrigin
}

// OpenOrigin updates the channel bank to tag new data with the next L1 reference
func (ib *ChannelBank) OpenOrigin(ref eth.L1BlockRef) error {
	if ref.ParentHash != ib.currentOrigin.Hash {
		return fmt.Errorf("reorg detected, cannot start consuming this L1 block without using a new channel bank: new.parent: %s, expected: %s", ref.ParentID(), ib.currentOrigin.ParentID())
	}
	ib.currentOrigin = ref
	ib.originOpen = true
	return nil
}

func (ib *ChannelBank) CloseOrigin() {
	ib.originOpen = false
}

func (ib *ChannelBank) IsOriginOpen() bool {
	return ib.originOpen
}

func (ib *ChannelBank) prune() {
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
}

// IngestData adds new L1 data to the channel bank.
// Read() should be called repeatedly first, until everything has been read, before adding new data.
// Then NextL1(ref) should be called to move forward to the next L1 input
func (ib *ChannelBank) IngestData(data []byte) error {
	if len(data) < 1 {
		ib.log.Error("data must be at least have a version byte, but got empty string")
		return nil
	}

	if data[0] != DerivationVersion0 {
		return fmt.Errorf("unrecognized derivation version: %d", data)
	}

	ib.prune()

	offset := 1
	if len(data[offset:]) < minimumFrameSize {
		return fmt.Errorf("data must be at least have one frame")
	}

	// Iterate over all frames. They may have different channel IDs to indicate that they stream consumer should reset.
	for {
		if len(data) < offset+ChannelIDDataSize+1 {
			return nil
		}
		var chID ChannelID
		copy(chID.Data[:], data[offset:])
		offset += ChannelIDDataSize
		chIDTime, n := binary.Uvarint(data[offset:])
		if n <= 0 {
			return fmt.Errorf("failed to read frame number")
		}
		offset += n
		chID.Time = chIDTime

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

		// check if the channel is not timed out
		if chID.Time+ChannelTimeout < ib.currentOrigin.Time {
			ib.log.Info("channel is timed out, ignore frame", "channel", chID, "id_time", chID.Time, "frame", frameNumber)
			continue
		}
		// check if the channel is not included too soon (otherwise timeouts wouldn't be effective)
		if chID.Time > ib.currentOrigin.Time {
			ib.log.Info("channel claims to be from the future, ignore frame", "channel", chID, "id_time", chID.Time, "frame", frameNumber)
			continue
		}

		currentCh, ok := ib.channels[chID]
		if !ok { // create new channel if it doesn't exist yet
			currentCh = &ChannelIn{id: chID}
			ib.channels[chID] = currentCh
			ib.channelQueue = append(ib.channelQueue, chID)
		}

		ib.log.Debug("ingesting frame", "channel", chID, "frame_number", frameNumber, "length", len(frameData))
		if err := currentCh.IngestData(frameNumber, isLast, frameData); err != nil {
			ib.log.Debug("failed to ingest frame into channel", "channel", chID, "frame_number", frameNumber, "err", err)
			continue
		}
	}
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
	timedOut := first.Time+ChannelTimeout < ib.currentOrigin.Time
	if timedOut {
		ib.log.Info("channel timed out", "channel", first, "frames", len(ch.inputs))
	}
	if ch.closed {
		ib.log.Debug("channel closed", "channel", first)
	}
	if !timedOut && !ch.closed {
		return ChannelID{}, nil
	}
	delete(ib.channels, first)
	ib.channelQueue = ib.channelQueue[1:]
	return first, ch.Read()
}

func (ib *ChannelBank) Reset(origin eth.L1BlockRef) {
	ib.currentOrigin = origin
	ib.channels = make(map[ChannelID]*ChannelIn)
	ib.channelQueue = ib.channelQueue[:0]
	ib.originOpen = true
}

func (ib *ChannelBank) Step(ctx context.Context) error {
	// If the bank is behind the channel reader, then we are replaying old data to prepare the bank.
	// Read if we can, and drop if it gives anything
	if ib.next.CurrentOrigin().Number > ib.CurrentOrigin().Number {
		id, _ := ib.Read()
		if id == (ChannelID{}) {
			return io.EOF
		}
		return nil
	}

	// move forward the ch reader if the bank has new L1 data
	if ib.next.CurrentOrigin() != ib.CurrentOrigin() {
		return ib.next.OpenOrigin(ib.CurrentOrigin())
	}
	// otherwise, read the next channel data from the bank
	id, data := ib.Read()
	if id == (ChannelID{}) { // need new L1 data in the bank before we can read more channel data
		ib.next.CloseOrigin()
		return io.EOF
	}
	ib.log.Info("writing channel", "channel", id)
	ib.next.WriteChannel(data)
	return nil
}

// ResetStep walks back the L1 chain, starting at the origin of the next stage,
// to find the origin that the channel bank should be reset to,
// to get consistent reads starting at origin.
// Any channel data before this origin will be timed out by the time the channel bank is synced up to the origin,
// so it is not relevant to replay it into the bank.
func (ib *ChannelBank) ResetStep(ctx context.Context, l1Fetcher L1Fetcher) error {
	if !ib.resetting {
		ib.currentOrigin = ib.next.CurrentOrigin()
		ib.originOpen = false
		ib.resetting = true
	}
	if ib.currentOrigin.Time+ChannelTimeout < ib.next.CurrentOrigin().Time || ib.currentOrigin.Number == 0 {
		ib.resetting = false
		return io.EOF
	}
	// go back in history if we are not distant enough from the next stage
	parent, err := l1Fetcher.L1BlockRefByHash(ctx, ib.currentOrigin.ParentHash)
	if err != nil {
		ib.log.Error("failed to find channel bank block, failed to retrieve L1 reference", "err", err)
	}
	ib.currentOrigin = parent
	return nil
}

type L1BlockRefByHashFetcher interface {
	L1BlockRefByHash(context.Context, common.Hash) (eth.L1BlockRef, error)
}
