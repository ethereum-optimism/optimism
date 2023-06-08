package batcher

import (
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var ErrReorg = errors.New("block does not extend existing chain")

// channelManager stores a contiguous set of blocks & turns them into channels.
// Upon receiving tx confirmation (or a tx failure), it does channel error handling.
//
// For simplicity, it only creates a single pending channel at a time & waits for
// the channel to either successfully be submitted or timeout before creating a new
// channel.
// Functions on channelManager are not safe for concurrent access.
type channelManager struct {
	log  log.Logger
	metr metrics.Metricer
	cfg  ChannelConfig

	// All blocks since the last request for new tx data.
	blocks []*types.Block
	// last block hash - for reorg detection
	tip common.Hash

	// channel to write new block data to
	currentChannel *channel
	// channels to read frame data from, for writing batches onchain
	channelQueue []*channel
	// used to lookup channels by tx ID upon tx success / failure
	txChannels map[txID]*channel

	// if set to true, prevents production of any new channel frames
	closed bool
}

func NewChannelManager(log log.Logger, metr metrics.Metricer, cfg ChannelConfig) *channelManager {
	return &channelManager{
		log:        log,
		metr:       metr,
		cfg:        cfg,
		txChannels: make(map[txID]*channel),
	}
}

// Clear clears the entire state of the channel manager.
// It is intended to be used after an L2 reorg.
func (s *channelManager) Clear() {
	s.log.Trace("clearing channel manager state")
	s.blocks = s.blocks[:0]
	s.tip = common.Hash{}
	s.closed = false
	s.currentChannel = nil
	s.channelQueue = nil
	s.txChannels = make(map[txID]*channel)
}

// TxFailed records a transaction as failed. It will attempt to resubmit the data
// in the failed transaction.
func (s *channelManager) TxFailed(id txID) {
	if channel, ok := s.txChannels[id]; ok {
		delete(s.txChannels, id)
		channel.TxFailed(id)
		if s.closed && channel.NoneSubmitted() {
			s.log.Info("Channel has no submitted transactions, clearing for shutdown", "chID", channel.ID())
			s.removePendingChannel(channel)
		}
	} else {
		s.log.Warn("transaction from unknown channel marked as failed", "id", id)
	}
}

// TxConfirmed marks a transaction as confirmed on L1. Unfortunately even if all frames in
// a channel have been marked as confirmed on L1 the channel may be invalid & need to be
// resubmitted.
// This function may reset the pending channel if the pending channel has timed out.
func (s *channelManager) TxConfirmed(id txID, inclusionBlock eth.BlockID) {
	if channel, ok := s.txChannels[id]; ok {
		delete(s.txChannels, id)
		done, blocks := channel.TxConfirmed(id, inclusionBlock)
		s.blocks = append(blocks, s.blocks...)
		if done {
			s.removePendingChannel(channel)
		}
	} else {
		s.log.Warn("transaction from unknown channel marked as confirmed", "id", id)
	}
	s.metr.RecordBatchTxSubmitted()
	s.log.Debug("marked transaction as confirmed", "id", id, "block", inclusionBlock)
}

// removePendingChannel removes the given completed channel from the manager's state.
func (s *channelManager) removePendingChannel(channel *channel) {
	if s.currentChannel == channel {
		s.currentChannel = nil
	}
	index := -1
	for i, c := range s.channelQueue {
		if c == channel {
			index = i
			break
		}
	}
	if index < 0 {
		s.log.Warn("channel not found in channel queue", "id", channel.ID())
		return
	}
	s.channelQueue = append(s.channelQueue[:index], s.channelQueue[index+1:]...)
}

// nextTxData pops off s.datas & handles updating the internal state
func (s *channelManager) nextTxData(channel *channel) (txData, error) {
	if channel == nil || !channel.HasFrame() {
		s.log.Trace("no next tx data")
		return txData{}, io.EOF // TODO: not enough data error instead
	}
	tx := channel.NextTxData()
	s.txChannels[tx.ID()] = channel
	return tx, nil
}

// TxData returns the next tx data that should be submitted to L1.
//
// It currently only uses one frame per transaction. If the pending channel is
// full, it only returns the remaining frames of this channel until it got
// successfully fully sent to L1. It returns io.EOF if there's no pending frame.
func (s *channelManager) TxData(l1Head eth.BlockID) (txData, error) {
	var firstWithFrame *channel
	for _, ch := range s.channelQueue {
		if ch.HasFrame() {
			firstWithFrame = ch
			break
		}
	}

	dataPending := firstWithFrame != nil && firstWithFrame.HasFrame()
	s.log.Debug("Requested tx data", "l1Head", l1Head, "data_pending", dataPending, "blocks_pending", len(s.blocks))

	// Short circuit if there is a pending frame or the channel manager is closed.
	if dataPending || s.closed {
		return s.nextTxData(firstWithFrame)
	}

	// No pending frame, so we have to add new blocks to the channel

	// If we have no saved blocks, we will not be able to create valid frames
	if len(s.blocks) == 0 {
		return txData{}, io.EOF
	}

	if err := s.ensureChannelWithSpace(l1Head); err != nil {
		return txData{}, err
	}

	if err := s.processBlocks(); err != nil {
		return txData{}, err
	}

	// Register current L1 head only after all pending blocks have been
	// processed. Even if a timeout will be triggered now, it is better to have
	// all pending blocks be included in this channel for submission.
	s.registerL1Block(l1Head)

	if err := s.outputFrames(); err != nil {
		return txData{}, err
	}

	return s.nextTxData(s.currentChannel)
}

// ensureChannelWithSpace ensures currentChannel is populated with a channel that has
// space for more data (i.e. channel.IsFull returns false). If currentChannel is nil
// or full, a new channel is created.
func (s *channelManager) ensureChannelWithSpace(l1Head eth.BlockID) error {
	if s.currentChannel != nil && !s.currentChannel.IsFull() {
		return nil
	}

	pc, err := newChannel(s.log, s.metr, s.cfg)
	if err != nil {
		return fmt.Errorf("creating new channel: %w", err)
	}
	s.currentChannel = pc
	s.channelQueue = append(s.channelQueue, pc)
	s.log.Info("Created channel",
		"id", pc.ID(),
		"l1Head", l1Head,
		"blocks_pending", len(s.blocks))
	s.metr.RecordChannelOpened(pc.ID(), len(s.blocks))

	return nil
}

// registerL1Block registers the given block at the pending channel.
func (s *channelManager) registerL1Block(l1Head eth.BlockID) {
	s.currentChannel.RegisterL1Block(l1Head.Number)
	s.log.Debug("new L1-block registered at channel builder",
		"l1Head", l1Head,
		"channel_full", s.currentChannel.IsFull(),
		"full_reason", s.currentChannel.FullErr(),
	)
}

// processBlocks adds blocks from the blocks queue to the pending channel until
// either the queue got exhausted or the channel is full.
func (s *channelManager) processBlocks() error {
	var (
		blocksAdded int
		_chFullErr  *ChannelFullError // throw away, just for type checking
		latestL2ref eth.L2BlockRef
	)
	for i, block := range s.blocks {
		l1info, err := s.currentChannel.AddBlock(block)
		if errors.As(err, &_chFullErr) {
			// current block didn't get added because channel is already full
			break
		} else if err != nil {
			return fmt.Errorf("adding block[%d] to channel builder: %w", i, err)
		}
		blocksAdded += 1
		latestL2ref = l2BlockRefFromBlockAndL1Info(block, l1info)
		// current block got added but channel is now full
		if s.currentChannel.IsFull() {
			break
		}
	}

	if blocksAdded == len(s.blocks) {
		// all blocks processed, reuse slice
		s.blocks = s.blocks[:0]
	} else {
		// remove processed blocks
		s.blocks = s.blocks[blocksAdded:]
	}

	s.metr.RecordL2BlocksAdded(latestL2ref,
		blocksAdded,
		len(s.blocks),
		s.currentChannel.InputBytes(),
		s.currentChannel.ReadyBytes())
	s.log.Debug("Added blocks to channel",
		"blocks_added", blocksAdded,
		"blocks_pending", len(s.blocks),
		"channel_full", s.currentChannel.IsFull(),
		"input_bytes", s.currentChannel.InputBytes(),
		"ready_bytes", s.currentChannel.ReadyBytes(),
	)
	return nil
}

func (s *channelManager) outputFrames() error {
	if err := s.currentChannel.OutputFrames(); err != nil {
		return fmt.Errorf("creating frames with channel builder: %w", err)
	}
	if !s.currentChannel.IsFull() {
		return nil
	}

	inBytes, outBytes := s.currentChannel.InputBytes(), s.currentChannel.OutputBytes()
	s.metr.RecordChannelClosed(
		s.currentChannel.ID(),
		len(s.blocks),
		s.currentChannel.TotalFrames(),
		inBytes,
		outBytes,
		s.currentChannel.FullErr(),
	)

	var comprRatio float64
	if inBytes > 0 {
		comprRatio = float64(outBytes) / float64(inBytes)
	}
	s.log.Info("Channel closed",
		"id", s.currentChannel.ID(),
		"blocks_pending", len(s.blocks),
		"num_frames", s.currentChannel.TotalFrames(),
		"input_bytes", inBytes,
		"output_bytes", outBytes,
		"full_reason", s.currentChannel.FullErr(),
		"compr_ratio", comprRatio,
	)
	return nil
}

// AddL2Block adds an L2 block to the internal blocks queue. It returns ErrReorg
// if the block does not extend the last block loaded into the state. If no
// blocks were added yet, the parent hash check is skipped.
func (s *channelManager) AddL2Block(block *types.Block) error {
	if s.tip != (common.Hash{}) && s.tip != block.ParentHash() {
		return ErrReorg
	}
	s.blocks = append(s.blocks, block)
	s.tip = block.Hash()

	return nil
}

func l2BlockRefFromBlockAndL1Info(block *types.Block, l1info derive.L1BlockInfo) eth.L2BlockRef {
	return eth.L2BlockRef{
		Hash:           block.Hash(),
		Number:         block.NumberU64(),
		ParentHash:     block.ParentHash(),
		Time:           block.Time(),
		L1Origin:       eth.BlockID{Hash: l1info.BlockHash, Number: l1info.Number},
		SequenceNumber: l1info.SequenceNumber,
	}
}

// Close closes the current pending channel, if one exists, outputs any remaining frames,
// and prevents the creation of any new channels.
// Any outputted frames still need to be published.
func (s *channelManager) Close() error {
	if s.closed {
		return nil
	}

	s.closed = true

	// Any pending state can be proactively cleared if there are no submitted transactions
	for _, ch := range s.channelQueue {
		if ch.NoneSubmitted() {
			s.removePendingChannel(ch)
		}
	}

	if s.currentChannel == nil {
		return nil
	}

	s.currentChannel.Close()

	return s.outputFrames()
}
