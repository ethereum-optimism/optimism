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
		{value: "4294967296", expected: gameTypes.UnknownGameType, err: gameTypes.ErrUnknownGameType},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			actual, err := parseGameType(test.value)
			require.ErrorIs(t, err, test.err)
			require.Equal(t, test.expected, actual)
		})
	}
}

// Both inputs would otherwise silently become 0 and create a game that loses its bond. No
// --l1-eth-rpc is given, so each must fail before any RPC call.
func TestCreateGameRejectsZeroingArgs(t *testing.T) {
	run := func(args ...string) error {
		app := &cli.App{Commands: []*cli.Command{CreateGameCommand}}
		return app.Run(append([]string{"op-challenger", "create-game", "--game-type", "zk"}, args...))
	}
	require.ErrorContains(t, run("--l2-chain-id", "10", "--parent-index", "4294967296"), "parent index 4294967296 exceeds uint32 max")
	require.ErrorContains(t, run(), "--l2-chain-id must be a non-zero chain ID for game type zk")
}
