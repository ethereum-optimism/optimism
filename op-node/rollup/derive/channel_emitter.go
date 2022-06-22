package derive

import (
	"context"
	"crypto/rand"
	"fmt"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-node/rollup"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/log"
)

type BlocksSource interface {
	PayloadByHash(context.Context, common.Hash) (*eth.ExecutionPayload, error)
}

type UnsafeBlocksSource interface {
	BlocksSource
	UnsafeBlockIDs(ctx context.Context, safeHead eth.BlockID, max uint64) ([]eth.BlockID, error)
}

type OutputMetaData struct {
	// Current safe-point of the L2 chain, derived from L1 data.
	SafeHead eth.L2BlockRef `json:"safe_head"`

	// Current tip of the L2 chain
	UnsafeHead eth.L2BlockRef `json:"unsafe_head"`

	// Number of blocks that have been included in a channel, but not finished yet.
	// Within the channel timeout.
	// This may include non-canonical L2 blocks, if L1 reorged the L2 chain.
	OpenedBlocks uint64 `json:"opened_blocks"`

	// Number of blocks that have been fully submitted (i.e. closed channel).
	// Within the channel timeout.
	// This may include non-canonical L2 blocks, if L1 reorged the L2 chain.
	ClosedBlocks uint64 `json:"closed_blocks"`
}

type BatcherChannelData struct {
	// Channels identifies all channels that were involved in this output, with their last frame ID.
	// Empty if no new data was produced.
	Channels map[ChannelID]uint64 `json:"channels"`

	// Data to post to L1, encodes channel version byte and one or more frames.
	Data []byte `json:"data"`

	Meta OutputMetaData `json:"meta"`
}

// ChannelEmitter maintains open channels and emits data with channel frames to confirm the L2 unsafe blocks.
type ChannelEmitter struct {
	mu sync.Mutex

	log log.Logger

	cfg *rollup.Config

	source UnsafeBlocksSource

	// pruned when timed out. We keep track of fully read channels to avoid resubmitting data.
	channels map[ChannelID]*ChannelOut

	// context used to fetch data for channels, might outlive a single request
	ctx context.Context
	// cancels above context
	cancel context.CancelFunc

	l1Time       uint64
	l2SafeHead   eth.L2BlockRef
	l2UnsafeHead eth.L2BlockRef
}

func NewChannelEmitter(log log.Logger, cfg *rollup.Config, source UnsafeBlocksSource) *ChannelEmitter {
	ctx, cancel := context.WithCancel(context.Background())
	return &ChannelEmitter{
		log:      log,
		cfg:      cfg,
		source:   source,
		channels: make(map[ChannelID]*ChannelOut),
		ctx:      ctx,
		cancel:   cancel,
		l1Time:   0,
	}
}

func (og *ChannelEmitter) Close() error {
	og.cancel()
	return nil
}

// SetL1Time updates the tracked L1 time, so the old channels can be pruned
func (og *ChannelEmitter) SetL1Time(l1Time uint64) {
	og.l1Time = l1Time
}

// SetL2SafeHead updates the tracked L2 safe head,
// so we can submit data for all unsafe blocks after it, and provide it as metadata in responses.
func (og *ChannelEmitter) SetL2SafeHead(l2SafeHead eth.L2BlockRef) {
	og.l2SafeHead = l2SafeHead
}

// SetL2UnsafeHead updates the tracked L2 unsafe head, to provide as metadata in responses
func (og *ChannelEmitter) SetL2UnsafeHead(l2UnsafeHead eth.L2BlockRef) {
	og.l2UnsafeHead = l2UnsafeHead
}

// TODO: based on previous data we may be able to reconstruct a partially-consumed channel, to continue it on a fresh (restarted or different instance) rollup node.

// history is the collection of channels that have been submitted, and the frame ID of the last submission
func (og *ChannelEmitter) Output(ctx context.Context, history map[ChannelID]uint64, minSize uint64, maxSize uint64, maxBlocksPerChannel uint64) (*BatcherChannelData, error) {
	og.mu.Lock()
	defer og.mu.Unlock()

	// TODO: handle minSize argument

	if og.channels == nil {
		og.channels = make(map[ChannelID]*ChannelOut)
	}

	// prune timed out channels, before adding new ones
	for id, ch := range og.channels {
		if id.Time+ChannelTimeout < og.l1Time {
			if ch.Closed() {
				og.log.Debug("cleaning up closed timed-out channel", "channel", id)
			} else {
				og.log.Warn("channel timed out without completing", "channel", id, "frame", ch.frame, "offset", ch.offset)
			}
			delete(og.channels, id)
		}
	}

	// We find the first 1000 unsafe blocks that we may want to put in the output
	unsafeBlocks := make(map[eth.BlockID]struct{})
	if blocks, err := og.source.UnsafeBlockIDs(ctx, og.l2SafeHead.ID(), 1000); err != nil {
		return nil, fmt.Errorf("failed to get list of unsafe blocks to submit: %v", err)
	} else {
		for _, b := range blocks {
			unsafeBlocks[b] = struct{}{}
		}
	}

	out := &BatcherChannelData{
		Channels: make(map[ChannelID]uint64),
		Data:     make([]byte, 0, 1000),
		Meta: OutputMetaData{
			SafeHead:     og.l2SafeHead,
			UnsafeHead:   og.l2UnsafeHead,
			OpenedBlocks: 0,
			ClosedBlocks: 0,
		},
	}

	for _, ch := range og.channels {
		if ch.Closed() {
			out.Meta.ClosedBlocks += uint64(len(ch.blocks))
		} else {
			out.Meta.OpenedBlocks += uint64(len(ch.blocks))
		}
	}

	if og.l1Time == 0 { // ig l1 time is not initialized, don't create new channels with it
		return out, nil
	}

	out.Data = append(out.Data, DerivationVersion0)

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
			og.log.Error("failed to output frame for channel", "channel", outCh.id, "err", err)
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

		// submitted everything, yay!
		if len(unsafeBlocksSorted) == 0 {
			return out, nil
		}

		// check if we can fit in one more frame
		if uint64(len(out.Data))+minimumFrameSize > maxSize {
			return out, nil
		}

		var id ChannelID
		if _, err := rand.Read(id.Data[:]); err != nil {
			return nil, fmt.Errorf("failed to create new random ID: %v", err)
		}
		id.Time = og.l1Time

		if _, ok := og.channels[id]; ok {
			og.log.Warn("generated a channel ID that already exists", "channel", id)
			continue
		}

		blocks := unsafeBlocksSorted
		// don't put too many L2 blocks into the same frame.
		if uint64(len(blocks)) > maxBlocksPerChannel {
			blocks = blocks[:maxBlocksPerChannel]
		}
		for _, b := range blocks {
			og.log.Debug("opening channel with block", "block", b, "id", id)
		}
		// and don't repeat them
		unsafeBlocksSorted = unsafeBlocksSorted[len(blocks):]
		// use the emitter ctx, not the request ctx, since this reader may live longer than the request
		r, err := newChannelOutReader(og.ctx, &og.cfg.Genesis, og.source, blocks)
		if err != nil {
			// no log&continue, something is wrong, abort.
			return nil, fmt.Errorf("failed to create channel reader for blocks: %v", err)
		}
		outCh := &ChannelOut{
			id:     id,
			blocks: blocks,
			frame:  0,
			offset: 0,
			reader: r,
		}
		og.channels[id] = outCh

		frame, err := outCh.Output(maxSize - uint64(len(out.Data)))
		if err != nil {
			og.log.Error("failed to output frame for new channel", "channel", id, "err", err)
			continue
		}
		og.log.Info("frame", "data", hexutil.Bytes(outCh.scratch.Bytes()), "blocks", len(blocks))
		out.Data = append(out.Data, frame...)
		out.Channels[id] = 0
	}
}
