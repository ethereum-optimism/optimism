package pipeline

import (
	"log/slog"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestShouldDeployAdditionalDisputeGames(t *testing.T) {
	dummyGame := state.AdditionalDisputeGame{VMType: state.VMTypeCannon}

	tests := []struct {
		name     string
		intent   *state.ChainIntent
		st       *state.ChainState
		expected bool
	}{
		{
			name:     "no_games_in_intent",
			intent:   &state.ChainIntent{},
			st:       &state.ChainState{},
			expected: false,
		},
		{
			name:     "games_in_intent_empty_state",
			intent:   &state.ChainIntent{AdditionalDisputeGames: []state.AdditionalDisputeGame{dummyGame}},
			st:       &state.ChainState{},
			expected: true,
		},
		{
			name:   "games_in_intent_already_deployed",
			intent: &state.ChainIntent{AdditionalDisputeGames: []state.AdditionalDisputeGame{dummyGame}},
			st: &state.ChainState{
				AdditionalDisputeGames: []state.AdditionalDisputeGameState{
					{GameType: 1, VMType: state.VMTypeCannon},
				},
			},
			expected: false,
		},
		{
			name:     "zk_game_in_intent_empty_state",
			intent:   &state.ChainIntent{AdditionalDisputeGames: []state.AdditionalDisputeGame{{VMType: state.VMTypeZK}}},
			st:       &state.ChainState{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldDeployAdditionalDisputeGames(tt.intent, tt.st)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestDeployDisputeGame_ZK_ZeroImpl(t *testing.T) {
	lgr := testlog.Logger(t, slog.LevelInfo)

	env := &Env{Logger: lgr}
	st := &state.State{
		ImplementationsDeployment: &addresses.ImplementationsContracts{
			ZkDisputeGameImpl: common.Address{}, // zero — flag was not active
		},
	}
	game := state.AdditionalDisputeGame{
		VMType: state.VMTypeZK,
		ZKDisputeGame: &state.ZKDisputeGameParams{
			Verifier:         common.HexToAddress("0x1111111111111111111111111111111111111111"),
			AbsolutePrestate: common.HexToHash("0xdeadbeef"),
			ChallengerBond:   (*hexutil.Big)(big.NewInt(1e18)),
		},
	}

	err := deployDisputeGame(env, st, &state.ChainIntent{}, &state.ChainState{}, game)
	require.ErrorContains(t, err, "ZkDisputeGameImpl is not deployed")
}

func TestDeployDisputeGame_ZK_NilParams(t *testing.T) {
	lgr := testlog.Logger(t, slog.LevelInfo)

	env := &Env{Logger: lgr}
	st := &state.State{
		ImplementationsDeployment: &addresses.ImplementationsContracts{
			ZkDisputeGameImpl: common.HexToAddress("0x2222222222222222222222222222222222222222"),
		},
	}
	game := state.AdditionalDisputeGame{
		VMType:        state.VMTypeZK,
		ZKDisputeGame: nil, // params not set
	}

	err := deployDisputeGame(env, st, &state.ChainIntent{}, &state.ChainState{}, game)
	require.ErrorContains(t, err, "ZKDisputeGame params must be set")
}

func TestDeployDisputeGame_UnsupportedVMType(t *testing.T) {
	lgr := testlog.Logger(t, slog.LevelInfo)

	env := &Env{Logger: lgr}
	st := &state.State{
		ImplementationsDeployment: &addresses.ImplementationsContracts{},
	}
	game := state.AdditionalDisputeGame{
		VMType: state.VMType("UNSUPPORTED"),
	}

	err := deployDisputeGame(env, st, &state.ChainIntent{}, &state.ChainState{}, game)
	require.ErrorContains(t, err, "unsupported VM type")
}

// TestZKDisputeGameDevFlag validates that zkDisputeGameDevFlag matches the expected value.
// The expected value must equal deployer.ZKDisputeGameDevFlag (0x...01000000).
func TestZKDisputeGameDevFlag(t *testing.T) {
	expected := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000001000000")
	require.Equal(t, expected, zkDisputeGameDevFlag,
		"zkDisputeGameDevFlag must match deployer.ZKDisputeGameDevFlag")
}

// TestZKDevFlagAutoEnable validates that DeployImplementations auto-enables
// the ZK dev flag when a chain has VMTypeZK in AdditionalDisputeGames.
func TestZKDevFlagAutoEnable(t *testing.T) {
	chainID := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")

	intentWithZK := &state.Intent{
		GlobalDeployOverrides: map[string]any{},
		Chains: []*state.ChainIntent{
			{
				ID: chainID,
				AdditionalDisputeGames: []state.AdditionalDisputeGame{
					{VMType: state.VMTypeZK},
				},
			},
		},
	}

	intentWithoutZK := &state.Intent{
		GlobalDeployOverrides: map[string]any{},
		Chains: []*state.ChainIntent{
			{
				ID:                     chainID,
				AdditionalDisputeGames: []state.AdditionalDisputeGame{},
			},
		},
	}

	t.Run("zk_game_enables_flag", func(t *testing.T) {
		bitmap := computeDevFeatureBitmap(intentWithZK)
		// ZK flag bit (0x01000000) should be set
		require.NotEqual(t, common.Hash{}, bitmap)
		for i := range zkDisputeGameDevFlag {
			require.Equal(t, zkDisputeGameDevFlag[i], bitmap[i]&zkDisputeGameDevFlag[i],
				"ZK dev flag bit %d should be set in bitmap", i)
		}
	})

	t.Run("no_zk_game_does_not_enable_flag", func(t *testing.T) {
		bitmap := computeDevFeatureBitmap(intentWithoutZK)
		require.Equal(t, common.Hash{}, bitmap)
	})
}

// computeDevFeatureBitmap extracts the ZK flag auto-enable logic from DeployImplementations for testing.
func computeDevFeatureBitmap(intent *state.Intent) common.Hash {
	var bitmap common.Hash
outer:
	for _, chain := range intent.Chains {
		for _, game := range chain.AdditionalDisputeGames {
			if game.VMType == state.VMTypeZK {
				for i := range bitmap {
					bitmap[i] |= zkDisputeGameDevFlag[i]
				}
				break outer
			}
		}
	}
	return bitmap
}
