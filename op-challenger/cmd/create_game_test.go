package main

import (
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestParseGameType(t *testing.T) {
	tests := []struct {
		value    string
		expected gameTypes.GameType
		err      error
	}{
		{value: "10", expected: gameTypes.ZKDisputeGameType},
		{value: "zk", expected: gameTypes.ZKDisputeGameType},
		{value: "super-permissioned", expected: gameTypes.SuperPermissionedGameType},
		{value: GameTypeFlag.Value, expected: gameTypes.CannonGameType},
		{value: "zkk", expected: gameTypes.UnknownGameType, err: gameTypes.ErrUnknownGameType},
		{value: "2", expected: gameTypes.UnknownGameType, err: gameTypes.ErrUnknownGameType},
		{value: "op-succinct", expected: gameTypes.UnknownGameType, err: gameTypes.ErrUnknownGameType},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			actual, err := parseGameType(test.value)
			require.ErrorIs(t, err, test.err)
			require.Equal(t, test.expected, actual)
		})
	}
}

// Out-of-range or misplaced --parent-index and zero --l2-chain-id fail before any RPC call.
func TestCreateGameRejectsInvalidSuperArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		err  string
	}{
		{name: "ZKMissingChainID", args: []string{"--game-type", "zk"}, err: "--l2-chain-id must be a non-zero chain ID for game type zk"},
		{name: "SuperCannonKonaMissingChainID", args: []string{"--game-type", "super-cannon-kona"}, err: "--l2-chain-id must be a non-zero chain ID for game type super-cannon-kona"},
		{name: "SuperPermissionedMissingChainID", args: []string{"--game-type", "super-permissioned"}, err: "--l2-chain-id must be a non-zero chain ID for game type super-permissioned"},
		{name: "ParentIndexAboveUint32", args: []string{"--game-type", "zk", "--l2-chain-id", "10", "--parent-index", "4294967296"}, err: "parent index 4294967296 exceeds uint32 max"},
		{name: "ParentIndexForNonZKGame", args: []string{"--game-type", "super-cannon-kona", "--l2-chain-id", "10", "--parent-index", "3"}, err: "--parent-index is only valid for game type zk"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := &cli.App{Commands: []*cli.Command{CreateGameCommand}}
			err := app.Run(append([]string{"op-challenger", "create-game"}, test.args...))
			require.ErrorContains(t, err, test.err)
		})
	}
}
