package embedded

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestUpgradeOPChainInput_UpgradeInputV2(t *testing.T) {
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

func TestUpgrader_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         UpgradeOPChainInput
		errorContains string
	}{
		{
			name: "neither input provided - validation fails",
			input: UpgradeOPChainInput{
				Prank: common.Address{0xaa},
				Opcm:  common.Address{0xbb},
			},
			errorContains: "failed to read either an upgrade input or config array",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upgrader := DefaultUpgrader

			// Convert input to JSON to test the Upgrader.Upgrade method
			inputJSON, err := json.Marshal(tt.input)
			require.NoError(t, err)

			// Call Upgrade with nil host - validation should fail before script execution
			err = upgrader.Upgrade(nil, inputJSON)

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.errorContains)
		})
	}
}

func TestUpgrader_ValidationPasses(t *testing.T) {
	tests := []struct {
		name        string
		input       UpgradeOPChainInput
		description string
	}{
		{
			name: "V2 input provided",
			input: UpgradeOPChainInput{
				Prank: common.Address{0xaa},
				Opcm:  common.Address{0xbb},
				UpgradeInputV2: &UpgradeInputV2{
					SystemConfig: common.Address{0x01},
					DisputeGameConfigs: []DisputeGameConfig{
						{
							Enabled:  true,
							InitBond: big.NewInt(1000),
							GameType: GameTypeCannon,
							GameArgs: []byte{0x01, 0x02},
						},
					},
				},
			},
			description: "Validation should pass when V2 input is provided and ShouldAllowV1 is false",
		},
		{
			name: "only V1 input provided",
			input: UpgradeOPChainInput{
				Prank: common.Address{0xaa},
				Opcm:  common.Address{0xbb},
				ChainConfigs: []OPChainConfig{
					{
						SystemConfigProxy:  common.Address{0x01},
						CannonPrestate:     common.Hash{0xaa},
						CannonKonaPrestate: common.Hash{0xbb},
					},
				},
			},
			description: "Validation should pass when V1 input is provided",
		},
		{
			name: "both inputs provided",
			input: UpgradeOPChainInput{
				Prank: common.Address{0xaa},
				Opcm:  common.Address{0xbb},
				UpgradeInputV2: &UpgradeInputV2{
					SystemConfig: common.Address{0x01},
					DisputeGameConfigs: []DisputeGameConfig{
						{
							Enabled:  true,
							InitBond: big.NewInt(1000),
							GameType: GameTypeCannon,
							GameArgs: []byte{0x01, 0x02},
						},
					},
				},
				ChainConfigs: []OPChainConfig{
					{
						SystemConfigProxy:  common.Address{0x02},
						CannonPrestate:     common.Hash{0xcc},
						CannonKonaPrestate: common.Hash{0xdd},
					},
				},
			},
			description: "Validation should pass when both inputs are provided and ShouldAllowV1 is true (should prefer V2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify that encoding works (validation passes)
			// We test the encoding separately since we can't test the full Upgrade flow without a script host
			upgradeInput := tt.input

			// Test that the correct encoding path would be chosen
			if upgradeInput.UpgradeInputV2 != nil {
				_, err := upgradeInput.EncodedUpgradeInputV2()
				require.NoError(t, err, "V2 encoding should succeed when V2 input is present")
			} else if len(upgradeInput.ChainConfigs) > 0 {
				_, err := upgradeInput.EncodedOpChainConfigs()
				require.NoError(t, err, "V1 encoding should succeed when V1 input is present")
			}
		})
	}
}
