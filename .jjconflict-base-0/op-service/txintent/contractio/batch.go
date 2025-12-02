package contractio

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
)

// BatchableCall represents a contract read call (eth_call) that can be executed as part of a larger batch of requests
// using `batching.MultiCaller`
type BatchableCall[T any] struct {
	typedCall bindings.TypedCall[T]
}

var _ batching.Call = (*BatchableCall[any])(nil)

func NewBatchableCall[T any](typedCall bindings.TypedCall[T]) *BatchableCall[T] {
	return &BatchableCall[T]{
		typedCall: typedCall,
	}
}

func (c *BatchableCall[T]) ToBatchElemCreator() (batching.BatchElementCreator, error) {
	args, err := c.callArgs()
	if err != nil {
		return nil, err
	}
	f := func(block rpcblock.Block) (any, rpc.BatchElem) {
		out := new(hexutil.Bytes)
		return out, rpc.BatchElem{
			Method: "eth_call",
			Args:   []interface{}{args, block.ArgValue()},
			Result: &out,
		}
	}
	return f, nil
}

func (c *BatchableCall[T]) HandleResult(result interface{}) (*batching.CallResult, error) {
	hex := *result.(*hexutil.Bytes)
	out, err := c.typedCall.DecodeOutput(hex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode output: %w", err)
	}
	return batching.NewCallResult([]any{out}), nil
}

func (c *BatchableCall[T]) input() ([]byte, error) {
	return c.typedCall.EncodeInputLambda()
}

func (c *BatchableCall[T]) callArgs() (interface{}, error) {
	data, err := c.input()
	if err != nil {
		return nil, fmt.Errorf("failed to encode input data: %w", err)
	}

	to, err := c.typedCall.To()
	if err != nil {
		return nil, fmt.Errorf("failed to determine contract address: %w", err)
	}

	arg := map[string]interface{}{
		"to":    to,
		"input": hexutil.Bytes(data),
	}
	return arg, nil
}
