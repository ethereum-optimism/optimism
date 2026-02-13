package batching

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestContractCall_ToCallArgs(t *testing.T) {
	addr := common.Address{0xbd}
	testAbi, err := test.ERC20MetaData.GetAbi()
	require.NoError(t, err)
	call := NewContractCall(testAbi, addr, "approve", common.Address{0xcc}, big.NewInt(1234444))
	call.From = common.Address{0xab}
	args, err := call.ToCallArgs()
	require.NoError(t, err)
	argMap, ok := args.(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, argMap["from"], call.From)
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
	testAbi, err := test.ERC20MetaData.GetAbi()
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
	testAbi, err := test.ERC20MetaData.GetAbi()
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
	testAbi, err := test.ERC20MetaData.GetAbi()
	require.NoError(t, err)
	// Second arg should be a *big.Int so packing should fail
	call := NewContractCall(testAbi, addr, "approve", common.Address{0xcc}, uint32(123))
	_, err = call.Pack()
	require.Error(t, err)
}

func TestContractCall_Unpack(t *testing.T) {
	addr := common.Address{0xbd}
	testAbi, err := test.ERC20MetaData.GetAbi()
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
	testAbi, err := test.ERC20MetaData.GetAbi()
	require.NoError(t, err)
	call := NewContractCall(testAbi, addr, "balanceOf", common.Address{0xcc})

	// Input data is the wrong format and won't unpack successfully
	inputPacked, err := call.Pack()
	require.NoError(t, err)

	_, err = call.Unpack(inputPacked)
	require.Error(t, err)
}
