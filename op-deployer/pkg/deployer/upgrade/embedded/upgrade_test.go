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
			errorContains: "UpgradeInputV2 is required",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify that encoding works (validation passes)
			_, err := tt.input.EncodedUpgradeInputV2()
			require.NoError(t, err, "V2 encoding should succeed")
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
			name: "ZK_DISPUTE_GAME requires ZKDisputeGameConfig",
			gameConfig: DisputeGameConfig{
				Enabled:  true,
				InitBond: big.NewInt(1000),
				GameType: GameTypeZKDisputeGame,
				// Missing ZKDisputeGameConfig
			},
			errorContains: fmt.Sprintf("zkDisputeGameConfig is required for game type %d", GameTypeZKDisputeGame),
			shouldPass:    false,
		},
		{
			name: "ZK_DISPUTE_GAME with zero Verifier returns error",
			gameConfig: DisputeGameConfig{
				Enabled:  true,
				InitBond: big.NewInt(1000),
				GameType: GameTypeZKDisputeGame,
				ZKDisputeGameConfig: &ZKDisputeGameConfig{
					AbsolutePrestate:     common.HexToHash("0x038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c"),
					Verifier:             common.Address{}, // zero
					MaxChallengeDuration: 3600,
					MaxProveDuration:     7200,
					ChallengerBond:       new(big.Int).SetUint64(1e9),
				},
			},
			errorContains: "Verifier must not be zero address",
			shouldPass:    false,
		},
		{
			name: "ZK_DISPUTE_GAME with zero AbsolutePrestate returns error",
			gameConfig: DisputeGameConfig{
				Enabled:  true,
				InitBond: big.NewInt(1000),
				GameType: GameTypeZKDisputeGame,
				ZKDisputeGameConfig: &ZKDisputeGameConfig{
					AbsolutePrestate:     common.Hash{}, // zero
					Verifier:             common.HexToAddress("0x3333333333333333333333333333333333333333"),
					MaxChallengeDuration: 3600,
					MaxProveDuration:     7200,
					ChallengerBond:       new(big.Int).SetUint64(1e9),
				},
			},
			errorContains: "AbsolutePrestate must not be zero",
			shouldPass:    false,
		},
		{
			name: "ZK_DISPUTE_GAME with zero MaxChallengeDuration returns error",
			gameConfig: DisputeGameConfig{
				Enabled:  true,
				InitBond: big.NewInt(1000),
				GameType: GameTypeZKDisputeGame,
				ZKDisputeGameConfig: &ZKDisputeGameConfig{
					AbsolutePrestate:     common.HexToHash("0x038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c"),
					Verifier:             common.HexToAddress("0x3333333333333333333333333333333333333333"),
					MaxChallengeDuration: 0, // zero
					MaxProveDuration:     7200,
					ChallengerBond:       new(big.Int).SetUint64(1e9),
				},
			},
			errorContains: "MaxChallengeDuration must be > 0",
			shouldPass:    false,
		},
		{
			name: "ZK_DISPUTE_GAME with zero MaxProveDuration returns error",
			gameConfig: DisputeGameConfig{
				Enabled:  true,
				InitBond: big.NewInt(1000),
				GameType: GameTypeZKDisputeGame,
				ZKDisputeGameConfig: &ZKDisputeGameConfig{
					AbsolutePrestate:     common.HexToHash("0x038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c"),
					Verifier:             common.HexToAddress("0x3333333333333333333333333333333333333333"),
					MaxChallengeDuration: 3600,
					MaxProveDuration:     0, // zero
					ChallengerBond:       new(big.Int).SetUint64(1e9),
				},
			},
			errorContains: "MaxProveDuration must be > 0",
			shouldPass:    false,
		},
		{
			name: "ZK_DISPUTE_GAME with nil ChallengerBond returns error",
			gameConfig: DisputeGameConfig{
				Enabled:  true,
				InitBond: big.NewInt(1000),
				GameType: GameTypeZKDisputeGame,
				ZKDisputeGameConfig: &ZKDisputeGameConfig{
					AbsolutePrestate:     common.HexToHash("0x038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c"),
					Verifier:             common.HexToAddress("0x3333333333333333333333333333333333333333"),
					MaxChallengeDuration: 3600,
					MaxProveDuration:     7200,
					ChallengerBond:       nil, // nil
				},
			},
			errorContains: "ChallengerBond must be set to a positive value",
			shouldPass:    false,
		},
		{
			name: "ZK_DISPUTE_GAME with zero ChallengerBond returns error",
			gameConfig: DisputeGameConfig{
				Enabled:  true,
				InitBond: big.NewInt(1000),
				GameType: GameTypeZKDisputeGame,
				ZKDisputeGameConfig: &ZKDisputeGameConfig{
					AbsolutePrestate:     common.HexToHash("0x038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c"),
					Verifier:             common.HexToAddress("0x3333333333333333333333333333333333333333"),
					MaxChallengeDuration: 3600,
					MaxProveDuration:     7200,
					ChallengerBond:       big.NewInt(0), // zero
				},
			},
			errorContains: "ChallengerBond must be set to a positive value",
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
		{
			name: "ZK_DISPUTE_GAME with correct ZKDisputeGameConfig",
			gameConfig: DisputeGameConfig{
				Enabled:  true,
				InitBond: big.NewInt(1000),
				GameType: GameTypeZKDisputeGame,
				ZKDisputeGameConfig: &ZKDisputeGameConfig{
					AbsolutePrestate:     common.HexToHash("0x038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c"),
					Verifier:             common.HexToAddress("0x3333333333333333333333333333333333333333"),
					MaxChallengeDuration: 3600,
					MaxProveDuration:     7200,
					ChallengerBond:       new(big.Int).SetUint64(1e9),
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
			name: "disabled ZK game with empty config",
			gameConfigs: []DisputeGameConfig{
				{
					Enabled:  false,
					InitBond: big.NewInt(0),
					GameType: GameTypeZKDisputeGame,
					// No ZKDisputeGameConfig needed when disabled
				},
			},
			description: "Disabled ZK game should encode successfully with no config",
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

	t.Run("ZKDisputeGameConfig encodes correctly", func(t *testing.T) {
		absolutePrestate := common.HexToHash("0x038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c")
		verifier := common.HexToAddress("0x3333333333333333333333333333333333333333")
		// maxChallengeDuration = 3600 = 0xe10
		// maxProveDuration = 7200 = 0x1c20
		// challengerBond = 1e18 = 0xde0b6b3a7640000
		challengerBond, _ := new(big.Int).SetString("1000000000000000000", 10)

		input := &UpgradeOPChainInput{
			Prank: common.Address{0xaa},
			Opcm:  common.Address{0xbb},
			UpgradeInputV2: &UpgradeInputV2{
				SystemConfig: common.Address{0x01},
				DisputeGameConfigs: []DisputeGameConfig{
					{
						Enabled:  true,
						InitBond: big.NewInt(1000),
						GameType: GameTypeZKDisputeGame,
						ZKDisputeGameConfig: &ZKDisputeGameConfig{
							AbsolutePrestate:     absolutePrestate,
							Verifier:             verifier,
							MaxChallengeDuration: 3600,
							MaxProveDuration:     7200,
							ChallengerBond:       challengerBond,
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
			"00000000000000000000000000000000000000000000000000000000000001e0" + // offset to extraInstructions
			"0000000000000000000000000000000000000000000000000000000000000001" + // disputeGameConfigs.length
			"0000000000000000000000000000000000000000000000000000000000000020" + // offset to disputeGameConfigs[0]
			"0000000000000000000000000000000000000000000000000000000000000001" + // disputeGameConfigs[0].enabled
			"00000000000000000000000000000000000000000000000000000000000003e8" + // disputeGameConfigs[0].initBond (1000)
			"000000000000000000000000000000000000000000000000000000000000000a" + // disputeGameConfigs[0].gameType (10)
			"0000000000000000000000000000000000000000000000000000000000000080" + // offset to gameArgs
			"00000000000000000000000000000000000000000000000000000000000000a0" + // gameArgs.length (160 bytes)
			"038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c" + // absolutePrestate
			"0000000000000000000000003333333333333333333333333333333333333333" + // verifier
			"0000000000000000000000000000000000000000000000000000000000000e10" + // maxChallengeDuration (3600)
			"0000000000000000000000000000000000000000000000000000000000001c20" + // maxProveDuration (7200)
			"0000000000000000000000000000000000000000000000000de0b6b3a7640000" + // challengerBond (1e18)
			"0000000000000000000000000000000000000000000000000000000000000000" // extraInstructions.length

		require.Equal(t, expected, hex.EncodeToString(data))
	})
}
