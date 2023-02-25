package batcher

import (
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var ErrReorg = errors.New("block does not extend existing chain")

// txID is an opaque identifier for a transaction.
// It's internal fields should not be inspected after creation & are subject to change.
// This ID must be trivially comparable & work as a map key.
type txID struct {
	chID        derive.ChannelID
	frameNumber uint16
}

func (id txID) String() string {
	return fmt.Sprintf("%s:%d", id.chID.String(), id.frameNumber)
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (id txID) TerminalString() string {
	return fmt.Sprintf("%s:%d", id.chID.TerminalString(), id.frameNumber)
}

type taggedData struct {
	data []byte
	id   txID
}

// channelManager stores a contiguous set of blocks & turns them into channels.
// Upon receiving tx confirmation (or a tx failure), it does channel error handling.
//
// For simplicity, it only creates a single pending channel at a time & waits for
// the channel to either successfully be submitted or timeout before creating a new
// channel.
// Functions on channelManager are not safe for concurrent access.
type channelManager struct {
	log log.Logger
	cfg ChannelConfig

	// All blocks since the last request for new tx data.
	blocks []*types.Block
	// last block hash - for reorg detection
	tip common.Hash

	// Pending data returned by TxData waiting on Tx Confirmed/Failed

	// pending channel builder
	pendingChannel *channelBuilder
	// Set of unconfirmed txID -> frame data. For tx resubmission
	pendingTransactions map[txID][]byte
	// Set of confirmed txID -> inclusion block. For determining if the channel is timed out
	confirmedTransactions map[txID]eth.BlockID
}

func NewChannelManager(log log.Logger, cfg ChannelConfig) *channelManager {
	return &channelManager{
		log:                   log,
		cfg:                   cfg,
		pendingTransactions:   make(map[txID][]byte),
		confirmedTransactions: make(map[txID]eth.BlockID),
	}
}

// Clear clears the entire state of the channel manager.
// It is intended to be used after an L2 reorg.
func (s *channelManager) Clear() {
	s.log.Trace("clearing channel manager state")
	s.blocks = s.blocks[:0]
	s.tip = common.Hash{}
	s.clearPendingChannel()
}

// TxFailed records a transaction as failed. It will attempt to resubmit the data
// in the failed transaction.
func (s *channelManager) TxFailed(id txID) {
	if data, ok := s.pendingTransactions[id]; ok {
		s.log.Trace("marked transaction as failed", "id", id)
		s.pendingChannel.PushFrame(id, data)
		delete(s.pendingTransactions, id)
	} else {
		s.log.Warn("unknown transaction marked as failed", "id", id)
	}
}

// TxConfirmed marks a transaction as confirmed on L1. Unfortunately even if all frames in
// a channel have been marked as confirmed on L1 the channel may be invalid & need to be
// resubmitted.
// This function may reset the pending channel if the pending channel has timed out.
func (s *channelManager) TxConfirmed(id txID, inclusionBlock eth.BlockID) {
	s.log.Trace("marked transaction as confirmed", "id", id, "block", inclusionBlock)
	if _, ok := s.pendingTransactions[id]; !ok {
		s.log.Warn("unknown transaction marked as confirmed", "id", id, "block", inclusionBlock)
		// TODO: This can occur if we clear the channel while there are still pending transactions
		// We need to keep track of stale transactions instead
		return
	}
	delete(s.pendingTransactions, id)
	s.confirmedTransactions[id] = inclusionBlock
	s.pendingChannel.FramePublished(inclusionBlock.Number)

	// If this channel timed out, put the pending blocks back into the local saved blocks
	// and then reset this state so it can try to build a new channel.
	if s.pendingChannelIsTimedOut() {
		s.log.Warn("Channel timed out", "chID", s.pendingChannel.ID())
		s.blocks = append(s.pendingChannel.Blocks(), s.blocks...)
		s.clearPendingChannel()
	}
	// If we are done with this channel, record that.
	if s.pendingChannelIsFullySubmitted() {
		s.log.Info("Channel is fully submitted", "chID", s.pendingChannel.ID())
		s.clearPendingChannel()
	}
}

// clearPendingChannel resets all pending state back to an initialized but empty state.
// TODO: Create separate "pending" state
func (s *channelManager) clearPendingChannel() {
	s.pendingChannel = nil
	s.pendingTransactions = make(map[txID][]byte)
	s.confirmedTransactions = make(map[txID]eth.BlockID)
}

// pendingChannelIsTimedOut returns true if submitted channel has timed out.
// A channel has timed out if the difference in L1 Inclusion blocks between
// the first & last included block is greater than or equal to the channel timeout.
func (s *channelManager) pendingChannelIsTimedOut() bool {
	if s.pendingChannel == nil {
		return false // no channel to be timed out
	}
	// No confirmed transactions => not timed out
	if len(s.confirmedTransactions) == 0 {
		return false
	}
	// If there are confirmed transactions, find the first + last confirmed block numbers
	min := uint64(math.MaxUint64)
	max := uint64(0)
	for _, inclusionBlock := range s.confirmedTransactions {
		if inclusionBlock.Number < min {
			min = inclusionBlock.Number
		}
		if inclusionBlock.Number > max {
			max = inclusionBlock.Number
		}
	}
	return max-min >= s.cfg.ChannelTimeout
}

// pendingChannelIsFullySubmitted returns true if the channel has been fully submitted.
func (s *channelManager) pendingChannelIsFullySubmitted() bool {
	if s.pendingChannel == nil {
		return false // todo: can decide either way here. Nonsensical answer though
	}
	return s.pendingChannel.IsFull() && len(s.pendingTransactions)+s.pendingChannel.NumFrames() == 0
}

// nextTxData pops off s.datas & handles updating the internal state
func (s *channelManager) nextTxData() ([]byte, txID, error) {
	if s.pendingChannel == nil || !s.pendingChannel.HasFrame() {
		s.log.Trace("no next tx data")
		return nil, txID{}, io.EOF // TODO: not enough data error instead
	}

	id, data := s.pendingChannel.NextFrame()
	// prepend version byte for first frame of transaction
	// TODO: more memory efficient solution; shouldn't be responsibility of
	// channelBuilder though.
	data = append([]byte{0}, data...)

	s.log.Trace("returning next tx data", "id", id)
	s.pendingTransactions[id] = data
	return data, id, nil
}

// TxData returns the next tx data that should be submitted to L1.
//
// It currently only uses one frame per transaction. If the pending channel is
// full, it only returns the remaining frames of this channel until it got
// successfully fully sent to L1. It returns io.EOF if there's no pending frame.
func (s *channelManager) TxData(l1Head eth.BlockID) ([]byte, txID, error) {
	dataPending := s.pendingChannel != nil && s.pendingChannel.HasFrame()
	s.log.Debug("Requested tx data", "l1Head", l1Head, "data_pending", dataPending, "blocks_pending", len(s.blocks))

	// Short circuit if there is a pending frame.
	if dataPending {
		return s.nextTxData()
	}

	// No pending frame, so we have to add new blocks to the channel

	// If we have no saved blocks, we will not be able to create valid frames
	if len(s.blocks) == 0 {
		return nil, txID{}, io.EOF
	}

	if err := s.ensurePendingChannel(l1Head); err != nil {
		return nil, txID{}, err
	}

	if err := s.processBlocks(); err != nil {
		return nil, txID{}, err
	}

	// Register current L1 head only after all pending blocks have been
	// processed. Even if a timeout will be triggered now, it is better to have
	// all pending blocks be included in this channel for submission.
	s.registerL1Block(l1Head)

	if err := s.pendingChannel.OutputFrames(); err != nil {
		return nil, txID{}, fmt.Errorf("creating frames with channel builder: %w", err)
	}

	return s.nextTxData()
}

func (s *channelManager) ensurePendingChannel(l1Head eth.BlockID) error {
	if s.pendingChannel != nil {
		return nil
	}

	cb, err := newChannelBuilder(s.cfg)
	if err != nil {
		return fmt.Errorf("creating new channel: %w", err)
	}
	s.pendingChannel = cb
	s.log.Info("Created channel", "chID", cb.ID(), "l1Head", l1Head)

	return nil
}

// registerL1Block registers the given block at the pending channel.
func (s *channelManager) registerL1Block(l1Head eth.BlockID) {
	s.pendingChannel.RegisterL1Block(l1Head.Number)
	s.log.Debug("new L1-block registered at channel builder",
		"l1Head", l1Head,
		"channel_full", s.pendingChannel.IsFull(),
		"full_reason", s.pendingChannel.FullErr(),
	)
}

// processBlocks adds blocks from the blocks queue to the pending channel until
// either the queue got exhausted or the channel is full.
func (s *channelManager) processBlocks() error {
	var blocksAdded int
	var _chFullErr *ChannelFullError // throw away, just for type checking
	for i, block := range s.blocks {
		if err := s.pendingChannel.AddBlock(block); errors.As(err, &_chFullErr) {
			// current block didn't get added because channel is already full
			break
		} else if err != nil {
			return fmt.Errorf("adding block[%d] to channel builder: %w", i, err)
		}
		blocksAdded += 1
		// current block got added but channel is now full
		if s.pendingChannel.IsFull() {
			break
		}
	}

	s.log.Debug("Added blocks to channel",
		"blocks_added", blocksAdded,
		"channel_full", s.pendingChannel.IsFull(),
		"blocks_pending", len(s.blocks)-blocksAdded,
		"input_bytes", s.pendingChannel.InputBytes(),
	)
	if blocksAdded == len(s.blocks) {
		// all blocks processed, reuse slice
		s.blocks = s.blocks[:0]
	} else {
		// remove processed blocks
		s.blocks = s.blocks[blocksAdded:]
	}
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
