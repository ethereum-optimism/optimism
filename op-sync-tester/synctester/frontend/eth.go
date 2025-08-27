package frontend

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type EthBackend interface {
	GetBlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	GetBlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
	GetBlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]*types.Receipt, error)
	ChainId(ctx context.Context) (eth.ChainID, error)
}

type EthFrontend struct {
	b EthBackend
}

func NewEthFrontend(b EthBackend) *EthFrontend {
	return &EthFrontend{b: b}
}

func (e *EthFrontend) GetBlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return e.b.GetBlockByNumber(ctx, number)
}

func (e *EthFrontend) GetBlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return e.b.GetBlockByHash(ctx, hash)
}

func (e *EthFrontend) GetBlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]*types.Receipt, error) {
	return e.b.GetBlockReceipts(ctx, blockNrOrHash)
}

func (e *EthFrontend) ChainId(ctx context.Context) (eth.ChainID, error) {
	return e.b.ChainId(ctx)
}
