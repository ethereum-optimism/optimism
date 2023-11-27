package stack

import (
	"context"

	oppio "github.com/ethereum-optimism/optimism/op-program/io"
)

type Sink func(preimageRW oppio.FileChannel, hintRW oppio.FileChannel) (Stoppable, error)

type Source func() (preimageRW oppio.FileChannel, hintRW oppio.FileChannel, stop Stoppable, err error)

type Stoppable interface {
	// Stop stops the service gracefully. If ctx is cancelled, the service is killed.
	Stop(ctx context.Context) error
}

type StopFn func(ctx context.Context) error

func (fn StopFn) Stop(ctx context.Context) error {
	return fn(ctx)
}
