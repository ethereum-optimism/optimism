package frontend

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type SyncBackend interface {
	ChainID(ctx context.Context) (eth.ChainID, error)
	Init(ctx context.Context) (string, error)
	GetBlockByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	GetBlockByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	GetBlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]*types.Receipt, error)
}

type SyncFrontend struct {
	b SyncBackend
}

var _ apis.SyncTester = (*SyncFrontend)(nil)

func NewSyncFrontend(b SyncBackend) *SyncFrontend {
	return &SyncFrontend{b: b}
}

func (s *SyncFrontend) ChainID(ctx context.Context) (eth.ChainID, error) {
	return s.b.ChainID(ctx)
}

func (s *SyncFrontend) Init(ctx context.Context) (string, error) {
	return s.b.Init(ctx)
}

func (s *SyncFrontend) GetBlockByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return s.b.GetBlockByNumber(ctx, number)
}

func (s *SyncFrontend) GetBlockByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return s.b.GetBlockByHash(ctx, hash)
}

func (s *SyncFrontend) GetBlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]*types.Receipt, error) {
	return s.b.GetBlockReceipts(ctx, blockNrOrHash)
}
