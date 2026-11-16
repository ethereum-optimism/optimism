package backend

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// ReadOnlyELBackend defines the minimal, read-only execution layer
// interface used by the sync tester and its mock backends.
// The interface exposes two flavors of accessors:
//   - JSON-returning methods (GetBlockByNumberJSON, GetBlockByHashJSON,
//     GetBlockReceiptsJSON) which return the raw RPC payload exactly as
//     delivered by the EL. These are useful for relaying the response from
//     the read-only exec layer directly, without dropping OP Stack fields.
//   - Typed methods (PayloadByNumber, PayloadByHash) which return the block
//     as an execution payload, with transactions in their binary encoding,
//     for structured inspection in code. The envelopes may be shared and
//     cached by the implementation: callers must not modify them.
//   - Additionally, ChainId.
//
// Implementation forwards RPC calls to the EL. For testing, a mock
// implementation can be provided to return deterministic values without
// requiring a live execution layer node.
type ReadOnlyELBackend interface {
	GetBlockByNumberJSON(ctx context.Context, number rpc.BlockNumber, fullTx bool) (json.RawMessage, error)
	GetBlockByHashJSON(ctx context.Context, hash common.Hash, fullTx bool) (json.RawMessage, error)
	GetBlockReceiptsJSON(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (json.RawMessage, error)
	PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error)
	PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayloadEnvelope, error)
	ChainId(ctx context.Context) (hexutil.Big, error)
}

var _ ReadOnlyELBackend = (*ELReader)(nil)

type ELReader struct {
	el  client.RPC
	eth *sources.EthClient
}

func NewELReader(logger log.Logger, el client.RPC) (*ELReader, error) {
	ethClient, err := sources.NewEthClient(el, logger, nil, sources.DefaultEthClientConfig(10))
	if err != nil {
		return nil, fmt.Errorf("failed to create EL client: %w", err)
	}
	return &ELReader{el: el, eth: ethClient}, nil
}

func (g *ELReader) GetBlockByNumberJSON(ctx context.Context, number rpc.BlockNumber, fullTx bool) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := g.el.CallContext(ctx, &raw, "eth_getBlockByNumber", number, fullTx); err != nil {
		return nil, err
	}
	return raw, nil
}

func (g *ELReader) GetBlockByHashJSON(ctx context.Context, hash common.Hash, fullTx bool) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := g.el.CallContext(ctx, &raw, "eth_getBlockByHash", hash, fullTx); err != nil {
		return nil, err
	}
	return raw, nil
}

func (g *ELReader) GetBlockReceiptsJSON(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := g.el.CallContext(ctx, &raw, "eth_getBlockReceipts", blockNrOrHash); err != nil {
		return nil, err
	}
	return raw, nil
}

func (g *ELReader) PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error) {
	return g.eth.PayloadByNumber(ctx, number)
}

func (g *ELReader) PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayloadEnvelope, error) {
	return g.eth.PayloadByHash(ctx, hash)
}

func (g *ELReader) ChainId(ctx context.Context) (hexutil.Big, error) {
	chainID, err := g.eth.ChainID(ctx)
	if err != nil {
		return hexutil.Big{}, err
	}
	return hexutil.Big(*chainID), nil
}
