package batcher

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

const piRenderGenesis = uint64(900)

// piRenderingCfg is the RENDERING's rollup config: same timing as the private chain (the design's
// block-for-block correspondence), with a real genesis block so the genesis edge case is testable.
func piRenderingCfg() *rollup.Config {
	cfg := piRollupCfg()
	cfg.Genesis.L2 = eth.BlockID{Hash: common.Hash{0xaa}, Number: piRenderGenesis}
	cfg.Genesis.L1 = eth.BlockID{Hash: refOf(headerChain(40)[3]).Hash, Number: 3}
	return cfg
}

// piRenderedBlock is a rendering block as the range source sees one: an identity, and nothing else.
// Since origin-copy there is no attributes deposit to decode here — the rendering block's origin and
// sequence number are its private block's own, so nothing carries across a range boundary.
func piRenderedBlock(number uint64) *RenderingBlock {
	return &RenderingBlock{
		Hash:   common.BigToHash(new(big.Int).SetUint64(0xbeef0000 + number)),
		Number: number,
	}
}

type fakeFollower struct {
	blocks map[uint64]*RenderingBlock
	nonces map[uint64]uint64
}

func (f *fakeFollower) BlockByNumber(_ context.Context, number uint64) (*RenderingBlock, error) {
	b, ok := f.blocks[number]
	if !ok {
		return nil, fmt.Errorf("the rendering has no block %d yet", number)
	}
	return b, nil
}

func (f *fakeFollower) NonceAt(_ context.Context, _ common.Address, number uint64) (uint64, error) {
	n, ok := f.nonces[number]
	if !ok {
		return 0, fmt.Errorf("no state at %d", number)
	}
	return n, nil
}

// headerChain builds a parent-linked chain of real L1 headers.
func headerChain(n int) []*types.Header {
	out := make([]*types.Header, 0, n)
	var parent common.Hash
	for i := range n {
		h := &types.Header{
			ParentHash: parent,
			Number:     new(big.Int).SetUint64(uint64(i)),
			Time:       piL1Genesis + uint64(i)*12,
			Extra:      []byte{byte(i)},
		}
		out = append(out, h)
		parent = h.Hash()
	}
	return out
}

// refOf is the L1BlockRef a header stands for.
func refOf(h *types.Header) eth.L1BlockRef {
	return eth.L1BlockRef{Hash: h.Hash(), Number: h.Number.Uint64(), ParentHash: h.ParentHash, Time: h.Time}
}

func piSource(t *testing.T, follower *fakeFollower) *privateInteropRangeSource {
	t.Helper()
	src, err := NewPrivateInteropRangeSource(PrivateInteropRangeSourceConfig{
		Log:             testlog.Logger(t, log.LevelError),
		RenderingRollup: piRenderingCfg(),
		Rendering:       follower,
		Batcher:         piOtherAddr,
	})
	require.NoError(t, err)
	return src.(*privateInteropRangeSource)
}

// TestRangeStartReadsTheRenderingsOwnBookkeeping: both fields of a range's start are facts about the
// rendering's previous block, and each is read rather than guessed.
//
// Since origin-copy there are two fields rather than four. The origin and sequence number used to be
// read out of that block's attributes deposit and carried across the range boundary; now a rendering
// block's origin is its private block's own, so there is nothing to continue.
func TestRangeStartReadsTheRenderingsOwnBookkeeping(t *testing.T) {
	prev := piRenderedBlock(950)
	f := &fakeFollower{
		blocks: map[uint64]*RenderingBlock{950: prev},
		nonces: map[uint64]uint64{950: 42},
	}
	src := piSource(t, f)

	got, err := src.RangeStart(context.Background(), 951)
	require.NoError(t, err)
	require.Equal(t, prev.Hash, got.PrevTerminalRenderingHash, "the span's parent check")
	require.Equal(t, uint64(42), got.StartNonce, "the batcher's nonce comes from the chain, not from memory")
}

// TestRangeStartAtTheRenderingsFirstRange: the rendering's genesis block has no attributes deposit,
// which used to be a special case here. Origin-copy removed the reason to read one at all, so the
// only rule left is that a range cannot open at or below genesis.
func TestRangeStartAtTheRenderingsFirstRange(t *testing.T) {
	genesis := &RenderingBlock{Hash: common.Hash{0xaa}, Number: piRenderGenesis}
	f := &fakeFollower{
		blocks: map[uint64]*RenderingBlock{piRenderGenesis: genesis},
		nonces: map[uint64]uint64{piRenderGenesis: 0},
	}
	src := piSource(t, f)

	got, err := src.RangeStart(context.Background(), piRenderGenesis+1)
	require.NoError(t, err)
	require.Equal(t, genesis.Hash, got.PrevTerminalRenderingHash)

	// And a range cannot open at or below genesis: there is nothing for it to continue from.
	_, err = src.RangeStart(context.Background(), piRenderGenesis)
	require.ErrorContains(t, err, "the rendering's genesis is block")
}

// TestRangeStartWaitsForTheRendering: an underived predecessor is a WAIT. Failing here is the
// design's correct behaviour — a guessed parent check is a batch every verifier drops.
func TestRangeStartWaitsForTheRendering(t *testing.T) {
	src := piSource(t, &fakeFollower{blocks: map[uint64]*RenderingBlock{}})
	_, err := src.RangeStart(context.Background(), 951)
	require.ErrorContains(t, err, "reading the rendering's block 950")
}
