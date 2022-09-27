package derive

import (
	"context"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// ChannelBank is a stateful stage that does the following:
// 1. Unmarshalls frames from L1 transaction data
// 2. Applies those frames to a channel
// 3. Attempts to read from the channel when it is ready
// 4. Prunes channels (not frames) when the channel bank is too large.
//
// Note: we prune before we ingest data.
// As we switch between ingesting data & reading, the prune step occurs at an odd point
// Specifically, the channel bank is not allowed to become too large between successive calls
// to `IngestData`. This means that we can do an ingest and then do a read while becoming too large.

type ChannelBankOutput interface {
	StageProgress
	WriteChannel(data []byte)
}

// ChannelBank buffers channel frames, and emits full channel data
type ChannelBank struct {
	log log.Logger
	cfg *rollup.Config

	channels     map[ChannelID]*Channel // channels by ID
	channelQueue []ChannelID            // channels in FIFO order

	progress Progress

	next ChannelBankOutput
	prev *L1Retrieval
}

var _ Stage = (*ChannelBank)(nil)

// NewChannelBank creates a ChannelBank, which should be Reset(origin) before use.
func NewChannelBank(log log.Logger, cfg *rollup.Config, next ChannelBankOutput, prev *L1Retrieval) *ChannelBank {
	return &ChannelBank{
		log:          log,
		cfg:          cfg,
		channels:     make(map[ChannelID]*Channel),
		channelQueue: make([]ChannelID, 0, 10),
		next:         next,
		prev:         prev,
	}
}

func (ib *ChannelBank) Progress() Progress {
	return ib.progress
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
// Read() should be called repeatedly first, until everything has been read, before adding new data.\
func (ib *ChannelBank) IngestData(data []byte) {
	if ib.progress.Closed {
		panic("write data to bank while closed")
	}
	ib.log.Debug("channel bank got new data", "origin", ib.progress.Origin, "data_len", len(data))

	// TODO: Why is the prune here?
	ib.prune()

	frames, err := ParseFrames(data)
	if err != nil {
		ib.log.Warn("malformed frame", "err", err)
		return
	}

	// Process each frame
	for _, f := range frames {
		currentCh, ok := ib.channels[f.ID]
		if !ok {
			// create new channel if it doesn't exist yet
			currentCh = NewChannel(f.ID, ib.progress.Origin)
			ib.channels[f.ID] = currentCh
			ib.channelQueue = append(ib.channelQueue, f.ID)
		}

		// check if the channel is not timed out
		if currentCh.OpenBlockNumber()+ib.cfg.ChannelTimeout < ib.progress.Origin.Number {
			ib.log.Warn("channel is timed out, ignore frame", "channel", f.ID, "frame", f.FrameNumber)
			continue
		}

		ib.log.Trace("ingesting frame", "channel", f.ID, "frame_number", f.FrameNumber, "length", len(f.Data))
		if err := currentCh.AddFrame(f, ib.progress.Origin); err != nil {
			ib.log.Warn("failed to ingest frame into channel", "channel", f.ID, "frame_number", f.FrameNumber, "err", err)
			continue
		}
	}
}

// Read the raw data of the first channel, if it's timed-out or closed.
// Read returns io.EOF if there is nothing new to read.
func (ib *ChannelBank) Read() (data []byte, err error) {
	if len(ib.channelQueue) == 0 {
		return nil, io.EOF
	}
	first := ib.channelQueue[0]
	ch := ib.channels[first]
	timedOut := ch.OpenBlockNumber()+ib.cfg.ChannelTimeout < ib.progress.Origin.Number
	if timedOut {
		ib.log.Debug("channel timed out", "channel", first, "frames", len(ch.inputs))
		delete(ib.channels, first)
		ib.channelQueue = ib.channelQueue[1:]
		return nil, io.EOF
	}
	if !ch.IsReady() {
		return nil, io.EOF
	}

	delete(ib.channels, first)
	ib.channelQueue = ib.channelQueue[1:]
	r := ch.Reader()
	// Suprress error here. io.ReadAll does return nil instead of io.EOF though.
	data, _ = io.ReadAll(r)
	return data, nil
}

// Step does the advancement for the channel bank.
// Channel bank as the first non-pull stage does it's own progress maintentance.
// When closed, it checks against the previous origin to determine if to open itself
func (ib *ChannelBank) Step(ctx context.Context, _ Progress) error {
	// Open ourselves
	// This is ok to do b/c we would not have yielded control to the lower stages
	// of the pipeline without being completely done reading from L1.
	if ib.progress.Closed {
		if ib.progress.Origin != ib.prev.Origin() {
			ib.progress.Closed = false
			ib.progress.Origin = ib.prev.Origin()
			return nil
		}
	}

	skipIngest := ib.next.Progress().Origin.Number > ib.progress.Origin.Number
	outOfData := false

	if data, err := ib.prev.NextData(ctx); err == io.EOF {
		outOfData = true
	} else if err != nil {
		return err
	} else {
		ib.IngestData(data)
	}

	// otherwise, read the next channel data from the bank
	data, err := ib.Read()
	if err == io.EOF { // need new L1 data in the bank before we can read more channel data
		if outOfData {
			if !ib.progress.Closed {
				ib.progress.Closed = true
				return nil
			}
			return io.EOF
		} else {
			return nil
		}
	} else if err != nil {
		return err
	} else {
		if !skipIngest {
			ib.next.WriteChannel(data)
			return nil
		}
	}
	return nil
}

// ResetStep walks back the L1 chain, starting at the origin of the next stage,
// to find the origin that the channel bank should be reset to,
// to get consistent reads starting at origin.
// Any channel data before this origin will be timed out by the time the channel bank is synced up to the origin,
// so it is not relevant to replay it into the bank.
func (ib *ChannelBank) ResetStep(ctx context.Context, l1Fetcher L1Fetcher) error {
	ib.progress = ib.next.Progress()
	ib.log.Debug("walking back to find reset origin for channel bank", "origin", ib.progress.Origin)
	// go back in history if we are not distant enough from the next stage
	resetBlock := ib.progress.Origin.Number - ib.cfg.ChannelTimeout
	if ib.progress.Origin.Number < ib.cfg.ChannelTimeout {
		resetBlock = 0 // don't underflow
	}
	parent, err := l1Fetcher.L1BlockRefByNumber(ctx, resetBlock)
	if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to find channel bank block, failed to retrieve L1 reference: %w", err))
	}
	ib.progress.Origin = parent
	return io.EOF
}

type L1BlockRefByHashFetcher interface {
	L1BlockRefByHash(context.Context, common.Hash) (eth.L1BlockRef, error)
}
