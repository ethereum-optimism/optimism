package frontend

import (
	"context"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type EthBackend interface {
	GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (json.RawMessage, error)
	GetBlockByHash(ctx context.Context, hash common.Hash, fullTx bool) (json.RawMessage, error)
	GetBlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]*types.Receipt, error)
	ChainId(ctx context.Context) (hexutil.Big, error)
}

type EthFrontend struct {
	b EthBackend
}

func NewEthFrontend(b EthBackend) *EthFrontend {
	return &EthFrontend{b: b}
}

func (e *EthFrontend) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (json.RawMessage, error) {
	return e.b.GetBlockByNumber(ctx, number, fullTx)
}

func (e *EthFrontend) GetBlockByHash(ctx context.Context, hash common.Hash, fullTx bool) (json.RawMessage, error) {
	return e.b.GetBlockByHash(ctx, hash, fullTx)
}

func (e *EthFrontend) GetBlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]*types.Receipt, error) {
	return e.b.GetBlockReceipts(ctx, blockNrOrHash)
}

func (e *EthFrontend) ChainId(ctx context.Context) (hexutil.Big, error) {
	return e.b.ChainId(ctx)
}
