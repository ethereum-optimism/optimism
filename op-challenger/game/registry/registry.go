package registry

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/claims"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
)

var ErrUnsupportedGameType = errors.New("unsupported game type")

type GameTypeRegistry struct {
	types        map[faultTypes.GameType]scheduler.PlayerCreator
	bondCreators map[faultTypes.GameType]claims.BondContractCreator
}

func NewGameTypeRegistry() *GameTypeRegistry {
	return &GameTypeRegistry{
		types:        make(map[faultTypes.GameType]scheduler.PlayerCreator),
		bondCreators: make(map[faultTypes.GameType]claims.BondContractCreator),
	}
}

// RegisterGameType registers a scheduler.PlayerCreator to use for a specific game type.
// Panics if the same game type is registered multiple times, since this indicates a significant programmer error.
func (r *GameTypeRegistry) RegisterGameType(gameType faultTypes.GameType, creator scheduler.PlayerCreator) {
	if _, ok := r.types[gameType]; ok {
		panic(fmt.Errorf("duplicate creator registered for game type: %v", gameType))
	}
	r.types[gameType] = creator
}

func (r *GameTypeRegistry) RegisterBondContract(gameType faultTypes.GameType, creator claims.BondContractCreator) {
	if _, ok := r.bondCreators[gameType]; ok {
		panic(fmt.Errorf("duplicate bond contract registered for game type: %v", gameType))
	}
	r.bondCreators[gameType] = creator
}

// CreatePlayer creates a new game player for the given game, using the specified directory for persisting data.
func (r *GameTypeRegistry) CreatePlayer(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
	creator, ok := r.types[faultTypes.GameType(game.GameType)]
	if !ok {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedGameType, game.GameType)
	}
	return creator(game, dir)
}

func (r *GameTypeRegistry) CreateBondContract(game types.GameMetadata) (claims.BondContract, error) {
	creator, ok := r.bondCreators[faultTypes.GameType(game.GameType)]
	if !ok {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedGameType, game.GameType)
	}
	return creator(game)
}
