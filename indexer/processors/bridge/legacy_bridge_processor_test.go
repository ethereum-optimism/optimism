package bridge

import (
	"math/big"
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

	hash, err := LegacyBridgeMessageWithdrawalHash(420, &msg)
	require.NoError(t, err)
	require.Equal(t, expectedWithdrawalHash, hash)

	// Ensure the relayed message hash (v1) matches
	expectedMessageHash := common.HexToHash("0xcb16ecc1967f5d7aed909349a4351d28fbb396429ef7faf1c9d2a670e3ca906f")
	v1MessageHash, err := LegacyBridgeMessageV1MessageHash(&msg)
	require.NoError(t, err)
	require.Equal(t, expectedMessageHash, v1MessageHash)

	// OP Mainnet hashes to also check for
	// - since the message hash doesn't depend on the preset, we only need to check for the withdrawal hash

	expectedWithdrawalHash = common.HexToHash("0x9c0bc28a77328a405f21d51a32d32f038ebf7ce70e377ca48b2cd194ec024f15")
	msg = database.BridgeMessage{
		Nonce:    big.NewInt(100180),
		GasLimit: bigint.Zero,
		Tx: database.Transaction{
			FromAddress: common.HexToAddress("0x4200000000000000000000000000000000000010"),
			ToAddress:   common.HexToAddress("0x99c9fc46f92e8a1c0dec1b1747d010903e884be1"),
			Amount:      bigint.Zero,
			Data:        common.FromHex("0x1532ec34000000000000000000000000094a9009fe93a85658e4b49604fd8177620f8cd8000000000000000000000000094a9009fe93a85658e4b49604fd8177620f8cd8000000000000000000000000000000000000000000000000013abb2a2774ab0000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000000"),
		},
	}

	hash, err = LegacyBridgeMessageWithdrawalHash(10, &msg)
	require.NoError(t, err)
	require.Equal(t, expectedWithdrawalHash, hash)

	expectedWithdrawalHash = common.HexToHash("0xeb1dd5ead967ad6860d64407413f86e50330ab123ca9adf2768145524c3f5323")
	msg = database.BridgeMessage{
		Nonce:    big.NewInt(100618),
		GasLimit: bigint.Zero,
		Tx: database.Transaction{
			FromAddress: common.HexToAddress("0x4200000000000000000000000000000000000010"),
			ToAddress:   common.HexToAddress("0x99c9fc46f92e8a1c0dec1b1747d010903e884be1"),
			Amount:      bigint.Zero,
			Data:        common.FromHex("0xa9f9e67500000000000000000000000028e1de268616a6ba0de59717ac5547589e6bb1180000000000000000000000003ef241d9ae02f2253d8a1bf0b35d68eab9925b400000000000000000000000003e579180cf01f0e2abf6ff4d566b7891fbf9b8680000000000000000000000003e579180cf01f0e2abf6ff4d566b7891fbf9b868000000000000000000000000000000000000000000000000000000174876e80000000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000000"),
		},
	}

	hash, err = LegacyBridgeMessageWithdrawalHash(10, &msg)
	require.NoError(t, err)
	require.Equal(t, expectedWithdrawalHash, hash)
}
