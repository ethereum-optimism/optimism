package dial

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

// EthClientInterface is an interface for providing an ethclient.Client
// It does not describe all of the functions an ethclient.Client has, only the ones used by callers of the L2 Providers
type EthClientInterface interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)

	Close()
}
