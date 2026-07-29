package batching

import (
	"context"

	"github.com/ethereum/go-ethereum/rpc"
)

type BatchCallContextFn func(ctx context.Context, b []rpc.BatchElem) error

type CallContextFn func(ctx context.Context, result any, method string, args ...any) error
