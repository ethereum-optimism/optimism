package batching

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
)

// readArrayPageSize is the number of array elements ReadArray requests at a time.
const readArrayPageSize = 1024

// ReadArray uses batch calls to load all entries from an array.
// countCall is used to retrieve the current array length, then getCall is used to create calls for each element
// which are sent in a batch call.
// The returned *CallResult slice, contains a result for each entry in the array, in the same order as in the contract.
func ReadArray(ctx context.Context, caller *MultiCaller, block rpcblock.Block, countCall *ContractCall, getCall func(i *big.Int) *ContractCall) ([]*CallResult, error) {
	result, err := caller.SingleCall(ctx, block, countCall)
	if err != nil {
		return nil, fmt.Errorf("failed to load array length: %w", err)
	}
	countBig := result.GetBigInt(0)
	if !countBig.IsUint64() {
		return nil, fmt.Errorf("array length is not a uint64: %v", countBig)
	}
	count := bigs.Uint64Strict(countBig)
	// The length comes from the RPC, so it only bounds the loop: memory grows with the elements
	// actually returned, not with the claimed length.
	results := make([]*CallResult, 0, min(count, readArrayPageSize))
	for start := uint64(0); start < count; {
		end := start + min(readArrayPageSize, count-start)
		calls := make([]Call, 0, end-start)
		for i := start; i < end; i++ {
			calls = append(calls, getCall(new(big.Int).SetUint64(i)))
		}
		page, err := caller.Call(ctx, block, calls...)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch array data: %w", err)
		}
		results = append(results, page...)
		start = end
	}
	return results, nil
}
