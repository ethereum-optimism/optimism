package batching

import (
	"context"
	"fmt"
	"math/big"
)

// ReadArray uses batch calls to load all entries from an array.
// countCall is used to retrieve the current array length, then getCall is used to create calls for each element
// which are sent in a batch call.
// The returned *CallResult slice, contains a result for each entry in the array, in the same order as in the contract.
func ReadArray(ctx context.Context, caller *MultiCaller, block Block, countCall *ContractCall, getCall func(i *big.Int) *ContractCall) ([]*CallResult, error) {
	result, err := caller.SingleCall(ctx, block, countCall)
	if err != nil {
		return nil, fmt.Errorf("failed to load array length: %w", err)
	}
	count := result.GetBigInt(0).Uint64()
	calls := make([]*ContractCall, count)
	for i := uint64(0); i < count; i++ {
		calls[i] = getCall(new(big.Int).SetUint64(i))
	}
	results, err := caller.Call(ctx, block, calls...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch array data: %w", err)
	}
	return results, nil
}
