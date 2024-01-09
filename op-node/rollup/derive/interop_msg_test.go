package derive

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestInteropMessageDeposit(t *testing.T) {
	interopMsgs := []InteropMessages{
		{
			SourceInfo: InteropMessageSourceInfo{ToBlockNumber: big.NewInt(1)},
			Messages:   []InteropMessage{{Value: big.NewInt(1), GasLimit: big.NewInt(0), Data: []byte{}, MessageRoot: common.Hash{byte('a')}}},
		},
		{
			SourceInfo: InteropMessageSourceInfo{ToBlockNumber: big.NewInt(1)},
			Messages:   []InteropMessage{{Value: big.NewInt(1), GasLimit: big.NewInt(0), Data: []byte{}, MessageRoot: common.Hash{byte('b')}}},
		},
	}

	deposit, err := InteropMessagesDeposit(interopMsgs)
	require.NoError(t, err)
	require.Equal(t, CrossL2InboxDepositorAddr, deposit.From)
	require.Equal(t, CrossL2InboxAddr, *deposit.To)
	require.Equal(t, uint64(2), deposit.Mint.Uint64())
	require.Equal(t, uint64(2), deposit.Value.Uint64())
	require.Equal(t, uint64(RegolithSystemTxGas), deposit.Gas)

	abi, err := bindings.CrossL2InboxMetaData.GetAbi()
	require.NoError(t, err)

	inputs, err := abi.Methods["deliverMessages"].Inputs.Unpack(deposit.Data)
	require.NoError(t, err)

	// easiest way to convert anonymous interface is to marshal/unmarshal into json
	bytes, err := json.Marshal(inputs[0])
	require.NoError(t, err)

	var inboxMsgs []bindings.InboxMessages
	require.NoError(t, json.Unmarshal(bytes, &inboxMsgs))
	require.Len(t, inboxMsgs, 2)
	require.Equal(t, common.Hash{byte('a')}, common.Hash(inboxMsgs[0].MessageRoots[0]))
	require.Equal(t, common.Hash{byte('b')}, common.Hash(inboxMsgs[1].MessageRoots[0]))
}
