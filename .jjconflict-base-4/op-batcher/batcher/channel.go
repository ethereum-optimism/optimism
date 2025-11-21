package batcher

import (
	"math"

	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

// channel is a lightweight wrapper around a ChannelBuilder which keeps track of pending
// and confirmed transactions for a single channel.
type channel struct {
	*ChannelBuilder // pending channel builder

	log  log.Logger
	metr metrics.Metricer
	cfg  ChannelConfig

	pendingTransactions   map[string]txData      // Set of unconfirmed txID -> tx data. For tx resubmission
	confirmedTransactions map[string]eth.BlockID // Set of confirmed txID -> inclusion block. For determining if the channel is timed out

	minInclusionBlock uint64 // Inclusion block number of first confirmed TX
	maxInclusionBlock uint64 // Inclusion block number of last confirmed TX
}

func newChannel(log log.Logger, metr metrics.Metricer, cfg ChannelConfig, rollupCfg *rollup.Config, latestL1OriginBlockNum uint64, channelOut derive.ChannelOut) *channel {
	cb := NewChannelBuilderWithChannelOut(log, cfg, rollupCfg, latestL1OriginBlockNum, channelOut)
	return &channel{
		ChannelBuilder:        cb,
		log:                   log,
		metr:                  metr,
		cfg:                   cfg,
		pendingTransactions:   make(map[string]txData),
		confirmedTransactions: make(map[string]eth.BlockID),
		minInclusionBlock:     math.MaxUint64,
	}
}

// TxFailed records a transaction as failed. It will attempt to resubmit the data
// in the failed transaction.
func (c *channel) TxFailed(id string) {
	if data, ok := c.pendingTransactions[id]; ok {
		c.log.Trace("marked transaction as failed", "id", id)
		// Rewind to the first frame of the failed tx
		// -- the frames are ordered, and we want to send them
		// all again.
		c.RewindFrameCursor(data.Frames()[0])
		delete(c.pendingTransactions, id)
	} else {
		c.log.Warn("unknown transaction marked as failed", "id", id)
	}

	c.metr.RecordBatchTxFailed()
}

// TxConfirmed marks a transaction as confirmed on L1. Returns a bool indicating
// whether the channel timed out on chain.
func (c *channel) TxConfirmed(id string, inclusionBlock eth.BlockID) bool {
	c.metr.RecordBatchTxSuccess()
	c.log.Debug("marked transaction as confirmed", "id", id, "block", inclusionBlock)
	if _, ok := c.pendingTransactions[id]; !ok {
		c.log.Warn("unknown transaction marked as confirmed", "id", id, "block", inclusionBlock)
		// TODO: This can occur if we clear the channel while there are still pending transactions
		// We need to keep track of stale transactions instead
		return false
	}
	delete(c.pendingTransactions, id)
	c.confirmedTransactions[id] = inclusionBlock
	c.FramePublished(inclusionBlock.Number)

	// Update min/max inclusion blocks for timeout check
	c.minInclusionBlock = min(c.minInclusionBlock, inclusionBlock.Number)
	c.maxInclusionBlock = max(c.maxInclusionBlock, inclusionBlock.Number)

	if c.isFullySubmitted() {
		c.metr.RecordChannelFullySubmitted(c.ID())
		c.log.Info("Channel is fully submitted", "id", c.ID(), "min_inclusion_block", c.minInclusionBlock, "max_inclusion_block", c.maxInclusionBlock)
	}

	// If this channel timed out, put the pending blocks back into the local saved blocks
	// and then reset this state so it can try to build a new channel.
	if c.isTimedOut() {
		c.metr.RecordChannelTimedOut(c.ID())
		c.log.Warn("Channel timed out", "id", c.ID(), "min_inclusion_block", c.minInclusionBlock, "max_inclusion_block", c.maxInclusionBlock)
		return true
	}

	return false
}

// isTimedOut returns true if submitted channel has timed out.
// A channel has timed out if the difference in L1 Inclusion blocks between
// the first & last included block is greater than or equal to the channel timeout.
func (c *channel) isTimedOut() bool {
	// Prior to the granite hard fork activating, the use of the shorter ChannelTimeout here may cause the batcher
	// to believe the channel timed out when it was valid. It would then resubmit the blocks needlessly.
	// This wastes batcher funds but doesn't cause any problems for the chain progressing safe head.
	return len(c.confirmedTransactions) > 0 && c.maxInclusionBlock-c.minInclusionBlock >= c.cfg.ChannelTimeout
}

// isFullySubmitted returns true if the channel has been fully submitted (all transactions are confirmed).
func (c *channel) isFullySubmitted() bool {
	return c.IsFull() && len(c.pendingTransactions)+c.PendingFrames() == 0
}

func (c *channel) noneSubmitted() bool {
	return len(c.confirmedTransactions) == 0 && len(c.pendingTransactions) == 0
}

// NextTxData dequeues the next frames from the channel and returns them encoded in a tx data packet.
// If cfg.UseBlobs is false, it returns txData with a single frame.
// If cfg.UseBlobs is true, it will read frames from its channel builder
// until it either doesn't have more frames or the target number of frames is reached.
//
// NextTxData should only be called after HasTxData returned true.
func (c *channel) NextTxData() txData {
	nf := c.cfg.MaxFramesPerTx()
	txdata := txData{frames: make([]frameData, 0, nf), asBlob: c.cfg.UseBlobs}
	for i := 0; i < nf && c.HasPendingFrame(); i++ {
		frame := c.NextFrame()
		txdata.frames = append(txdata.frames, frame)
	}

	id := txdata.ID().String()
	c.log.Debug("returning next tx data", "id", id, "num_frames", len(txdata.frames), "as_blob", txdata.asBlob)
	c.pendingTransactions[id] = txdata

	return txdata
}

func (c *channel) HasTxData() bool {
	if c.IsFull() || // If the channel is full, we should start to submit it
		!c.cfg.UseBlobs { // If using calldata, we only send one frame per tx
		return c.HasPendingFrame()
	}
	// Collect enough frames if channel is not full yet
	return c.PendingFrames() >= int(c.cfg.MaxFramesPerTx())
}

func (c *channel) MaxInclusionBlock() uint64 {
	return c.maxInclusionBlock
}
