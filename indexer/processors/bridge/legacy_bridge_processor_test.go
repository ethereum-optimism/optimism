package bridge

import (
	"testing"

	"github.com/ethereum-optimism/optimism/indexer/bigint"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/processors/contracts"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/stretchr/testify/require"
)

func TestLegacyWithdrawalAndMessageHash(t *testing.T) {
	// Pre-Bedrock OP-Goerli withdrawal that was proven post-bedrock
	// - L1 proven withdrawal tx: 0xa8853a3532f40052385602c66512e438bc1e3736d3cb7abde359f5b9377441c7
	value := bigint.Zero
	expectedWithdrawalHash := common.HexToHash("0xae99d25df3e38730f6ee6588733417e20a131923b84870be6aedb4f863b6302d")

	// Ensure the L2 Tx which correlates with the above proven withdrawal results in the same computed withdrawal hash
	//  - L2 withdrawal tx: 0x254d9c28add020404142f840ed794cea51f86c0f0a737e3e7bdd7e1e4550962e
	abi, err := bindings.CrossDomainMessengerMetaData.GetAbi()
	require.NoError(t, err)

	var sentMessage bindings.CrossDomainMessengerSentMessage
	sentMessageEvent := abi.Events["SentMessage"]
	logData := common.FromHex("0x0000000000000000000000004200000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000186a0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000e4a9f9e67500000000000000000000000007865c6e87b9f70255377e024ace6630c1eaa37f0000000000000000000000003b8e53b3ab8e01fb57d0c9e893bc4d655aa67d84000000000000000000000000b91882244f7f82540f2941a759724523c7b9a166000000000000000000000000b91882244f7f82540f2941a759724523c7b9a166000000000000000000000000000000000000000000000000000000000000271000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	require.NoError(t, contracts.UnpackLog(&sentMessage, &types.Log{Data: logData, Topics: []common.Hash{sentMessageEvent.ID, common.HexToHash("0x000000000000000000000000636af16bf2f682dd3109e60102b8e1a089fedaa8")}}, sentMessageEvent.Name, abi))

	// timestamp and message hash are filled in fields. not core to the event
	msg := database.BridgeMessage{
		Nonce:    sentMessage.MessageNonce,
		GasLimit: sentMessage.GasLimit,
		Tx:       database.Transaction{FromAddress: sentMessage.Sender, ToAddress: sentMessage.Target, Amount: value, Data: sentMessage.Message},
	}

	hash, err := legacyBridgeMessageWithdrawalHash(420, &msg)
	require.NoError(t, err)
	require.Equal(t, expectedWithdrawalHash, hash)

	// Ensure the relayed message hash (v1) matches
	expectedMessageHash := common.HexToHash("0xcb16ecc1967f5d7aed909349a4351d28fbb396429ef7faf1c9d2a670e3ca906f")
	v1MessageHash, err := legacyBridgeMessageV1MessageHash(&msg)
	require.NoError(t, err)
	require.Equal(t, expectedMessageHash, v1MessageHash)
}
