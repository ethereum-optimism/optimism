package pure

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestL1InputBlockRef(t *testing.T) {
	input := L1Input{
		Header: &types.Header{
			ParentHash: common.HexToHash("0x99"),
			Number:     big.NewInt(100),
			Time:       1000,
			BaseFee:    big.NewInt(1),
		},
	}
	ref := input.BlockRef()
	require.Equal(t, input.Header.Hash(), ref.Hash)
	require.Equal(t, uint64(100), ref.Number)
	require.Equal(t, uint64(1000), ref.Time)
	require.Equal(t, input.Header.ParentHash, ref.ParentHash)
}

func TestL1InputBlockID(t *testing.T) {
	input := L1Input{
		Header: &types.Header{
			Number: big.NewInt(42),
		},
	}
	id := input.BlockID()
	require.Equal(t, input.Header.Hash(), id.Hash)
	require.Equal(t, uint64(42), id.Number)
}

func TestCursorAdvance(t *testing.T) {
	c := newCursor(eth.L2BlockRef{
		Number:         10,
		Time:           100,
		L1Origin:       eth.BlockID{Number: 5},
		SequenceNumber: 2,
	})
	require.Equal(t, uint64(10), c.Number)
	require.Equal(t, uint64(100), c.Timestamp)
	require.Equal(t, uint64(2), c.SequenceNumber)

	c.advance(102, eth.BlockID{Number: 5}, 3)
	require.Equal(t, uint64(11), c.Number)
	require.Equal(t, uint64(102), c.Timestamp)
	require.Equal(t, uint64(3), c.SequenceNumber)
}

func TestL1InputBlockInfo(t *testing.T) {
	header := &types.Header{
		ParentHash:    common.HexToHash("0x99"),
		Number:        big.NewInt(100),
		Time:          1000,
		MixDigest:     common.HexToHash("0xdd"),
		BaseFee:       big.NewInt(7),
		ExcessBlobGas: ptrTo(uint64(0)),
	}
	input := &L1Input{Header: header}
	info := input.blockInfo()

	require.Equal(t, header.Hash(), info.Hash())
	require.Equal(t, header.ParentHash, info.ParentHash())
	require.Equal(t, uint64(100), info.NumberU64())
	require.Equal(t, uint64(1000), info.Time())
	require.Equal(t, header.MixDigest, info.MixDigest())
	require.Equal(t, header.BaseFee, info.BaseFee())

	// Header and HeaderRLP delegate to the underlying header
	require.Equal(t, header, info.Header())
	_, err := info.HeaderRLP()
	require.NoError(t, err)
}

func TestCursorNeedsEmptyBatch(t *testing.T) {
	cfg := testRollupConfig() // SeqWindowSize = 10

	cursor := l2Cursor{
		Number:    10,
		Timestamp: 100,
		L1Origin:  eth.BlockID{Number: 5},
	}

	// currentL1.Number (15) == cursor.L1Origin.Number (5) + SeqWindowSize (10)
	// Not strictly greater, so window not expired
	require.False(t, cursor.needsEmptyBatch(eth.L1BlockRef{Number: 15}, cfg))

	// currentL1.Number (16) > cursor.L1Origin.Number (5) + SeqWindowSize (10)
	require.True(t, cursor.needsEmptyBatch(eth.L1BlockRef{Number: 16}, cfg))
}
