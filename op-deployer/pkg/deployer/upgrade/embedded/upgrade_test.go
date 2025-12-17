package embedded

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestUpgradeOPChainInput_UpgradeInput(t *testing.T) {
	input := &UpgradeOPChainInput{
		Prank: common.Address{0xaa},
		Opcm:  common.Address{0xbb},
		UpgradeInputV2: &UpgradeInputV2{
			SystemConfig: common.Address{0x01},
			DisputeGameConfigs: []DisputeGameConfig{
				{
					Enabled:  true,
					InitBond: big.NewInt(1000),
					GameType: GameTypeCannon,
					GameArgs: []byte{0x01, 0x02, 0x03},
				},
			},
			ExtraInstructions: []ExtraInstruction{
				{
					Key:  "test-key",
					Data: []byte{0x04, 0x05, 0x06},
				},
			},
		},
	}
	data, err := input.EncodedUpgradeInputV2()

	require.NoError(t, err)
	require.NotEmpty(t, data)

	expected := "0000000000000000000000000000000000000000000000000000000000000020" + // offset to tuple
		"0000000000000000000000000100000000000000000000000000000000000000" + // systemConfig
		"0000000000000000000000000000000000000000000000000000000000000060" + // offset to disputeGameConfigs
		"0000000000000000000000000000000000000000000000000000000000000160" + // offset to extraInstructions
		"0000000000000000000000000000000000000000000000000000000000000001" + // disputeGameConfigs.length
		"0000000000000000000000000000000000000000000000000000000000000020" + // offset to disputeGameConfigs[0]
		"0000000000000000000000000000000000000000000000000000000000000001" + // disputeGameConfigs[0].enabled
		"00000000000000000000000000000000000000000000000000000000000003e8" + // disputeGameConfigs[0].initBond (1000)
		"0000000000000000000000000000000000000000000000000000000000000000" + // disputeGameConfigs[0].gameType
		"0000000000000000000000000000000000000000000000000000000000000080" + // offset to gameArgs
		"0000000000000000000000000000000000000000000000000000000000000003" + // gameArgs.length
		"0102030000000000000000000000000000000000000000000000000000000000" + // gameArgs data
		"0000000000000000000000000000000000000000000000000000000000000001" + // extraInstructions.length
		"0000000000000000000000000000000000000000000000000000000000000020" + // offset to extraInstructions[0]
		"0000000000000000000000000000000000000000000000000000000000000040" + // offset to key
		"0000000000000000000000000000000000000000000000000000000000000080" + // offset to data
		"0000000000000000000000000000000000000000000000000000000000000008" + // key.length
		"746573742d6b65790000000000000000000000000000000000000000000000" + // "test-key"
		"00" + // padding
		"0000000000000000000000000000000000000000000000000000000000000003" + // data.length
		"0405060000000000000000000000000000000000000000000000000000000000" // data

	require.Equal(t, expected, hex.EncodeToString(data))
}

func TestUpgradeOPChainInput_OpChainConfigs(t *testing.T) {
	input := &UpgradeOPChainInput{
		Prank: common.Address{0xaa},
		Opcm:  common.Address{0xbb},
		ChainConfigs: []OPChainConfig{
			{
				SystemConfigProxy:  common.Address{0x01},
				CannonPrestate:     common.Hash{0xaa},
				CannonKonaPrestate: common.Hash{0xbb},
			},
		},
	}
	data, err := input.EncodedOpChainConfigs()

	require.NoError(t, err)
	require.NotEmpty(t, data)

	expected := "0000000000000000000000000000000000000000000000000000000000000020" + // offset to array
		"0000000000000000000000000000000000000000000000000000000000000001" + // array.length
		"0000000000000000000000000100000000000000000000000000000000000000" + // systemConfigProxy
		"aa00000000000000000000000000000000000000000000000000000000000000" + // cannonPrestate
		"bb00000000000000000000000000000000000000000000000000000000000000" // cannonKonaPrestate

	require.Equal(t, expected, hex.EncodeToString(data))
}
