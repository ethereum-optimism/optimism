package contractio

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Write receives a TypedCall and uses to plan transaction, and attempts to write.
func Write[O any](call bindings.TypedCall[O], ctx context.Context, opts ...txplan.Option) (*types.Receipt, error) {
	plan, err := Plan(call)
	if err != nil {
		return nil, err
	}
	tx := txplan.NewPlannedTx(plan, txplan.Combine(opts...))
	receipt, err := tx.Included.Eval(ctx)
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

// Read receives a TypedCall and uses to plan transaction, and attempts to read.
func Read[O any](view bindings.TypedCall[O], ctx context.Context, opts ...txplan.Option) (O, error) {
	plan, err := Plan(view)
	if err != nil {
		return *new(O), err
	}
	client := view.Client()
	tx := txplan.NewPlannedTx(
		plan,
		txplan.WithAgainstLatestBlock(client),
		txplan.WithReader(client),
		// use default sender as null
		txplan.WithSender(common.Address{}),
		txplan.Combine(opts...),
	)
	res, err := tx.Read.Eval(ctx)
	if err != nil {
		return *new(O), err
	}
	decoded, err := view.DecodeOutput(res)
	if err != nil {
		return *new(O), err
	}
	return decoded, nil
}

// ReadArray uses batch calls to load all entries from an array.
func ReadArray[T any](ctx context.Context, caller *batching.MultiCaller, countCall bindings.TypedCall[*big.Int], elemCall func(i *big.Int) bindings.TypedCall[T]) ([]T, error) {
	block := rpcblock.Latest

	countResult, err := Read(countCall, ctx)
	if err != nil {
		return nil, fmt.Errorf("error reading array size: %w", err)
	}

	count := countResult.Uint64()
	calls := make([]batching.Call, count)
	for i := uint64(0); i < count; i++ {
		typedCall := elemCall(new(big.Int).SetUint64(i))
		calls[i] = NewBatchableCall(typedCall)
	}

	callResults, err := caller.Call(ctx, block, calls...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch array data: %w", err)
	}

	// Convert results to expected type
	var results []T
	for _, callResult := range callResults {
		result, ok := callResult.Get(0).(T)
		if !ok {
			return nil, fmt.Errorf("failed to cast result: %v", callResult)
		}
		results = append(results, result)
	}

	return results, nil
}

func Plan[O any](call bindings.TypedCall[O]) (txplan.Option, error) {
	target, err := call.To()
	if err != nil {
		return nil, err
	}
	calldata, err := call.EncodeInput()
	if err != nil {
		return nil, err
	}
	tx := txplan.Combine(
		txplan.WithData(calldata),
		txplan.WithTo(target),
	)
	return tx, nil
}
