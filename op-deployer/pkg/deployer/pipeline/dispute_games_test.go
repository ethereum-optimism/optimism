package pipeline

import (
	"log/slog"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
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

func TestDeployDisputeGame_ZK_WrongDisputeGameType(t *testing.T) {
	lgr := testlog.Logger(t, slog.LevelInfo)

	env := &Env{Logger: lgr}
	st := &state.State{
		ImplementationsDeployment: &addresses.ImplementationsContracts{
			ZkDisputeGameImpl: common.HexToAddress("0x2222222222222222222222222222222222222222"),
		},
	}
	game := state.AdditionalDisputeGame{
		VMType:           state.VMTypeZK,
		ZKDisputeGame:    &state.ZKDisputeGameParams{},
		ChainProofParams: state.ChainProofParams{DisputeGameType: 0}, // wrong — must be GameTypeZKDisputeGame (10)
	}

	err := deployDisputeGame(env, st, &state.ChainIntent{}, &state.ChainState{}, game)
	require.ErrorContains(t, err, "DisputeGameType must be")
}

func TestDeployDisputeGame_ZK_NilChallengerBond(t *testing.T) {
	lgr := testlog.Logger(t, slog.LevelInfo)

	env := &Env{Logger: lgr}
	st := &state.State{
		ImplementationsDeployment: &addresses.ImplementationsContracts{
			ZkDisputeGameImpl: common.HexToAddress("0x2222222222222222222222222222222222222222"),
		},
	}
	game := state.AdditionalDisputeGame{
		VMType: state.VMTypeZK,
		ZKDisputeGame: &state.ZKDisputeGameParams{
			ChallengerBond: nil,
		},
		ChainProofParams: state.ChainProofParams{DisputeGameType: 10},
	}

	err := deployDisputeGame(env, st, &state.ChainIntent{}, &state.ChainState{}, game)
	require.ErrorContains(t, err, "ChallengerBond must be set")
}

func TestDeployDisputeGame_ZK_ZeroChallengerBond(t *testing.T) {
	lgr := testlog.Logger(t, slog.LevelInfo)

	env := &Env{Logger: lgr}
	st := &state.State{
		ImplementationsDeployment: &addresses.ImplementationsContracts{
			ZkDisputeGameImpl: common.HexToAddress("0x2222222222222222222222222222222222222222"),
		},
	}
	game := state.AdditionalDisputeGame{
		VMType: state.VMTypeZK,
		ZKDisputeGame: &state.ZKDisputeGameParams{
			ChallengerBond: (*hexutil.Big)(big.NewInt(0)),
		},
		ChainProofParams: state.ChainProofParams{DisputeGameType: 10},
	}

	err := deployDisputeGame(env, st, &state.ChainIntent{}, &state.ChainState{}, game)
	require.ErrorContains(t, err, "ChallengerBond must be set")
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

// TestZKDisputeGameFlag validates that devfeatures.ZKDisputeGameFlag matches the expected value.
func TestZKDisputeGameFlag(t *testing.T) {
	expected := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000001000000")
	require.Equal(t, expected, devfeatures.ZKDisputeGameFlag,
		"devfeatures.ZKDisputeGameFlag must match the expected value")
}
