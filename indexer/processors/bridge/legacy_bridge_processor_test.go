package bridge

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/indexer/bigint"
	"github.com/ethereum-optimism/optimism/indexer/database"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"
)

func TestLegacyWithdrawalAndMessageHash(t *testing.T) {
	// OP Mainnet hashes to also check for
	// - since the message hash doesn't depend on the preset, we only need to check for the withdrawal hash

	expectedWithdrawalHash := common.HexToHash("0x9c0bc28a77328a405f21d51a32d32f038ebf7ce70e377ca48b2cd194ec024f15")
	msg := database.BridgeMessage{
		Nonce:    big.NewInt(100180),
		GasLimit: bigint.Zero,
		Tx: database.Transaction{
			FromAddress: common.HexToAddress("0x4200000000000000000000000000000000000010"),
			ToAddress:   common.HexToAddress("0x99c9fc46f92e8a1c0dec1b1747d010903e884be1"),
			Amount:      bigint.Zero,
			Data:        common.FromHex("0x1532ec34000000000000000000000000094a9009fe93a85658e4b49604fd8177620f8cd8000000000000000000000000094a9009fe93a85658e4b49604fd8177620f8cd8000000000000000000000000000000000000000000000000013abb2a2774ab0000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000000"),
		},
	}

	hash, err := LegacyBridgeMessageWithdrawalHash(10, &msg)
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
