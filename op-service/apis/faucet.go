package apis

import (
	"context"

	"github.com/HashKeyChain/verse/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

type Faucet interface {
	ChainID(ctx context.Context) (eth.ChainID, error)
	RequestETH(ctx context.Context, addr common.Address, amount eth.ETH) error
	Balance(ctx context.Context) (eth.ETH, error)
}
