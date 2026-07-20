package crossdomain

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// testWireReceipt mirrors the stored receipt wire format so tests can encode
// receipts with either historical log encoding.
type testWireReceipt[L any] struct {
	PostStateOrStatus []byte
	CumulativeGasUsed uint64
	Logs              []L
	L1GasUsed         *big.Int
	L1GasPrice        *big.Int
	L1Fee             *big.Int
	FeeScalar         string
}

func testLog() *types.Log {
	return &types.Log{
		Address: common.Address{0x42},
		Topics:  []common.Hash{{0x01}, {0x02}},
		Data:    []byte{0xde, 0xad, 0xbe, 0xef},
	}
}

func encodeTestReceipt[L any](t *testing.T, logs []L) []byte {
	blob, err := rlp.EncodeToBytes(&testWireReceipt[L]{
		PostStateOrStatus: []byte{0x01},
		CumulativeGasUsed: 21000,
		Logs:              logs,
		L1GasUsed:         big.NewInt(100),
		L1GasPrice:        big.NewInt(2),
		L1Fee:             big.NewInt(200),
		FeeScalar:         "1.0",
	})
	require.NoError(t, err)
	return blob
}

func TestLegacyReceiptDecodeStoredLogFormats(t *testing.T) {
	want := testLog()

	t.Run("reduced", func(t *testing.T) {
		type reducedLog struct {
			Address common.Address
			Topics  []common.Hash
			Data    []byte
		}
		blob := encodeTestReceipt(t, []reducedLog{{
			Address: want.Address,
			Topics:  want.Topics,
			Data:    want.Data,
		}})

		var receipt LegacyReceipt
		require.NoError(t, rlp.DecodeBytes(blob, &receipt))
		require.Equal(t, []*types.Log{want}, receipt.Logs)
	})

	t.Run("legacy", func(t *testing.T) {
		blob := encodeTestReceipt(t, []legacyRlpStorageLog{{
			Address:     want.Address,
			Topics:      want.Topics,
			Data:        want.Data,
			BlockNumber: 1234,
			TxHash:      common.Hash{0xaa},
			TxIndex:     1,
			BlockHash:   common.Hash{0xbb},
			Index:       2,
		}})

		var receipt LegacyReceipt
		require.NoError(t, rlp.DecodeBytes(blob, &receipt))
		require.Equal(t, []*types.Log{want}, receipt.Logs)
	})
}
