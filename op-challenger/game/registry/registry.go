package registry

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
)

var (
	ErrUnsupportedGameType = errors.New("unsupported game type")
)

type GameTypeRegistry struct {
	types map[uint8]scheduler.PlayerCreator
}

func NewGameTypeRegistry() *GameTypeRegistry {
	return &GameTypeRegistry{
		types: make(map[uint8]scheduler.PlayerCreator),
	}
}

// RegisterGameType registers a scheduler.PlayerCreator to use for a specific game type.
// Panics if the same game type is registered multiple times, since this indicates a significant programmer error.
func (r *GameTypeRegistry) RegisterGameType(gameType uint8, creator scheduler.PlayerCreator) {
	if _, ok := r.types[gameType]; ok {
		panic(fmt.Errorf("duplicate creator registered for game type: %v", gameType))
	}
	r.types[gameType] = creator
}

// CreatePlayer creates a new game player for the given game, using the specified directory for persisting data.
func (r *GameTypeRegistry) CreatePlayer(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
	creator, ok := r.types[game.GameType]
	if !ok {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedGameType, game.GameType)
	}
	return creator(game, dir)
}
