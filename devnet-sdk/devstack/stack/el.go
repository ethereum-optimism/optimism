package stack

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/core/types"
)

type EthClient interface {
	SendTransaction(ctx context.Context, tx *types.Transaction) error
	// more methods may be added
}

type ELNode interface {
	Common
	ChainID() eth.ChainID
	EthClient() EthClient
}
