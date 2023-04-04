package l2

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type BlockSource interface {
	BlockByHash(ctx context.Context, blockHash common.Hash) (*types.Block, error)
}

type CallContext interface {
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
}

type FetchingL2Oracle struct {
	logger      log.Logger
	blockSource BlockSource
	callContext CallContext
}

func NewFetchingL2Oracle(logger log.Logger, l2Url string) (*FetchingL2Oracle, error) {
	rpcClient, err := rpc.Dial(l2Url)
	if err != nil {
		return nil, err
	}
	ethClient := ethclient.NewClient(rpcClient)
	return &FetchingL2Oracle{
		logger:      logger,
		blockSource: ethClient,
		callContext: rpcClient,
	}, nil
}

func (s FetchingL2Oracle) NodeByHash(ctx context.Context, nodeHash common.Hash) ([]byte, error) {
	var node hexutil.Bytes
	err := s.callContext.CallContext(ctx, &node, "debug_dbGet", nodeHash.Hex())
	if err != nil {
		return nil, fmt.Errorf("fetch node %s: %w", nodeHash.Hex(), err)
	}
	return node, nil
}

func (s FetchingL2Oracle) BlockByHash(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	block, err := s.blockSource.BlockByHash(ctx, blockHash)
	if err != nil {
		return nil, fmt.Errorf("fetch block %s: %w", blockHash.Hex(), err)
	}
	return block, nil
}
