package loader

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrMissingBlockNumber = errors.New("game loader missing block number")
)

// MinimalDisputeGameFactoryCaller is a minimal interface around [bindings.DisputeGameFactoryCaller].
// This needs to be updated if the [bindings.DisputeGameFactoryCaller] interface changes.
type MinimalDisputeGameFactoryCaller interface {
	GameCount(opts *bind.CallOpts) (*big.Int, error)
	GameAtIndex(opts *bind.CallOpts, _index *big.Int) (struct {
		GameType  uint8
		Timestamp uint64
		Proxy     common.Address
	}, error)
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
func (l *GameLoader) FetchAllGamesAtBlock(ctx context.Context, earliestTimestamp uint64, blockNumber *big.Int) ([]types.GameMetadata, error) {
	if blockNumber == nil {
		return nil, ErrMissingBlockNumber
	}
	callOpts := &bind.CallOpts{
		Context:     ctx,
		BlockNumber: blockNumber,
	}
	gameCount, err := l.caller.GameCount(callOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game count: %w", err)
	}

	games := make([]types.GameMetadata, 0)
	if gameCount.Uint64() == 0 {
		return games, nil
	}
	for i := gameCount.Uint64(); i > 0; i-- {
		game, err := l.caller.GameAtIndex(callOpts, big.NewInt(int64(i-1)))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch game at index %d: %w", i, err)
		}
		if game.Timestamp < earliestTimestamp {
			break
		}
		games = append(games, game)
	}

	return games, nil
}
