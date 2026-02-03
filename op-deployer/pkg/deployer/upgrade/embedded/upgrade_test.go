package embedded

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
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
					FaultDisputeGameConfig: &FaultDisputeGameConfig{
						AbsolutePrestate: common.Hash{0x01, 0x02, 0x03},
					},
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
		"0000000000000000000000000000000000000000000000000000000000000020" + // gameArgs.length (32 bytes for ABI-encoded bytes32)
		"0102030000000000000000000000000000000000000000000000000000000000" + // gameArgs data (absolutePrestate as bytes32)
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
							FaultDisputeGameConfig: &FaultDisputeGameConfig{
								AbsolutePrestate: common.Hash{0x01, 0x02},
							},
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
							FaultDisputeGameConfig: &FaultDisputeGameConfig{
								AbsolutePrestate: common.Hash{0x01, 0x02},
							},
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

func TestEncodedUpgradeInputV2_GameTypeConfigValidation(t *testing.T) {
	tests := []struct {
		name          string
		gameConfig    DisputeGameConfig
		errorContains string
		shouldPass    bool
	}{
		{
			name: "CANNON requires FaultDisputeGameConfig",
			gameConfig: DisputeGameConfig{
				Enabled:  true,
				InitBond: big.NewInt(1000),
				GameType: GameTypeCannon,
				// Missing FaultDisputeGameConfig
			},
			errorContains: fmt.Sprintf("faultDisputeGameConfig is required for game type %d", GameTypeCannon),
			shouldPass:    false,
		},
		{
			name: "CANNON_KONA requires FaultDisputeGameConfig",
			gameConfig: DisputeGameConfig{
				Enabled:  true,
				InitBond: big.NewInt(1000),
				GameType: GameTypeCannonKona,
				// Missing FaultDisputeGameConfig
			},
			errorContains: fmt.Sprintf("faultDisputeGameConfig is required for game type %d", GameTypeCannonKona),
			shouldPass:    false,
		},
		{
			name: "PERMISSIONED_CANNON requires PermissionedDisputeGameConfig",
			gameConfig: DisputeGameConfig{
				Enabled:  true,
				InitBond: big.NewInt(1000),
				GameType: GameTypePermissionedCannon,
				// Missing PermissionedDisputeGameConfig
			},
			errorContains: fmt.Sprintf("permissionedDisputeGameConfig is required for game type %d", GameTypePermissionedCannon),
			shouldPass:    false,
		},
		{
			name: "invalid game type returns error",
			gameConfig: DisputeGameConfig{
				Enabled:  true,
				InitBond: big.NewInt(1000),
				GameType: GameType(99), // not a valid game type (0, 1, 8)
			},
			errorContains: fmt.Sprintf("invalid game type %d for opcm v2", GameType(99)),
			shouldPass:    false,
		},
		{
			name: "CANNON with correct FaultDisputeGameConfig",
			gameConfig: DisputeGameConfig{
				Enabled:  true,
				InitBond: big.NewInt(1000),
				GameType: GameTypeCannon,
				FaultDisputeGameConfig: &FaultDisputeGameConfig{
					AbsolutePrestate: common.HexToHash("0x038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c"),
				},
			},
			shouldPass: true,
		},
		{
			name: "CANNON_KONA with correct FaultDisputeGameConfig",
			gameConfig: DisputeGameConfig{
				Enabled:  true,
				InitBond: big.NewInt(1000),
				GameType: GameTypeCannonKona,
				FaultDisputeGameConfig: &FaultDisputeGameConfig{
					AbsolutePrestate: common.HexToHash("0x03c3ebfb8e75ee51bec0814b9eb7f2e8034df88897c232eed36ea217ff1e9f40"),
				},
			},
			shouldPass: true,
		},
		{
			name: "PERMISSIONED_CANNON with correct PermissionedDisputeGameConfig",
			gameConfig: DisputeGameConfig{
				Enabled:  true,
				InitBond: big.NewInt(1000),
				GameType: GameTypePermissionedCannon,
				PermissionedDisputeGameConfig: &PermissionedDisputeGameConfig{
					AbsolutePrestate: common.HexToHash("0x038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c"),
					Proposer:         common.HexToAddress("0x1111111111111111111111111111111111111111"),
					Challenger:       common.HexToAddress("0x2222222222222222222222222222222222222222"),
				},
			},
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &UpgradeOPChainInput{
				Prank: common.Address{0xaa},
				Opcm:  common.Address{0xbb},
				UpgradeInputV2: &UpgradeInputV2{
					SystemConfig:       common.Address{0x01},
					DisputeGameConfigs: []DisputeGameConfig{tt.gameConfig},
				},
			}

			data, err := input.EncodedUpgradeInputV2()

			if tt.shouldPass {
				require.NoError(t, err, "encoding should succeed for valid config")
				require.NotEmpty(t, data, "encoded data should not be empty")
			} else {
				require.Error(t, err, "encoding should fail for invalid config")
				require.Contains(t, err.Error(), tt.errorContains, "error message should contain expected text")
			}
		})
	}
}

func TestEncodedUpgradeInputV2_DisabledGames(t *testing.T) {
	tests := []struct {
		name        string
		gameConfigs []DisputeGameConfig
		description string
	}{
		{
			name: "disabled CANNON game with empty config",
			gameConfigs: []DisputeGameConfig{
				{
					Enabled:  false,
					InitBond: big.NewInt(0),
					GameType: GameTypeCannon,
					// No FaultDisputeGameConfig needed when disabled
				},
			},
			description: "Disabled CANNON game should encode successfully with no config",
		},
		{
			name: "disabled CANNON_KONA game with empty config",
			gameConfigs: []DisputeGameConfig{
				{
					Enabled:  false,
					InitBond: big.NewInt(0),
					GameType: GameTypeCannonKona,
					// No FaultDisputeGameConfig needed when disabled
				},
			},
			description: "Disabled CANNON_KONA game should encode successfully with no config",
		},
		{
			name: "disabled PERMISSIONED_CANNON game with empty config",
			gameConfigs: []DisputeGameConfig{
				{
					Enabled:  false,
					InitBond: big.NewInt(0),
					GameType: GameTypePermissionedCannon,
					// No PermissionedDisputeGameConfig needed when disabled
				},
			},
			description: "Disabled PERMISSIONED_CANNON game should encode successfully with no config",
		},
		{
			name: "mix of enabled and disabled games",
			gameConfigs: []DisputeGameConfig{
				{
					Enabled:  true,
					InitBond: big.NewInt(1000),
					GameType: GameTypeCannon,
					FaultDisputeGameConfig: &FaultDisputeGameConfig{
						AbsolutePrestate: common.HexToHash("0x038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c"),
					},
				},
				{
					Enabled:  true,
					InitBond: big.NewInt(500),
					GameType: GameTypePermissionedCannon,
					PermissionedDisputeGameConfig: &PermissionedDisputeGameConfig{
						AbsolutePrestate: common.HexToHash("0x038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c"),
						Proposer:         common.HexToAddress("0x1111111111111111111111111111111111111111"),
						Challenger:       common.HexToAddress("0x2222222222222222222222222222222222222222"),
					},
				},
				{
					Enabled:  false,
					InitBond: big.NewInt(0),
					GameType: GameTypeCannonKona,
					// No config needed when disabled
				},
			},
			description: "Mix of enabled and disabled games should encode successfully",
		},
		{
			name: "all games disabled",
			gameConfigs: []DisputeGameConfig{
				{
					Enabled:  false,
					InitBond: big.NewInt(0),
					GameType: GameTypeCannon,
				},
				{
					Enabled:  false,
					InitBond: big.NewInt(0),
					GameType: GameTypePermissionedCannon,
				},
				{
					Enabled:  false,
					InitBond: big.NewInt(0),
					GameType: GameTypeCannonKona,
				},
			},
			description: "All games disabled should encode successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &UpgradeOPChainInput{
				Prank: common.Address{0xaa},
				Opcm:  common.Address{0xbb},
				UpgradeInputV2: &UpgradeInputV2{
					SystemConfig:       common.Address{0x01},
					DisputeGameConfigs: tt.gameConfigs,
				},
			}

			data, err := input.EncodedUpgradeInputV2()
			require.NoError(t, err, tt.description)
			require.NotEmpty(t, data, "encoded data should not be empty")
		})
	}
}

func TestEncodedUpgradeInputV2_GameArgsEncoding(t *testing.T) {
	t.Run("FaultDisputeGameConfig encodes correctly", func(t *testing.T) {
		absolutePrestate := common.HexToHash("0x038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c")
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
						FaultDisputeGameConfig: &FaultDisputeGameConfig{
							AbsolutePrestate: absolutePrestate,
						},
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
			"0000000000000000000000000000000000000000000000000000000000000020" + // gameArgs.length (32 bytes)
			"038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c" + // gameArgs data (absolutePrestate)
			"0000000000000000000000000000000000000000000000000000000000000000" // extraInstructions.length

		require.Equal(t, expected, hex.EncodeToString(data))
	})

	t.Run("PermissionedDisputeGameConfig encodes correctly", func(t *testing.T) {
		absolutePrestate := common.HexToHash("0x038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c")
		proposer := common.HexToAddress("0x1111111111111111111111111111111111111111")
		challenger := common.HexToAddress("0x2222222222222222222222222222222222222222")

		input := &UpgradeOPChainInput{
			Prank: common.Address{0xaa},
			Opcm:  common.Address{0xbb},
			UpgradeInputV2: &UpgradeInputV2{
				SystemConfig: common.Address{0x01},
				DisputeGameConfigs: []DisputeGameConfig{
					{
						Enabled:  true,
						InitBond: big.NewInt(1000),
						GameType: GameTypePermissionedCannon,
						PermissionedDisputeGameConfig: &PermissionedDisputeGameConfig{
							AbsolutePrestate: absolutePrestate,
							Proposer:         proposer,
							Challenger:       challenger,
						},
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
			"00000000000000000000000000000000000000000000000000000000000001a0" + // offset to extraInstructions
			"0000000000000000000000000000000000000000000000000000000000000001" + // disputeGameConfigs.length
			"0000000000000000000000000000000000000000000000000000000000000020" + // offset to disputeGameConfigs[0]
			"0000000000000000000000000000000000000000000000000000000000000001" + // disputeGameConfigs[0].enabled
			"00000000000000000000000000000000000000000000000000000000000003e8" + // disputeGameConfigs[0].initBond (1000)
			"0000000000000000000000000000000000000000000000000000000000000001" + // disputeGameConfigs[0].gameType
			"0000000000000000000000000000000000000000000000000000000000000080" + // offset to gameArgs
			"0000000000000000000000000000000000000000000000000000000000000060" + // gameArgs.length (96 bytes)
			"038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c" + // gameArgs data (absolutePrestate)
			"0000000000000000000000001111111111111111111111111111111111111111" + // gameArgs data (proposer)
			"0000000000000000000000002222222222222222222222222222222222222222" + // gameArgs data (challenger)
			"0000000000000000000000000000000000000000000000000000000000000000" // extraInstructions.length

		require.Equal(t, expected, hex.EncodeToString(data))
	})
}
