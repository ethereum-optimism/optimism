package pure

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestL1InputBlockRef(t *testing.T) {
	input := L1Input{
		Hash:        common.HexToHash("0xaa"),
		Number:      100,
		Timestamp:   1000,
		ParentHash:  common.HexToHash("0x99"),
		BaseFee:     big.NewInt(1),
		BlobBaseFee: big.NewInt(1),
	}
	ref := input.BlockRef()
	require.Equal(t, input.Hash, ref.Hash)
	require.Equal(t, input.Number, ref.Number)
	require.Equal(t, input.Timestamp, ref.Time)
	require.Equal(t, input.ParentHash, ref.ParentHash)
}

func TestL1InputBlockID(t *testing.T) {
	input := L1Input{
		Hash:   common.HexToHash("0xbb"),
		Number: 42,
	}
	id := input.BlockID()
	require.Equal(t, input.Hash, id.Hash)
	require.Equal(t, input.Number, id.Number)
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

func TestL1InputInfoBlockInfo(t *testing.T) {
	input := &L1Input{
		Hash:        common.HexToHash("0xaa"),
		Number:      100,
		Timestamp:   1000,
		ParentHash:  common.HexToHash("0x99"),
		MixDigest:   common.HexToHash("0xdd"),
		BaseFee:     big.NewInt(7),
		BlobBaseFee: big.NewInt(3),
	}
	info := &l1InputInfo{input}

	require.Equal(t, input.Hash, info.Hash())
	require.Equal(t, input.ParentHash, info.ParentHash())
	require.Equal(t, input.Number, info.NumberU64())
	require.Equal(t, input.Timestamp, info.Time())
	require.Equal(t, input.MixDigest, info.MixDigest())
	require.Equal(t, input.BaseFee, info.BaseFee())
	require.Equal(t, input.BlobBaseFee, info.BlobBaseFee(nil))

	// Zero-value methods
	require.Equal(t, common.Address{}, info.Coinbase())
	require.Equal(t, common.Hash{}, info.Root())
	require.Equal(t, common.Hash{}, info.ReceiptHash())
	require.Equal(t, uint64(0), info.GasUsed())
	require.Equal(t, uint64(0), info.GasLimit())
	require.Nil(t, info.ParentBeaconRoot())
	require.Nil(t, info.WithdrawalsRoot())

	// ExcessBlobGas is non-nil when BlobBaseFee is set
	require.NotNil(t, info.ExcessBlobGas())

	// Header returns a valid header
	h := info.Header()
	require.Equal(t, input.ParentHash, h.ParentHash)
	require.Equal(t, input.Number, h.Number.Uint64())

	// HeaderRLP doesn't error
	_, err := info.HeaderRLP()
	require.NoError(t, err)
}

func TestL1InputInfoNilBlobBaseFee(t *testing.T) {
	input := &L1Input{
		Hash:    common.HexToHash("0xaa"),
		Number:  100,
		BaseFee: big.NewInt(7),
	}
	info := &l1InputInfo{input}
	require.Nil(t, info.BlobBaseFee(nil))
	require.Nil(t, info.ExcessBlobGas())
}
