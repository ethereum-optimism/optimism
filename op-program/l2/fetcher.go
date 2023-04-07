package l2

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
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
	ctx         context.Context
	logger      log.Logger
	blockSource BlockSource
	callContext CallContext
}

func NewFetchingL2Oracle(ctx context.Context, logger log.Logger, l2Url string) (*FetchingL2Oracle, error) {
	rpcClient, err := rpc.Dial(l2Url)
	if err != nil {
		return nil, err
	}
	ethClient := ethclient.NewClient(rpcClient)
	return &FetchingL2Oracle{
		ctx:         ctx,
		logger:      logger,
		blockSource: ethClient,
		callContext: rpcClient,
	}, nil
}

func (o *FetchingL2Oracle) NodeByHash(hash common.Hash) ([]byte, error) {
	// MPT nodes are stored as the hash of the node (with no prefix)
	return o.dbGet(hash.Bytes())
}

func (o *FetchingL2Oracle) CodeByHash(hash common.Hash) ([]byte, error) {
	// First try retrieving with the new code prefix
	code, err := o.dbGet(append(rawdb.CodePrefix, hash.Bytes()...))
	if err != nil {
		// Fallback to the legacy un-prefixed version
		return o.dbGet(hash.Bytes())
	}
	return code, nil
}

func (o *FetchingL2Oracle) dbGet(key []byte) ([]byte, error) {
	var node hexutil.Bytes
	err := o.callContext.CallContext(o.ctx, &node, "debug_dbGet", hexutil.Encode(key))
	if err != nil {
		return nil, fmt.Errorf("fetch node %s: %w", hexutil.Encode(key), err)
	}
	return node, nil
}

func (o *FetchingL2Oracle) BlockByHash(blockHash common.Hash) (*types.Block, error) {
	block, err := o.blockSource.BlockByHash(o.ctx, blockHash)
	if err != nil {
		return nil, fmt.Errorf("fetch block %s: %w", blockHash.Hex(), err)
	}
	return block, nil
}
