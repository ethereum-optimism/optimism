package batcher

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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
// Public functions on channelManager are safe for concurrent access.
type channelManager struct {
	mu        sync.Mutex
	log       log.Logger
	metr      metrics.Metricer
	cfg       ChannelConfig
	rollupCfg *rollup.Config

	// All blocks since the last request for new tx data.
	blocks []*types.Block
	// The latest L1 block from all the L2 blocks in the most recently closed channel
	l1OriginLastClosedChannel eth.BlockID
	// last block hash - for reorg detection
	tip common.Hash

	// channel to write new block data to
	currentChannel *channel
	// channels to read frame data from, for writing batches onchain
	channelQueue []*channel
	// used to lookup channels by tx ID upon tx success / failure
	txChannels map[string]*channel

	// if set to true, prevents production of any new channel frames
	closed bool
}

func NewChannelManager(log log.Logger, metr metrics.Metricer, cfg ChannelConfig, rollupCfg *rollup.Config) *channelManager {
	return &channelManager{
		log:        log,
		metr:       metr,
		cfg:        cfg,
		rollupCfg:  rollupCfg,
		txChannels: make(map[string]*channel),
	}
}

// Clear clears the entire state of the channel manager.
// It is intended to be used before launching op-batcher and after an L2 reorg.
func (s *channelManager) Clear(l1OriginLastClosedChannel eth.BlockID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.log.Trace("clearing channel manager state")
	s.blocks = s.blocks[:0]
	s.l1OriginLastClosedChannel = l1OriginLastClosedChannel
	s.tip = common.Hash{}
	s.closed = false
	s.currentChannel = nil
	s.channelQueue = nil
	s.txChannels = make(map[string]*channel)
}

// TxFailed records a transaction as failed. It will attempt to resubmit the data
// in the failed transaction.
func (s *channelManager) TxFailed(_id txID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := _id.String()
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
func (s *channelManager) TxConfirmed(_id txID, inclusionBlock eth.BlockID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := _id.String()
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
	if channel == nil || !channel.HasTxData() {
		s.log.Trace("no next tx data")
		return txData{}, io.EOF // TODO: not enough data error instead
	}
	tx := channel.NextTxData()
	s.txChannels[tx.ID().String()] = channel
	return tx, nil
}

// TxData returns the next tx data that should be submitted to L1.
//
// If the pending channel is
// full, it only returns the remaining frames of this channel until it got
// successfully fully sent to L1. It returns io.EOF if there's no pending tx data.
func (s *channelManager) TxData(l1Head eth.BlockID) (txData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var firstWithTxData *channel
	for _, ch := range s.channelQueue {
		if ch.HasTxData() {
			firstWithTxData = ch
			break
		}
	}

	dataPending := firstWithTxData != nil && firstWithTxData.HasTxData()
	s.log.Debug("Requested tx data", "l1Head", l1Head, "txdata_pending", dataPending, "blocks_pending", len(s.blocks))

	// Short circuit if there is pending tx data or the channel manager is closed.
	if dataPending || s.closed {
		return s.nextTxData(firstWithTxData)
	}

	// No pending tx data, so we have to add new blocks to the channel

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

	pc, err := newChannel(s.log, s.metr, s.cfg, s.rollupCfg, s.l1OriginLastClosedChannel.Number)
	if err != nil {
		return fmt.Errorf("creating new channel: %w", err)
	}

	s.currentChannel = pc
	s.channelQueue = append(s.channelQueue, pc)

	s.log.Info("Created channel",
		"id", pc.ID(),
		"l1Head", l1Head,
		"l1OriginLastClosedChannel", s.l1OriginLastClosedChannel,
		"blocks_pending", len(s.blocks),
		"batch_type", s.cfg.BatchType,
		"compression_algo", s.cfg.CompressorConfig.CompressionAlgo,
		"target_num_frames", s.cfg.TargetNumFrames,
		"max_frame_size", s.cfg.MaxFrameSize,
	)
	s.metr.RecordChannelOpened(pc.ID(), len(s.blocks))

	return nil
}

// registerL1Block registers the given block at the pending channel.
func (s *channelManager) registerL1Block(l1Head eth.BlockID) {
	s.currentChannel.CheckTimeout(l1Head.Number)
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
		s.log.Debug("Added block to channel", "id", s.currentChannel.ID(), "block", eth.ToBlockID(block))

		blocksAdded += 1
		latestL2ref = l2BlockRefFromBlockAndL1Info(block, l1info)
		s.metr.RecordL2BlockInChannel(block)
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

	lastClosedL1Origin := s.currentChannel.LatestL1Origin()
	if lastClosedL1Origin.Number > s.l1OriginLastClosedChannel.Number {
		s.l1OriginLastClosedChannel = lastClosedL1Origin
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
		"oldest_l1_origin", s.currentChannel.OldestL1Origin(),
		"l1_origin", lastClosedL1Origin,
		"oldest_l2", s.currentChannel.OldestL2(),
		"latest_l2", s.currentChannel.LatestL2(),
		"full_reason", s.currentChannel.FullErr(),
		"compr_ratio", comprRatio,
		"latest_l1_origin", s.l1OriginLastClosedChannel,
	)
	return nil
}

// AddL2Block adds an L2 block to the internal blocks queue. It returns ErrReorg
// if the block does not extend the last block loaded into the state. If no
// blocks were added yet, the parent hash check is skipped.
func (s *channelManager) AddL2Block(block *types.Block) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tip != (common.Hash{}) && s.tip != block.ParentHash() {
		return ErrReorg
	}

	s.metr.RecordL2BlockInPendingQueue(block)
	s.blocks = append(s.blocks, block)
	s.tip = block.Hash()

	return nil
}

func l2BlockRefFromBlockAndL1Info(block *types.Block, l1info *derive.L1BlockInfo) eth.L2BlockRef {
	return eth.L2BlockRef{
		Hash:           block.Hash(),
		Number:         block.NumberU64(),
		ParentHash:     block.ParentHash(),
		Time:           block.Time(),
		L1Origin:       eth.BlockID{Hash: l1info.BlockHash, Number: l1info.Number},
		SequenceNumber: l1info.SequenceNumber,
	}
}

var ErrPendingAfterClose = errors.New("pending channels remain after closing channel-manager")

// Close clears any pending channels that are not in-flight already, to leave a clean derivation state.
// Close then marks the remaining current open channel, if any, as "full" so it can be submitted as well.
// Close does NOT immediately output frames for the current remaining channel:
// as this might error, due to limitations on a single channel.
// Instead, this is part of the pending-channel submission work: after closing,
// the caller SHOULD drain pending channels by generating TxData repeatedly until there is none left (io.EOF).
// A ErrPendingAfterClose error will be returned if there are any remaining pending channels to submit.
func (s *channelManager) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}

	s.closed = true
	s.log.Info("Channel manager is closing")

	// Any pending state can be proactively cleared if there are no submitted transactions
	for _, ch := range s.channelQueue {
		if ch.NoneSubmitted() {
			s.log.Info("Channel has no past or pending submission - dropping", "id", ch.ID())
			s.removePendingChannel(ch)
		} else {
			s.log.Info("Channel is in-flight and will need to be submitted after close", "id", ch.ID(), "confirmed", len(ch.confirmedTransactions), "pending", len(ch.pendingTransactions))
		}
	}
	s.log.Info("Reviewed all pending channels on close", "remaining", len(s.channelQueue))

	if s.currentChannel == nil {
		return nil
	}

	// If the channel is already full, we don't need to close it or output frames.
	// This would already have happened in TxData.
	if !s.currentChannel.IsFull() {
		// Force-close the remaining open channel early (if not already closed):
		// it will be marked as "full" due to service termination.
		s.currentChannel.Close()

		// Final outputFrames call in case there was unflushed data in the compressor.
		if err := s.outputFrames(); err != nil {
			return fmt.Errorf("outputting frames during close: %w", err)
		}
	}

	if s.currentChannel.HasTxData() {
		// Make it clear to the caller that there is remaining pending work.
		return ErrPendingAfterClose
	}
	return nil
}
