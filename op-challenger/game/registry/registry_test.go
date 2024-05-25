package registry

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler"
	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler/test"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
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
	registry.RegisterGameType(0, creator)
	player, err := registry.CreatePlayer(types.GameMetadata{GameType: 0}, "")
	require.NoError(t, err)
	require.Same(t, expectedPlayer, player)
}

func TestPanicsOnDuplicateGameType(t *testing.T) {
	registry := NewGameTypeRegistry()
	creator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		return nil, nil
	}
	registry.RegisterGameType(0, creator)
	require.Panics(t, func() {
		registry.RegisterGameType(0, creator)
	})
}
