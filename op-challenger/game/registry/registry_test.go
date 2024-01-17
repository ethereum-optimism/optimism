package registry

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler"
	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler/test"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestUnknownGameType(t *testing.T) {
	registry := NewGameTypeRegistry()
	player, err := registry.CreatePlayer(types.GameMetadata{GameType: 0}, "")
	require.ErrorIs(t, err, ErrUnsupportedGameType)
	require.Nil(t, player)
}

func TestKnownGameType(t *testing.T) {
	registry := NewGameTypeRegistry()
	expectedPlayer := &test.StubGamePlayer{}
	creator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		return expectedPlayer, nil
	}
	registry.RegisterGameType(0, creator, nil)
	player, err := registry.CreatePlayer(types.GameMetadata{GameType: 0}, "")
	require.NoError(t, err)
	require.Same(t, expectedPlayer, player)
}

func TestPanicsOnDuplicateGameType(t *testing.T) {
	registry := NewGameTypeRegistry()
	creator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		return nil, nil
	}
	registry.RegisterGameType(0, creator, nil)
	require.Panics(t, func() {
		registry.RegisterGameType(0, creator, nil)
	})
}

func TestDeduplicateOracles(t *testing.T) {
	registry := NewGameTypeRegistry()
	creator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		return nil, nil
	}
	oracleA := stubPreimageOracle{0xaa}
	oracleB := stubPreimageOracle{0xbb}
	registry.RegisterGameType(0, creator, oracleA)
	registry.RegisterGameType(1, creator, oracleB)
	registry.RegisterGameType(2, creator, oracleB)
	oracles := registry.Oracles()
	require.Len(t, oracles, 2)
	require.Contains(t, oracles, oracleA)
	require.Contains(t, oracles, oracleB)
}

type stubPreimageOracle common.Address

func (s stubPreimageOracle) Addr() common.Address {
	return common.Address(s)
}
