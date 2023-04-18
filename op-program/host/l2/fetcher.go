package l2

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/eth"
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
	head        eth.BlockInfo
	blockSource BlockSource
	callContext CallContext
}

func NewFetchingL2Oracle(ctx context.Context, logger log.Logger, l2Url string, l2Head common.Hash) (*FetchingL2Oracle, error) {
	rpcClient, err := rpc.Dial(l2Url)
	if err != nil {
		return nil, err
	}
	ethClient := ethclient.NewClient(rpcClient)
	head, err := ethClient.HeaderByHash(ctx, l2Head)
	if err != nil {
		return nil, fmt.Errorf("retrieve l2 head %v: %w", l2Head, err)
	}
	return &FetchingL2Oracle{
		ctx:         ctx,
		logger:      logger,
		head:        eth.HeaderBlockInfo(head),
		blockSource: ethClient,
		callContext: rpcClient,
	}, nil
}

func (o *FetchingL2Oracle) NodeByHash(hash common.Hash) []byte {
	// MPT nodes are stored as the hash of the node (with no prefix)
	node, err := o.dbGet(hash.Bytes())
	if err != nil {
		panic(err)
	}
	return node
}

func (o *FetchingL2Oracle) CodeByHash(hash common.Hash) []byte {
	// First try retrieving with the new code prefix
	code, err := o.dbGet(append(rawdb.CodePrefix, hash.Bytes()...))
	if err != nil {
		// Fallback to the legacy un-prefixed version
		code, err = o.dbGet(hash.Bytes())
		if err != nil {
			panic(err)
		}
	}
	return code
}

func (o *FetchingL2Oracle) dbGet(key []byte) ([]byte, error) {
	var node hexutil.Bytes
	err := o.callContext.CallContext(o.ctx, &node, "debug_dbGet", hexutil.Encode(key))
	if err != nil {
		return nil, fmt.Errorf("fetch node %s: %w", hexutil.Encode(key), err)
	}
	return node, nil
}

func (o *FetchingL2Oracle) BlockByHash(blockHash common.Hash) *types.Block {
	block, err := o.blockSource.BlockByHash(o.ctx, blockHash)
	if err != nil {
		panic(fmt.Errorf("fetch block %s: %w", blockHash.Hex(), err))
	}
	if block.NumberU64() > o.head.NumberU64() {
		panic(fmt.Errorf("fetched block %v number %d above head block number %d", blockHash, block.NumberU64(), o.head.NumberU64()))
	}
	return block
}
