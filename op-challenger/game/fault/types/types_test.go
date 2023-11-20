package types

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	oneHash   = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")
	eliteHash = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000001337")
)

func TestNewPreimageOracleData(t *testing.T) {
	t.Run("LocalData", func(t *testing.T) {
		data := NewPreimageOracleData(common.Hash{0x01}, []byte{1, 2, 3}, []byte{4, 5, 6}, 7)
		require.True(t, data.IsLocal)
		require.Equal(t, common.Hash{0x01}, data.LocalContext)
		require.Equal(t, []byte{1, 2, 3}, data.OracleKey)
		require.Equal(t, []byte{4, 5, 6}, data.OracleData)
		require.Equal(t, uint32(7), data.OracleOffset)
	})

	t.Run("GlobalData", func(t *testing.T) {
		data := NewPreimageOracleData(common.Hash{0x01}, []byte{0, 2, 3}, []byte{4, 5, 6}, 7)
		require.False(t, data.IsLocal)
		require.Equal(t, common.Hash{0x01}, data.LocalContext)
		require.Equal(t, []byte{0, 2, 3}, data.OracleKey)
		require.Equal(t, []byte{4, 5, 6}, data.OracleData)
		require.Equal(t, uint32(7), data.OracleOffset)
	})
}

func TestPreimageOracleData_GetLocalContextBigInt(t *testing.T) {
	tests := []struct {
		name     string
		context  common.Hash
		expected *big.Int
	}{
		{name: "LocalContext", context: oneHash, expected: new(big.Int).SetUint64(1)},
		{name: "NoLocalContext", context: NoLocalContext, expected: new(big.Int).SetBytes(NoLocalContext.Bytes())},
		{name: "MultiBytesLocalContext", context: eliteHash, expected: big.NewInt(0x1337)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := NewPreimageOracleData(test.context, []byte{1, 2, 3}, []byte{4, 5, 6}, 7)
			require.Equal(t, test.expected, data.GetLocalContextBigInt())
		})
	}
}

func TestPreimageOracleData_GetOracleOffsetBigInt(t *testing.T) {
	tests := []struct {
		name     string
		offset   uint32
		expected *big.Int
	}{
		{name: "ZeroOffset", offset: 0, expected: new(big.Int).SetUint64(0)},
		{name: "NonZeroOffset", offset: 1, expected: new(big.Int).SetUint64(1)},
		{name: "MaxUint32Offset", offset: 0xffffffff, expected: new(big.Int).SetUint64(0xffffffff)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := NewPreimageOracleData(common.Hash{}, []byte{1, 2, 3}, []byte{4, 5, 6}, test.offset)
			require.Equal(t, test.expected, data.GetOracleOffsetBigInt())
		})
	}
}

func TestIsRootPosition(t *testing.T) {
	tests := []struct {
		name     string
		position Position
		expected bool
	}{
		{
			name:     "ZeroRoot",
			position: NewPositionFromGIndex(big.NewInt(0)),
			expected: true,
		},
		{
			name:     "ValidRoot",
			position: NewPositionFromGIndex(big.NewInt(1)),
			expected: true,
		},
		{
			name:     "NotRoot",
			position: NewPositionFromGIndex(big.NewInt(2)),
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, test.position.IsRootPosition())
		})
	}
}
