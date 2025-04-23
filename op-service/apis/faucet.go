package apis

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

type Faucet interface {
	ChainID(ctx context.Context) (eth.ChainID, error)
	RequestETH(ctx context.Context, addr common.Address, amount eth.ETH) error
}
