package superchain

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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

func TestMessageLogCheck(t *testing.T) {
	origin := common.HexToAddress("0xA")
	blockNum := big.NewInt(1)
	logIndex := uint64(1)

	// Specifiy the fields set in the log
	id := MessageIdentifier{Origin: origin, BlockNumber: blockNum, LogIndex: logIndex}

	log := &types.Log{
		Topics:      []common.Hash{common.HexToHash("0xA"), common.HexToHash("0xB")},
		Data:        []byte{byte(1), byte(2), byte(3)},
		BlockNumber: blockNum.Uint64(),
		Address:     origin,
		Index:       uint(logIndex),
	}

	payload := MessagePayloadBytes(log)
	require.NoError(t, MessageLogCheck(id, payload, log))

	// origin mismatch
	id.Origin = common.HexToAddress("0xB")
	require.Error(t, MessageLogCheck(id, payload, log))
	id.Origin = origin
	require.NoError(t, MessageLogCheck(id, payload, log))

	// block number mismatch
	id.BlockNumber = big.NewInt(2)
	require.Error(t, MessageLogCheck(id, payload, log))
	id.BlockNumber = blockNum
	require.NoError(t, MessageLogCheck(id, payload, log))

	// log index mismatch
	id.LogIndex = 2
	require.Error(t, MessageLogCheck(id, payload, log))
	id.LogIndex = logIndex

	// payload mismatch
	require.Error(t, MessageLogCheck(id, payload[:1], log))
	require.NoError(t, MessageLogCheck(id, payload, log))
}

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
