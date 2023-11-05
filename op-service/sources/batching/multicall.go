package batching

import (
	"context"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

var DefaultBatchSize = 100

type EthRpc interface {
	CallContext(ctx context.Context, out interface{}, method string, args ...interface{}) error
	BatchCallContext(ctx context.Context, b []rpc.BatchElem) error
}

type MultiCaller struct {
	rpc       EthRpc
	batchSize int
}

func NewMultiCaller(rpc EthRpc, batchSize int) *MultiCaller {
	return &MultiCaller{
		rpc:       rpc,
		batchSize: batchSize,
	}
}

func (m *MultiCaller) SingleCall(ctx context.Context, block Block, call *ContractCall) (*CallResult, error) {
	results, err := m.Call(ctx, block, call)
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

func (m *MultiCaller) Call(ctx context.Context, block Block, calls ...*ContractCall) ([]*CallResult, error) {
	keys := make([]interface{}, len(calls))
	for i := 0; i < len(calls); i++ {
		args, err := calls[i].ToCallArgs()
		if err != nil {
			return nil, err
		}
		keys[i] = args
	}
	fetcher := NewIterativeBatchCall[interface{}, *hexutil.Bytes](
		keys,
		func(args interface{}) (*hexutil.Bytes, rpc.BatchElem) {
			out := new(hexutil.Bytes)
			return out, rpc.BatchElem{
				Method: "eth_call",
				Args:   []interface{}{args, block.value},
				Result: &out,
			}
		},
		m.rpc.BatchCallContext,
		m.rpc.CallContext,
		m.batchSize)
	for {
		if err := fetcher.Fetch(ctx); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to fetch claims: %w", err)
		}
	}
	results, err := fetcher.Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get batch call results: %w", err)
	}

	callResults := make([]*CallResult, len(results))
	for i, result := range results {
		call := calls[i]
		out, err := call.Unpack(*result)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack result: %w", err)
		}
		callResults[i] = out
	}
	return callResults, nil
}

// Block represents the block ref value in RPC calls.
// It can be either a label (e.g. latest), a block number or block hash.
type Block struct {
	value any
}

func (b Block) ArgValue() any {
	return b.value
}

var (
	BlockPending   = Block{"pending"}
	BlockLatest    = Block{"latest"}
	BlockSafe      = Block{"safe"}
	BlockFinalized = Block{"finalized"}
)

// BlockByNumber references a canonical block by number.
func BlockByNumber(blockNum uint64) Block {
	return Block{rpc.BlockNumber(blockNum)}
}

// BlockByHash references a block by hash. Canonical or non-canonical blocks may be referenced.
func BlockByHash(hash common.Hash) Block {
	return Block{rpc.BlockNumberOrHashWithHash(hash, false)}
}
