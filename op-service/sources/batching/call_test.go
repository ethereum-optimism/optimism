package batching

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCallResult_GetValues(t *testing.T) {
	tests := []struct {
		name     string
		getter   func(result *CallResult, i int) interface{}
		expected interface{}
	}{
		{
			name: "GetUint8",
			getter: func(result *CallResult, i int) interface{} {
				return result.GetUint8(i)
			},
			expected: uint8(12),
		},
		{
			name: "GetUint32",
			getter: func(result *CallResult, i int) interface{} {
				return result.GetUint32(i)
			},
			expected: uint32(12346),
		},
		{
			name: "GetUint64",
			getter: func(result *CallResult, i int) interface{} {
				return result.GetUint64(i)
			},
			expected: uint64(12346),
		},
		{
			name: "GetBool",
			getter: func(result *CallResult, i int) interface{} {
				return result.GetBool(i)
			},
			expected: true,
		},
		{
			name: "GetAddress",
			getter: func(result *CallResult, i int) interface{} {
				return result.GetAddress(i)
			},
			expected: ([20]byte)(common.Address{0xaa, 0xbb, 0xcc}),
		},
		{
			name: "GetHash",
			getter: func(result *CallResult, i int) interface{} {
				return result.GetHash(i)
			},
			expected: ([32]byte)(common.Hash{0xaa, 0xbb, 0xcc}),
		},
		{
			name: "GetBytes",
			getter: func(result *CallResult, i int) interface{} {
				return result.GetBytes(i)
			},
			expected: []byte{0xaa, 0xbb, 0xcc},
		},
		{
			name: "GetBytes32",
			getter: func(result *CallResult, i int) interface{} {
				return result.GetBytes32(i)
			},
			expected: [32]byte{0xaa, 0xbb, 0xcc},
		},
		{
			name: "GetBytes32Slice",
			getter: func(result *CallResult, i int) interface{} {
				return result.GetBytes32Slice(i)
			},
			expected: [][32]byte{{0xaa, 0xbb, 0xcc}, {0xdd, 0xee, 0xff}, {0x11, 0x22, 0x33}},
		},
		{
			name: "GetBigInt",
			getter: func(result *CallResult, i int) interface{} {
				return result.GetBigInt(i)
			},
			expected: big.NewInt(2398423),
		},
		{
			name: "GetStruct",
			getter: func(result *CallResult, i int) interface{} {
				out := struct {
					a *big.Int
					b common.Hash
				}{}
				result.GetStruct(i, &out)
				return out
			},
			expected: struct {
				a *big.Int
				b common.Hash
			}{
				a: big.NewInt(6),
				b: common.Hash{0xee},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			callResult := &CallResult{[]interface{}{nil, 0, "abc", test.expected, "xyz", 3, nil}}
			actual := test.getter(callResult, 3)
			require.EqualValues(t, test.expected, actual)
		})
	}
}
