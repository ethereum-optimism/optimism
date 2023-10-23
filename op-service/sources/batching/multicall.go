package batching

import (
	"context"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

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

func (m *MultiCaller) SingleCallLatest(ctx context.Context, call *ContractCall) (*CallResult, error) {
	results, err := m.CallLatest(ctx, call)
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

func (m *MultiCaller) CallLatest(ctx context.Context, calls ...*ContractCall) ([]*CallResult, error) {
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
				Args:   []interface{}{args, "latest"},
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
		callResults[i] = &CallResult{
			out: out,
		}
	}
	return callResults, nil
}

func toCallArg(msg ethereum.CallMsg) interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.To,
	}
	if len(msg.Data) > 0 {
		arg["data"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	return arg
}
