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

func TestCheckDisputeGame_ZK_ZeroImpl(t *testing.T) {
	st := &state.State{
		ImplementationsDeployment: &addresses.ImplementationsContracts{
			ZkDisputeGameImpl: common.Address{}, // zero — predeployed OPCM without ZK
		},
	}
	game := state.AdditionalDisputeGame{
		VMType: state.VMTypeZK,
		ZKDisputeGame: &state.ZKDisputeGameParams{
			AbsolutePrestate: common.HexToHash("0xdeadbeef"),
			ChallengerBond:   (*hexutil.Big)(big.NewInt(1e18)),
		},
	}

	err := checkDisputeGame(st, game)
	require.ErrorContains(t, err, "ZkDisputeGameImpl is not deployed")
}

func TestCheckDisputeGame_ZK_NilParams(t *testing.T) {
	st := &state.State{
		ImplementationsDeployment: &addresses.ImplementationsContracts{
			ZkDisputeGameImpl: common.HexToAddress("0x2222222222222222222222222222222222222222"),
		},
	}
	game := state.AdditionalDisputeGame{
		VMType:        state.VMTypeZK,
		ZKDisputeGame: nil, // params not set
	}

	err := checkDisputeGame(st, game)
	require.ErrorContains(t, err, "ZKDisputeGame params must be set")
}

func TestCheckDisputeGame_ZK_WrongDisputeGameType(t *testing.T) {
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

	err := checkDisputeGame(st, game)
	require.ErrorContains(t, err, "DisputeGameType must be")
}

func TestCheckDisputeGame_ZK_NilChallengerBond(t *testing.T) {
	st := &state.State{
		ImplementationsDeployment: &addresses.ImplementationsContracts{
			ZkDisputeGameImpl:   common.HexToAddress("0x2222222222222222222222222222222222222222"),
			SP1PlonkAdapterImpl: common.HexToAddress("0x3333333333333333333333333333333333333333"),
		},
	}
	game := state.AdditionalDisputeGame{
		VMType: state.VMTypeZK,
		ZKDisputeGame: &state.ZKDisputeGameParams{
			ChallengerBond: nil,
		},
		ChainProofParams: state.ChainProofParams{DisputeGameType: 10},
	}

	err := checkDisputeGame(st, game)
	require.ErrorContains(t, err, "ChallengerBond must be set")
}

func TestCheckDisputeGame_ZK_ZeroChallengerBond(t *testing.T) {
	st := &state.State{
		ImplementationsDeployment: &addresses.ImplementationsContracts{
			ZkDisputeGameImpl:   common.HexToAddress("0x2222222222222222222222222222222222222222"),
			SP1PlonkAdapterImpl: common.HexToAddress("0x3333333333333333333333333333333333333333"),
		},
	}
	game := state.AdditionalDisputeGame{
		VMType: state.VMTypeZK,
		ZKDisputeGame: &state.ZKDisputeGameParams{
			ChallengerBond: (*hexutil.Big)(big.NewInt(0)),
		},
		ChainProofParams: state.ChainProofParams{DisputeGameType: 10},
	}

	err := checkDisputeGame(st, game)
	require.ErrorContains(t, err, "ChallengerBond must be set")
}

func TestCheckDisputeGame_ZK_ZeroSP1PlonkAdapter(t *testing.T) {
	st := &state.State{
		ImplementationsDeployment: &addresses.ImplementationsContracts{
			ZkDisputeGameImpl: common.HexToAddress("0x2222222222222222222222222222222222222222"),
		},
	}
	game := state.AdditionalDisputeGame{
		VMType:           state.VMTypeZK,
		ZKDisputeGame:    &state.ZKDisputeGameParams{},
		ChainProofParams: state.ChainProofParams{DisputeGameType: 10},
	}

	err := checkDisputeGame(st, game)
	require.ErrorContains(t, err, "SP1PlonkAdapterImpl is not deployed")
}

// TestDeployAdditionalDisputeGames_ChecksEveryGameFirst checks that a game that cannot be deployed
// fails the stage before the games before it send anything.
func TestDeployAdditionalDisputeGames_ChecksEveryGameFirst(t *testing.T) {
	chainID := common.Hash{31: 1}
	deployer := common.Address{'D'}
	intent := &state.Intent{Chains: []*state.ChainIntent{{
		ID:    chainID,
		Roles: state.ChainRoles{L1ProxyAdminOwner: deployer},
		AdditionalDisputeGames: []state.AdditionalDisputeGame{
			{VMType: state.VMTypeCannon},
			{VMType: state.VMTypeZK},
		},
	}}}
	st := &state.State{
		ImplementationsDeployment: &addresses.ImplementationsContracts{},
		Chains:                    []*state.ChainState{{ID: chainID}},
	}
	// The environment has no engine to run the scripts of the first game.
	env := &Env{Logger: testlog.Logger(t, slog.LevelInfo), Deployer: deployer}

	err := DeployAdditionalDisputeGames(env, intent, st, chainID)
	require.ErrorContains(t, err, "ZkDisputeGameImpl is not deployed")
}

func TestCheckDisputeGame_UnsupportedVMType(t *testing.T) {
	st := &state.State{
		ImplementationsDeployment: &addresses.ImplementationsContracts{},
	}
	game := state.AdditionalDisputeGame{
		VMType: state.VMType("UNSUPPORTED"),
	}

	err := checkDisputeGame(st, game)
	require.ErrorContains(t, err, "unsupported VM type")
}
