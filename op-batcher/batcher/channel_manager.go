package batcher

import (
	"bytes"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/core/types"
)

type txID struct {
	chID        derive.ChannelID
	frameNumber uint16
}

type taggedData struct {
	data []byte
	id   txID
}

type channelManager struct {
	// All blocks since the last request for new tx data
	blocks []*types.Block
	datas  []taggedData
}

// func (s *channelManager) TxConfirmed(id txID, inclusionBlock eth.BlockID) {
// 	// todo: implement
// }

// TxData returns the next tx.data that should be submitted to L1.
// It is very simple & currently ignores the l1Head provided (this will change).
// It may buffer very large channels as well.
func (s *channelManager) TxData(l1Head eth.L1BlockRef) ([]byte, txID, error) {
	// Note: l1Head is not actually used in this function.

	// Return a pre-existing frame if we have it.
	if len(s.datas) != 0 {
		r := s.datas[0]
		s.datas = s.datas[1:]
		return r.data, r.id, nil
	}

	// Also return io.EOF if we cannot create a channel
	if len(s.blocks) == 0 {
		return nil, txID{}, io.EOF
	}

	// Add all pending blocks to a channel
	ch, err := derive.NewChannelOut()
	if err != nil {
		return nil, txID{}, err
	}
	// TODO: use peek/pop paradigm here instead of manually slicing
	i := 0
	// Cap length at 100 blocks
	l := len(s.blocks)
	if l > 100 {
		l = 100
	}
	for ; i < l; i++ {
		if err := ch.AddBlock(s.blocks[i]); err == derive.ErrTooManyRLPBytes {
			break
		} else if err != nil {
			return nil, txID{}, err
		}
		// TODO: limit the RLP size of the channel to be lower than the limit to enable
		// channels to be fully submitted on time.
	}
	if err := ch.Close(); err != nil {
		return nil, txID{}, err
	}

	var t []taggedData
	frameNumber := uint16(0)
	for {
		var buf bytes.Buffer
		buf.WriteByte(derive.DerivationVersion0)
		err := ch.OutputFrame(&buf, 120_000)
		if err != io.EOF && err != nil {
			return nil, txID{}, err
		}

		t = append(t, taggedData{
			data: buf.Bytes(),
			id:   txID{ch.ID(), frameNumber},
		})
		frameNumber += 1
		if err == io.EOF {
			break
		}
	}

	s.datas = append(s.datas, t...)
	// Say i = 0, 1 are added to the channel, but i = 2 returns ErrTooManyRLPBytes. i remains 2 & is inclusive, so this works.
	// Say all blocks are added, i will be len(blocks) after exiting the loop (but never inside the loop).
	s.blocks = s.blocks[i:]

	if len(s.datas) == 0 {
		return nil, txID{}, io.EOF // TODO: not enough data error instead
	}

	r := s.datas[0]
	s.datas = s.datas[1:]
	return r.data, r.id, nil
}

// TODO: Continuity check here?
// Invariants about what's on L1?
func (s *channelManager) AddL2Block(block *types.Block) error {
	s.blocks = append(s.blocks, block)
	return nil
}
