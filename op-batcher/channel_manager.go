package op_batcher

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
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
	log            log.Logger
	channelTimeout uint64

	// All blocks since the last request for new tx data.
	blocks []*types.Block
	datas  []taggedData

	// Pending data returned by TxData waiting on Tx Confirmed/Failed
	// id of the pending channel
	pendingChannel derive.ChannelID
	// list of blocks in the channel. Saved in case the channel must be rebuilt
	pendingBlocks []*types.Block
	// Set of unconfirmed txID -> frame data. For tx resubmission
	pendingTransactions map[txID][]byte
	// Set of confirmed txID -> inclusion block. For determining if the channel is timed out
	confirmedTransactions map[txID]eth.BlockID
}

func NewChannelManager(log log.Logger, channelTimeout uint64) *channelManager {
	return &channelManager{
		log:                   log,
		channelTimeout:        channelTimeout,
		pendingTransactions:   make(map[txID][]byte),
		confirmedTransactions: make(map[txID]eth.BlockID),
	}
}

// Clear clears the entire state of the channel manager.
// It is intended to be used after an L2 reorg.
func (s *channelManager) Clear() {
	s.blocks = s.blocks[:0]
	s.datas = s.datas[:0]
}

func (s *channelManager) TxConfirmed(id txID, inclusionBlock eth.BlockID) {
	if _, ok := s.pendingTransactions[id]; !ok {
		s.log.Info("marked transaction as confirmed despite having no record of it")
		// TODO: This can occur if we clear the channel while there are still pending transactions
		// We need to keep track of stale transactions instead
		return
	}
	delete(s.pendingTransactions, id)
	s.confirmedTransactions[id] = inclusionBlock

	// If this channel timed out, put the pending blocks back into the local saved blocks
	// and then reset this state so it can try to build a new channel.
	if s.pendingChannelIsTimedOut() {
		s.log.Warn("Channel timed out", "id", s.pendingChannel)
		s.blocks = append(s.pendingBlocks, s.blocks...)
		s.clearPendingChannel()
	}
	// If we are done with this channel, record that.
	if s.pendingChannelIsFullySubmitted() {
		s.log.Info("Channel is fully submitted", "id", s.pendingChannel)
		s.clearPendingChannel()
	}
}

// TODO: Create separate "pending" state
func (s *channelManager) clearPendingChannel() {
	s.pendingChannel = derive.ChannelID{}
	s.pendingBlocks = nil
	s.pendingTransactions = make(map[txID][]byte)
	s.confirmedTransactions = make(map[txID]eth.BlockID)
}

func (s *channelManager) pendingChannelIsTimedOut() bool {
	if s.pendingChannel == (derive.ChannelID{}) {
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
	return max-min >= s.channelTimeout
}

func (s *channelManager) pendingChannelIsFullySubmitted() bool {
	if s.pendingChannel == (derive.ChannelID{}) {
		return false // todo: can decide either way here. Nonsensical answer though
	}
	return len(s.pendingTransactions)+len(s.datas) == 0
}

func (s *channelManager) TxFailed(id txID) {
	if data, ok := s.pendingTransactions[id]; ok {
		s.datas = append(s.datas, taggedData{data, id})
		delete(s.pendingTransactions, id)
	} else {
		s.log.Warn("marked transaction as failed despite having no record of it.")
	}
}

// blocksToFrames turns a set of blocks into a set of frames inside a channel.
// It will only create a single channel which contains up to `MAX_RLP_BYTES`. Any
// blocks not added to the channel are returned. It uses the max supplied frame size.
func blocksToFrames(blocks []*types.Block, maxFrameSize uint64) (derive.ChannelID, [][]byte, []*types.Block, error) {
	ch, err := derive.NewChannelOut()
	if err != nil {
		return derive.ChannelID{}, nil, nil, err
	}

	i := 0
	for ; i < len(blocks); i++ {
		if err := ch.AddBlock(blocks[i]); err == derive.ErrTooManyRLPBytes {
			break
		} else if err != nil {
			return derive.ChannelID{}, nil, nil, err
		}
	}
	if err := ch.Close(); err != nil {
		return derive.ChannelID{}, nil, nil, err
	}

	var frames [][]byte
	for {
		var buf bytes.Buffer
		buf.WriteByte(derive.DerivationVersion0)
		err := ch.OutputFrame(&buf, maxFrameSize-1)
		if err != io.EOF && err != nil {
			return derive.ChannelID{}, nil, nil, err
		}
		frames = append(frames, buf.Bytes())
		if err == io.EOF {
			break
		}
	}
	return ch.ID(), frames, blocks[i:], nil
}

// nextTxData pops off s.datas & handles updating the internal state
func (s *channelManager) nextTxData() ([]byte, txID, error) {
	if len(s.datas) != 0 {
		r := s.datas[0]
		s.pendingTransactions[r.id] = r.data
		s.datas = s.datas[1:]
		return r.data, r.id, nil
	} else {
		return nil, txID{}, io.EOF // TODO: not enough data error instead
	}
}

// TxData returns the next tx.data that should be submitted to L1.
// It is very simple & currently ignores the l1Head provided (this will change).
// It may buffer very large channels as well.
func (s *channelManager) TxData(l1Head eth.L1BlockRef) ([]byte, txID, error) {
	s.log.Trace("Requested tx data")

	// Short circuit if there is a pending channel.
	// We either submit the next frame from that channel or
	if s.pendingChannel != (derive.ChannelID{}) {
		return s.nextTxData()
	}
	// If we have no saved blocks, we will not be able to create valid frames
	if len(s.blocks) == 0 {
		return nil, txID{}, io.EOF
	}

	// Select range of blocks
	end := len(s.blocks)
	if end > 100 {
		end = 100
	}
	blocks := s.blocks[:end]
	s.blocks = s.blocks[end:]

	chID, frames, leftOverBlocks, err := blocksToFrames(blocks, 120_000)
	// If the range of blocks serialized to be too large, restore
	// blocks that could not be included inside the channel
	if len(leftOverBlocks) != 0 {
		s.blocks = append(leftOverBlocks, s.blocks...)
	}
	if err != nil {
		return nil, txID{}, err
	}

	var t []taggedData
	for i, data := range frames {
		t = append(t, taggedData{data: data, id: txID{chID: chID, frameNumber: uint16(i)}})
	}

	// Load up pending state. Note: pending transactions is taken care of by nextTxData
	s.datas = t
	s.pendingChannel = chID
	s.pendingBlocks = blocks[:len(leftOverBlocks)]

	return s.nextTxData()

}

// AddL2Block saves an L2 block to the internal state. It returns ErrReorg
// if the block does not extend the last block loaded into the state.
// If no block is already in the channel, the the parent hash check is skipped.
// TODO: Phantom last block b/c if the local state is fully drained we can reorg without realizing it.
func (s *channelManager) AddL2Block(block *types.Block) error {
	if len(s.blocks) > 0 {
		if s.blocks[len(s.blocks)-1].Hash() != block.ParentHash() {
			return ErrReorg
		}
	}
	s.blocks = append(s.blocks, block)
	return nil
}
