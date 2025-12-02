package geth

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

var ErrNotFound = errors.New("not found")

// FindBlock finds the first block for which the predicate [pred] matches
// and returns it. It starts at [from] and iterates until [to], inclusively,
// using the provided [client]. It supports both search directions, forwards
// and backwards.
func FindBlock(client *ethclient.Client,
	from, to int, timeout time.Duration,
	pred func(*types.Block) (bool, error),
) (*types.Block, error) {
	dir := 1
	if from > to {
		dir = -1
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for n := from; ; n += dir {
		b, err := client.BlockByNumber(ctx, big.NewInt(int64(n)))
		if err != nil {
			return nil, fmt.Errorf("fetching block[%d]: %w", n, err)
		}
		ok, err := pred(b)
		if err != nil {
			return nil, fmt.Errorf("predicate error[%d]: %w", n, err)
		} else if ok {
			return b, nil
		}

		// include n in range
		if n == to {
			break
		}
	}

	return nil, ErrNotFound
}
