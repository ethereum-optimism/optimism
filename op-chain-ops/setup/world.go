package setup

import (
	"context"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"
)

type world struct {
	ctx context.Context
	req *require.Assertions
	log log.Logger

	cachePath string

	chains      map[uint256.Int]*chain
	superchains map[string]*superchain
}

func (w *world) CreateChain(chainID *uint256.Int) {
	_, ok := w.chains[*chainID]
	w.req.False(ok, "chain must not already exist")
	w.chains[*chainID] = &chain{
		// TODO
	}
}

func (w *world) L1(chainID *uint256.Int) L1 {
	return w.chains[*chainID]
}

func (w *world) L2(chainID *uint256.Int) L2 {
	return w.chains[*chainID]
}
