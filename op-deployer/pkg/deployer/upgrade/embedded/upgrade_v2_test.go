package embedded

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestUpgradeOPChainV2Input_UpgradeInput tests that the UpgradeInput encoding works correctly.
func TestUpgradeOPChainV2Input_UpgradeInput(t *testing.T) {
	input := &UpgradeOPChainV2Input{
		Prank: common.Address{0xaa},
		Opcm:  common.Address{0xbb},
		UpgradeInputV2: UpgradeInputV2{
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

	// Test encoding
	data, err := input.UpgradeInput()
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// Verify expected encoding structure matches v5_0_0
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

// TestUpgradeOPChainV2Input_JSONMarshaling tests that the input can be marshaled and unmarshaled correctly.
func TestUpgradeOPChainV2Input_JSONMarshaling(t *testing.T) {
	input := &UpgradeOPChainV2Input{
		Prank: common.HexToAddress("0x1Eb2fFc903729a0F03966B917003800b145F56E2"),
		Opcm:  common.HexToAddress("0xEA055C82D6B0543CE0931b52B206242Cb9D262F9"),
		UpgradeInputV2: UpgradeInputV2{
			SystemConfig: common.HexToAddress("0x034edD2A225f7f429A63E0f1D2084B9E0A93b538"),
			DisputeGameConfigs: []DisputeGameConfig{
				{
					Enabled:  true,
					InitBond: big.NewInt(0),
					GameType: GameTypeCannon,
					GameArgs: []byte{},
				},
			},
			ExtraInstructions: []ExtraInstruction{},
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(input)
	require.NoError(t, err)
	require.NotEmpty(t, jsonData)

	// Unmarshal back
	var decoded UpgradeOPChainV2Input
	err = json.Unmarshal(jsonData, &decoded)
	require.NoError(t, err)

	// Verify fields
	require.Equal(t, input.Prank, decoded.Prank)
	require.Equal(t, input.Opcm, decoded.Opcm)
	require.Equal(t, input.UpgradeInputV2.SystemConfig, decoded.UpgradeInputV2.SystemConfig)
	require.Len(t, decoded.UpgradeInputV2.DisputeGameConfigs, 1)
	require.Equal(t, input.UpgradeInputV2.DisputeGameConfigs[0].Enabled, decoded.UpgradeInputV2.DisputeGameConfigs[0].Enabled)
	require.Equal(t, input.UpgradeInputV2.DisputeGameConfigs[0].GameType, decoded.UpgradeInputV2.DisputeGameConfigs[0].GameType)
}
