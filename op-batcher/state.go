package op_batcher

import (
	"bytes"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/core/types"
)

type batchState struct {
	// All blocks since the last request for new tx data
	blocks []*types.Block
	datas  [][]byte
}

// TxData returns the next tx.data that should be submitted to L1.
// It is very simple & currently ignores the l1Head provided (this will change).
// It may buffer very large channels as well.
func (s *batchState) TxData(l1Head eth.L1BlockRef) ([]byte, error) {
	// Note: l1Head is not actually used in this function.

	// Return a pre-existing frame if we have it.
	if len(s.datas) != 0 {
		r := s.datas[0]
		s.datas = s.datas[1:]
		return r, nil
	}

	ch, err := derive.NewChannelOut()
	if err != nil {
		return nil, err
	}
	// TODO: use peek/pop paradigm here instead of manually slicing
	i := 0
	for ; i < len(s.blocks); i++ {
		if err := ch.AddBlock(s.blocks[i]); err == derive.ErrTooManyRLPBytes {
			break
		} else if err != nil {
			return nil, err
		}
		// TODO: limit the RLP size of the channel to be lower than the limit to enable
		// channels to be fully submitted on time.
	}
	if err := ch.Close(); err != nil {
		return nil, err
	}

	var t [][]byte
	for {
		var buf bytes.Buffer
		buf.WriteByte(derive.DerivationVersion0)
		err := ch.OutputFrame(&buf, 120_000)
		if err != io.EOF && err != nil {
			return nil, err
		}

		t = append(t, buf.Bytes())
		if err == io.EOF {
			break
		}
	}

	s.datas = append(s.datas, t...)
	// Say i = 0, 1 are added to the channel, but i = 2 returns ErrTooManyRLPBytes. i remains 2 & is inclusive, so this works.
	// Say all blocks are added, i will be len(blocks) after exiting the loop (but never inside the loop).
	s.blocks = s.blocks[i:]

	if len(s.datas) == 0 {
		return nil, io.EOF // TODO: not enough data error instead
	}

	r := s.datas[0]
	s.datas = s.datas[1:]
	return r, nil
}

// TODO: Continuity check here?
// Invariants about what's on L1?
func (s *batchState) AddL2Block(block *types.Block) error {
	s.blocks = append(s.blocks, block)
	return nil
}

/*
type ChannelManager struct {
	channels set[channels]
  // potential for tx/timeout index into channels
}

// ChannelManager Functions
NewL2Block(block ...)
TxConfirmed(txId ??)
TxDatas(L1Head ??) (txDatas, ids, error)
*/
