package batcher

import (
	"fmt"
	"math"

	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// channel is a lightweight wrapper around a channelBuilder which keeps track of pending
// and confirmed transactions for a single channel.
type channel struct {
	log  log.Logger
	metr metrics.Metricer
	cfg  ChannelConfig

	// pending channel builder
	channelBuilder *channelBuilder
	// Set of unconfirmed txID -> frame data. For tx resubmission
	pendingTransactions map[txID]txData
	// Set of confirmed txID -> inclusion block. For determining if the channel is timed out
	confirmedTransactions map[txID]eth.BlockID
}

func newChannel(log log.Logger, metr metrics.Metricer, cfg ChannelConfig) (*channel, error) {
	cb, err := newChannelBuilder(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating new channel: %w", err)
	}
	return &channel{
		log:                   log,
		metr:                  metr,
		cfg:                   cfg,
		channelBuilder:        cb,
		pendingTransactions:   make(map[txID]txData),
		confirmedTransactions: make(map[txID]eth.BlockID),
	}, nil
}

// TxFailed records a transaction as failed. It will attempt to resubmit the data
// in the failed transaction.
func (s *channel) TxFailed(id txID) {
	if data, ok := s.pendingTransactions[id]; ok {
		s.log.Trace("marked transaction as failed", "id", id)
		// Note: when the batcher is changed to send multiple frames per tx,
		// this needs to be changed to iterate over all frames of the tx data
		// and re-queue them.
		s.channelBuilder.PushFrame(data.Frame())
		delete(s.pendingTransactions, id)
	} else {
		s.log.Warn("unknown transaction marked as failed", "id", id)
	}

	s.metr.RecordBatchTxFailed()
}

// TxConfirmed marks a transaction as confirmed on L1. Unfortunately even if all frames in
// a channel have been marked as confirmed on L1 the channel may be invalid & need to be
// resubmitted.
// This function may reset the pending channel if the pending channel has timed out.
func (s *channel) TxConfirmed(id txID, inclusionBlock eth.BlockID) (bool, []*types.Block) {
	s.metr.RecordBatchTxSubmitted()
	s.log.Debug("marked transaction as confirmed", "id", id, "block", inclusionBlock)
	if _, ok := s.pendingTransactions[id]; !ok {
		s.log.Warn("unknown transaction marked as confirmed", "id", id, "block", inclusionBlock)
		// TODO: This can occur if we clear the channel while there are still pending transactions
		// We need to keep track of stale transactions instead
		return false, nil
	}
	delete(s.pendingTransactions, id)
	s.confirmedTransactions[id] = inclusionBlock
	s.channelBuilder.FramePublished(inclusionBlock.Number)

	// If this channel timed out, put the pending blocks back into the local saved blocks
	// and then reset this state so it can try to build a new channel.
	if s.isTimedOut() {
		s.metr.RecordChannelTimedOut(s.ID())
		s.log.Warn("Channel timed out", "id", s.ID())
		return true, s.channelBuilder.Blocks()
	}
	// If we are done with this channel, record that.
	if s.isFullySubmitted() {
		s.metr.RecordChannelFullySubmitted(s.ID())
		s.log.Info("Channel is fully submitted", "id", s.ID())
		return true, nil
	}

	return false, nil
}

// pendingChannelIsTimedOut returns true if submitted channel has timed out.
// A channel has timed out if the difference in L1 Inclusion blocks between
// the first & last included block is greater than or equal to the channel timeout.
func (s *channel) isTimedOut() bool {
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
func (s *channel) isFullySubmitted() bool {
	return s.IsFull() && len(s.pendingTransactions)+s.PendingFrames() == 0
}

func (s *channel) NoneSubmitted() bool {
	return len(s.confirmedTransactions) == 0 && len(s.pendingTransactions) == 0
}

func (s *channel) ID() derive.ChannelID {
	return s.channelBuilder.ID()
}

func (s *channel) NextTxData() txData {
	frame := s.channelBuilder.NextFrame()

	txdata := txData{frame}
	id := txdata.ID()

	s.log.Trace("returning next tx data", "id", id)
	s.pendingTransactions[id] = txdata

	return txdata
}

func (s *channel) HasFrame() bool {
	return s.channelBuilder.HasFrame()
}

func (s *channel) IsFull() bool {
	return s.channelBuilder.IsFull()
}

func (s *channel) FullErr() error {
	return s.channelBuilder.FullErr()
}

func (s *channel) RegisterL1Block(l1BlockNum uint64) {
	s.channelBuilder.RegisterL1Block(l1BlockNum)
}

func (s *channel) AddBlock(block *types.Block) (derive.L1BlockInfo, error) {
	return s.channelBuilder.AddBlock(block)
}

func (s *channel) InputBytes() int {
	return s.channelBuilder.InputBytes()
}

func (s *channel) ReadyBytes() int {
	return s.channelBuilder.ReadyBytes()
}

func (s *channel) OutputBytes() int {
	return s.channelBuilder.OutputBytes()
}

func (s *channel) TotalFrames() int {
	return s.channelBuilder.TotalFrames()
}

func (s *channel) PendingFrames() int {
	return s.channelBuilder.PendingFrames()
}

func (s *channel) OutputFrames() error {
	return s.channelBuilder.OutputFrames()
}

func (s *channel) Close() {
	s.channelBuilder.Close()
}
