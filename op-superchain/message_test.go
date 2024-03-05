package superchain

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	BytesType, _   = abi.NewType("bytes", "", nil)
	AddressType, _ = abi.NewType("address", "", nil)
	MsgIdType, _   = abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "origin", Type: "address"},
		{Name: "blockNumber", Type: "uint256"},

		// for simplicity use uint64 since these go fields as parameterized
		// this way. makes no difference to the abi encoding of the tuple
		{Name: "logIndex", Type: "uint64"},
		{Name: "timestamp", Type: "uint64"},

		{Name: "chainId", Type: "uint256"},
	})

	ExecuteMessageMethod = abi.NewMethod(
		"executeMessage", // name
		"executeMessage", // raw name
		abi.Function,     // fn type
		"",               // mutability
		false,            // isConst
		false,            // isPayable
		abi.Arguments{{Type: MsgIdType}, {Type: AddressType}, {Type: BytesType}}, // inputs
		abi.Arguments{}, // ouputs
	)
)

func TestParseInboxExecuteMessageUnpacking(t *testing.T) {
	msgId := MessageIdentifier{common.HexToAddress("0xa"), big.NewInt(10), 1, 1, big.NewInt(10)}
	msgTarget := common.HexToAddress("0xb")

	calldata, err := ExecuteMessageMethod.Inputs.Pack(msgId, msgTarget, []byte{byte(1)})
	require.NoError(t, err)

	target, id, msg, err := ParseInboxExecuteMessageTxData(append(inboxExecuteMessageBytes4, calldata...))
	require.NoError(t, err)

	// target
	require.Equal(t, target, msgTarget)

	// id
	require.Equal(t, msgId.Origin, id.Origin)
	require.Equal(t, msgId.BlockNumber.Uint64(), id.BlockNumber.Uint64())
	require.Equal(t, msgId.LogIndex, id.LogIndex)
	require.Equal(t, msgId.Timestamp, id.Timestamp)
	require.Equal(t, msgId.ChainId.Uint64(), id.ChainId.Uint64())

	// msg
	require.Len(t, msg, 1)
	require.Equal(t, msg[0], byte(1))
}
