package eth

import (
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestChainID_String(t *testing.T) {
	tests := []struct {
		input    ChainID
		expected string
	}{
		{ChainIDFromUInt64(0), "0"},
		{ChainIDFromUInt64(1), "1"},
		{ChainIDFromUInt64(871975192374), "871975192374"},
		{ChainIDFromUInt64(math.MaxInt64), "9223372036854775807"},
		{ChainID(*uint256.NewInt(math.MaxUint64)), "18446744073709551615"},
		{ChainID(*uint256.MustFromDecimal("1844674407370955161618446744073709551616")), "1844674407370955161618446744073709551616"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.expected, func(t *testing.T) {
			t.Run("String", func(t *testing.T) {
				require.Equal(t, test.expected, test.input.String())
			})
			t.Run("MarshalText", func(t *testing.T) {
				data, err := test.input.MarshalText()
				require.NoError(t, err)
				require.Equal(t, test.expected, string(data))
			})
			t.Run("UnmarshalText", func(t *testing.T) {
				var id ChainID
				require.NoError(t, id.UnmarshalText([]byte(test.expected)))
				require.Equal(t, test.input, id)
			})
		})
	}
}

func TestSortChainIDs(t *testing.T) {
	ids := []ChainID{
		ChainIDFromUInt64(123),
		ChainIDFromUInt64(16),
		ChainIDFromBytes32([32]byte{0: 1}),
		ChainIDFromUInt64(1),
		ChainIDFromUInt64(2),
		ChainIDFromUInt64(0xdeadbeef),
	}
	expected := []ChainID{
		ChainIDFromUInt64(1),
		ChainIDFromUInt64(2),
		ChainIDFromUInt64(16),
		ChainIDFromUInt64(123),
		ChainIDFromUInt64(0xdeadbeef),
		ChainIDFromBytes32([32]byte{0: 1}),
	}
	require.NotEqual(t, expected, ids)
	SortChainID(ids)
	require.Equal(t, expected, ids)
}

func TestChainID_FromString(t *testing.T) {
	hash := "0x23b00f3aae2634f1b35057cca336e1950473a1e037d0b405302d988f360f8268"
	var h [32]byte
	dec := hexutil.MustDecode(hash)
	copy(h[:], dec)

	tests := []struct {
		input    string
		expected ChainID
	}{
		{"0", ChainIDFromUInt64(0)},
		{"1", ChainIDFromUInt64(1)},
		{"871975192374", ChainIDFromUInt64(871975192374)},
		{"9223372036854775807", ChainIDFromUInt64(math.MaxInt64)},
		{"18446744073709551615", ChainID(*uint256.NewInt(math.MaxUint64))},
		{"0x1234", ChainIDFromUInt64(0x1234)},
		{hash, ChainIDFromBytes32(h)},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			id, err := ChainIDFromString(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.expected, id)
		})
	}
}
