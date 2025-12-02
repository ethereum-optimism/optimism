package backend

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// ReadOnlyELBackend defines the minimal, read-only execution layer
// interface used by the sync tester and its mock backends.
// The interface exposes two flavors of block accessors:
//   - JSON-returning methods (GetBlockByNumberJSON, GetBlockByHashJSON)
//     which return the raw RPC payload exactly as delivered by the EL.
//     These are useful for relaying the response from read-only exec layer directly
//   - Typed methods (GetBlockByNumber, GetBlockByHash) which decode
//     the RPC response into geth *types.Block for structured
//     inspection in code.
//   - Additional helpers include GetBlockReceipts and ChainId
//
// Implementation wraps ethclient.Client to forward RPC
// calls. For testing, a mock implementation can be provided to return
// deterministic values without requiring a live execution layer node.
type ReadOnlyELBackend interface {
	GetBlockByNumberJSON(ctx context.Context, number rpc.BlockNumber, fullTx bool) (json.RawMessage, error)
	GetBlockByHashJSON(ctx context.Context, hash common.Hash, fullTx bool) (json.RawMessage, error)
	GetBlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error)
	GetBlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
	GetBlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]*types.Receipt, error)
	ChainId(ctx context.Context) (hexutil.Big, error)
}

var _ ReadOnlyELBackend = (*ELReader)(nil)

type ELReader struct {
	c *ethclient.Client
}

func NewELReader(c *ethclient.Client) *ELReader {
	return &ELReader{c: c}
}

func (g *ELReader) GetBlockByNumberJSON(ctx context.Context, number rpc.BlockNumber, fullTx bool) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := g.c.Client().CallContext(ctx, &raw, "eth_getBlockByNumber", number, fullTx); err != nil {
		return nil, err
	}
	return raw, nil
}

func (g *ELReader) GetBlockByHashJSON(ctx context.Context, hash common.Hash, fullTx bool) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := g.c.Client().CallContext(ctx, &raw, "eth_getBlockByHash", hash, fullTx); err != nil {
		return nil, err
	}
	return raw, nil
}

func (g *ELReader) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	return g.c.BlockByNumber(ctx, big.NewInt(number.Int64()))
}

func (g *ELReader) GetBlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return g.c.BlockByHash(ctx, hash)
}

func (g *ELReader) GetBlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]*types.Receipt, error) {
	return g.c.BlockReceipts(ctx, blockNrOrHash)
}

func (g *ELReader) ChainId(ctx context.Context) (hexutil.Big, error) {
	chainID, err := g.c.ChainID(ctx)
	if err != nil {
		return hexutil.Big{}, err
	}
	return hexutil.Big(*chainID), nil
}
