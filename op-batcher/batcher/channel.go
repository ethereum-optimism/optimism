package batcher

import (
	"fmt"
	"math"

	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// channel is a lightweight wrapper around a ChannelBuilder which keeps track of pending
// and confirmed transactions for a single channel.
type channel struct {
	log  log.Logger
	metr metrics.Metricer
	cfg  ChannelConfig

	// pending channel builder
	channelBuilder *ChannelBuilder
	// Set of unconfirmed txID -> tx data. For tx resubmission
	pendingTransactions map[string]txData
	// Set of confirmed txID -> inclusion block. For determining if the channel is timed out
	confirmedTransactions map[string]eth.BlockID

	// True if confirmed TX list is updated. Set to false after updated min/max inclusion blocks.
	confirmedTxUpdated bool
	// Inclusion block number of first confirmed TX
	minInclusionBlock uint64
	// Inclusion block number of last confirmed TX
	maxInclusionBlock uint64
}

func newChannel(log log.Logger, metr metrics.Metricer, cfg ChannelConfig, rollupCfg *rollup.Config, latestL1OriginBlockNum uint64) (*channel, error) {
	cb, err := NewChannelBuilder(cfg, rollupCfg, latestL1OriginBlockNum)
	if err != nil {
		return nil, fmt.Errorf("creating new channel: %w", err)
	}

	return &channel{
		log:                   log,
		metr:                  metr,
		cfg:                   cfg,
		channelBuilder:        cb,
		pendingTransactions:   make(map[string]txData),
		confirmedTransactions: make(map[string]eth.BlockID),
	}, nil
}

// TxFailed records a transaction as failed. It will attempt to resubmit the data
// in the failed transaction.
func (s *channel) TxFailed(id string) {
	if data, ok := s.pendingTransactions[id]; ok {
		s.log.Trace("marked transaction as failed", "id", id)
		// Note: when the batcher is changed to send multiple frames per tx,
		// this needs to be changed to iterate over all frames of the tx data
		// and re-queue them.
		s.channelBuilder.PushFrames(data.Frames()...)
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
func (s *channel) TxConfirmed(id string, inclusionBlock eth.BlockID) (bool, []*types.Block) {
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
	s.confirmedTxUpdated = true
	s.channelBuilder.FramePublished(inclusionBlock.Number)

	// If this channel timed out, put the pending blocks back into the local saved blocks
	// and then reset this state so it can try to build a new channel.
	if s.isTimedOut() {
		s.metr.RecordChannelTimedOut(s.ID())
		s.log.Warn("Channel timed out", "id", s.ID(), "min_inclusion_block", s.minInclusionBlock, "max_inclusion_block", s.maxInclusionBlock)
		return true, s.channelBuilder.Blocks()
	}
	// If we are done with this channel, record that.
	if s.isFullySubmitted() {
		s.metr.RecordChannelFullySubmitted(s.ID())
		s.log.Info("Channel is fully submitted", "id", s.ID(), "min_inclusion_block", s.minInclusionBlock, "max_inclusion_block", s.maxInclusionBlock)
		return true, nil
	}

	return false, nil
}

// Timeout returns the channel timeout L1 block number. If there is no timeout set, it returns 0.
func (s *channel) Timeout() uint64 {
	return s.channelBuilder.Timeout()
}

// updateInclusionBlocks finds the first & last confirmed tx and saves its inclusion numbers
func (s *channel) updateInclusionBlocks() {
	if len(s.confirmedTransactions) == 0 || !s.confirmedTxUpdated {
		return
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
	s.minInclusionBlock = min
	s.maxInclusionBlock = max
	s.confirmedTxUpdated = false
}

// pendingChannelIsTimedOut returns true if submitted channel has timed out.
// A channel has timed out if the difference in L1 Inclusion blocks between
// the first & last included block is greater than or equal to the channel timeout.
func (s *channel) isTimedOut() bool {
	// Update min/max inclusion blocks for timeout check
	s.updateInclusionBlocks()
	// Prior to the granite hard fork activating, the use of the shorter ChannelTimeout here may cause the batcher
	// to believe the channel timed out when it was valid. It would then resubmit the blocks needlessly.
	// This wastes batcher funds but doesn't cause any problems for the chain progressing safe head.
	return s.maxInclusionBlock-s.minInclusionBlock >= s.cfg.ChannelTimeout
}

// pendingChannelIsFullySubmitted returns true if the channel has been fully submitted.
func (s *channel) isFullySubmitted() bool {
	// Update min/max inclusion blocks for timeout check
	s.updateInclusionBlocks()
	return s.IsFull() && len(s.pendingTransactions)+s.PendingFrames() == 0
}

func (s *channel) NoneSubmitted() bool {
	return len(s.confirmedTransactions) == 0 && len(s.pendingTransactions) == 0
}

func (s *channel) ID() derive.ChannelID {
	return s.channelBuilder.ID()
}

// NextTxData dequeues the next frames from the channel and returns them encoded in a tx data packet.
// If cfg.UseBlobs is false, it returns txData with a single frame.
// If cfg.UseBlobs is true, it will read frames from its channel builder
// until it either doesn't have more frames or the target number of frames is reached.
//
// NextTxData should only be called after HasTxData returned true.
func (s *channel) NextTxData() txData {
	nf := s.cfg.MaxFramesPerTx()
	txdata := txData{frames: make([]frameData, 0, nf), asBlob: s.cfg.UseBlobs}
	for i := 0; i < nf && s.channelBuilder.HasFrame(); i++ {
		frame := s.channelBuilder.NextFrame()
		txdata.frames = append(txdata.frames, frame)
	}

	id := txdata.ID().String()
	s.log.Debug("returning next tx data", "id", id, "num_frames", len(txdata.frames), "as_blob", txdata.asBlob)
	s.pendingTransactions[id] = txdata

	return txdata
}

func (s *channel) HasTxData() bool {
	if s.IsFull() || // If the channel is full, we should start to submit it
		!s.cfg.UseBlobs { // If using calldata, we only send one frame per tx
		return s.channelBuilder.HasFrame()
	}
	// Collect enough frames if channel is not full yet
	return s.channelBuilder.PendingFrames() >= int(s.cfg.MaxFramesPerTx())
}

func (s *channel) IsFull() bool {
	return s.channelBuilder.IsFull()
}

func (s *channel) FullErr() error {
	return s.channelBuilder.FullErr()
}

func (s *channel) CheckTimeout(l1BlockNum uint64) {
	s.channelBuilder.CheckTimeout(l1BlockNum)
}

func (s *channel) AddBlock(block *types.Block) (*derive.L1BlockInfo, error) {
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

// LatestL1Origin returns the latest L1 block origin from all the L2 blocks that have been added to the channel
func (c *channel) LatestL1Origin() eth.BlockID {
	return c.channelBuilder.LatestL1Origin()
}

// OldestL1Origin returns the oldest L1 block origin from all the L2 blocks that have been added to the channel
func (c *channel) OldestL1Origin() eth.BlockID {
	return c.channelBuilder.OldestL1Origin()
}

// LatestL2 returns the latest L2 block from all the L2 blocks that have been added to the channel
func (c *channel) LatestL2() eth.BlockID {
	return c.channelBuilder.LatestL2()
}

// OldestL2 returns the oldest L2 block from all the L2 blocks that have been added to the channel
func (c *channel) OldestL2() eth.BlockID {
	return c.channelBuilder.OldestL2()
}

func (s *channel) Close() {
	s.channelBuilder.Close()
}
