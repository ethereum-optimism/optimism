package sources

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/client"
)

// callNonNull calls method and decodes its result, returning an error if the result is JSON
// null. The methods using it never answer null on success: absence is reported as an error.
func callNonNull[T any](ctx context.Context, rpc client.RPC, method string, args ...any) (*T, error) {
	var out *T
	if err := rpc.CallContext(ctx, &out, method, args...); err != nil {
		return nil, err
	}
	if out == nil {
		return nil, fmt.Errorf("%s returned null", method)
	}
	return out, nil
}
