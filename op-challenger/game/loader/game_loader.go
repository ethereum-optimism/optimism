package loader

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
)

var (
	ErrMissingBlockNumber = errors.New("game loader missing block number")
)

// MinimalDisputeGameFactoryCaller is a minimal interface around [bindings.DisputeGameFactoryCaller].
// This needs to be updated if the [bindings.DisputeGameFactoryCaller] interface changes.
type MinimalDisputeGameFactoryCaller interface {
	GetGameCount(ctx context.Context, blockNum uint64) (uint64, error)
	GetGame(ctx context.Context, idx uint64, blockNum uint64) (types.GameMetadata, error)
}

type GameLoader struct {
	caller MinimalDisputeGameFactoryCaller
}

// NewGameLoader creates a new services that can be used to fetch on chain dispute games.
func NewGameLoader(caller MinimalDisputeGameFactoryCaller) *GameLoader {
	return &GameLoader{
		caller: caller,
	}
}

// FetchAllGamesAtBlock fetches all dispute games from the factory at a given block number.
func (l *GameLoader) FetchAllGamesAtBlock(ctx context.Context, earliestTimestamp uint64, blockNumber uint64) ([]types.GameMetadata, error) {
	gameCount, err := l.caller.GetGameCount(ctx, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game count: %w", err)
	}

	games := make([]types.GameMetadata, 0, gameCount)
	for i := gameCount; i > 0; i-- {
		game, err := l.caller.GetGame(ctx, i-1, blockNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch game at index %d: %w", i-1, err)
		}
		if game.Timestamp < earliestTimestamp {
			break
		}
		games = append(games, game)
	}

	return games, nil
}
