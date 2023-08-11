package fault

import (
	"context"
	"errors"
	"fmt"
	"math/big"

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
		Proxy     common.Address
		Timestamp *big.Int
	}, error)
}

type FaultDisputeGame struct {
	Proxy     common.Address
	Timestamp *big.Int
}

// GameLoader is a minimal interface for fetching on chain dispute games.
type GameLoader interface {
	FetchAllGamesAtBlock(ctx context.Context) ([]FaultDisputeGame, error)
}

type gameLoader struct {
	caller MinimalDisputeGameFactoryCaller
}

// NewGameLoader creates a new services that can be used to fetch on chain dispute games.
func NewGameLoader(caller MinimalDisputeGameFactoryCaller) *gameLoader {
	return &gameLoader{
		caller: caller,
	}
}

// FetchAllGamesAtBlock fetches all dispute games from the factory at a given block number.
func (l *gameLoader) FetchAllGamesAtBlock(ctx context.Context, blockNumber *big.Int) ([]FaultDisputeGame, error) {
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

	games := make([]FaultDisputeGame, gameCount.Uint64())
	for i := uint64(0); i < gameCount.Uint64(); i++ {
		game, err := l.caller.GameAtIndex(callOpts, big.NewInt(int64(i)))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch game at index %d: %w", i, err)
		}

		games[i] = game
	}

	return games, nil
}
