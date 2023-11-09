package batching

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestContractCall_ToCallArgs(t *testing.T) {
	addr := common.Address{0xbd}
	testAbi, err := bindings.ERC20MetaData.GetAbi()
	require.NoError(t, err)
	call := NewContractCall(testAbi, addr, "approve", common.Address{0xcc}, big.NewInt(1234444))
	args, err := call.ToCallArgs()
	require.NoError(t, err)
	argMap, ok := args.(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, argMap["from"], common.Address{})
	require.Equal(t, argMap["to"], &addr)
	expectedData, err := call.Pack()
	require.NoError(t, err)
	require.Equal(t, argMap["input"], hexutil.Bytes(expectedData))

	require.NotContains(t, argMap, "value")
	require.NotContains(t, argMap, "gas")
	require.NotContains(t, argMap, "gasPrice")
}

func TestContractCall_ToTxCandidate(t *testing.T) {
	addr := common.Address{0xbd}
	testAbi, err := bindings.ERC20MetaData.GetAbi()
	require.NoError(t, err)
	call := NewContractCall(testAbi, addr, "approve", common.Address{0xcc}, big.NewInt(1234444))
	candidate, err := call.ToTxCandidate()
	require.NoError(t, err)
	require.Equal(t, candidate.To, &addr)
	expectedData, err := call.Pack()
	require.NoError(t, err)
	require.Equal(t, candidate.TxData, expectedData)

	require.Nil(t, candidate.Value)
	require.Zero(t, candidate.GasLimit)
}

func TestContractCall_Pack(t *testing.T) {
	addr := common.Address{0xbd}
	testAbi, err := bindings.ERC20MetaData.GetAbi()
	require.NoError(t, err)
	sender := common.Address{0xcc}
	amount := big.NewInt(1234444)
	call := NewContractCall(testAbi, addr, "approve", sender, amount)
	actual, err := call.Pack()
	require.NoError(t, err)

	expected, err := testAbi.Pack("approve", sender, amount)
	require.NoError(t, err)
	require.Equal(t, actual, expected)
}

func TestContractCall_PackInvalid(t *testing.T) {
	addr := common.Address{0xbd}
	testAbi, err := bindings.ERC20MetaData.GetAbi()
	require.NoError(t, err)
	// Second arg should be a *big.Int so packing should fail
	call := NewContractCall(testAbi, addr, "approve", common.Address{0xcc}, uint32(123))
	_, err = call.Pack()
	require.Error(t, err)
}

func TestContractCall_Unpack(t *testing.T) {
	addr := common.Address{0xbd}
	testAbi, err := bindings.ERC20MetaData.GetAbi()
	require.NoError(t, err)
	call := NewContractCall(testAbi, addr, "balanceOf", common.Address{0xcc})
	outputs := testAbi.Methods["balanceOf"].Outputs
	expected := big.NewInt(1234)
	packed, err := outputs.Pack(expected)
	require.NoError(t, err)

	unpacked, err := call.Unpack(packed)
	require.NoError(t, err)
	require.Equal(t, unpacked.GetBigInt(0), expected)
}

func TestContractCall_UnpackInvalid(t *testing.T) {
	addr := common.Address{0xbd}
	testAbi, err := bindings.ERC20MetaData.GetAbi()
	require.NoError(t, err)
	call := NewContractCall(testAbi, addr, "balanceOf", common.Address{0xcc})

	// Input data is the wrong format and won't unpack successfully
	inputPacked, err := call.Pack()
	require.NoError(t, err)

	_, err = call.Unpack(inputPacked)
	require.Error(t, err)
}

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
