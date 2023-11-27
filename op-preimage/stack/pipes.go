package stack

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	oppio "github.com/ethereum-optimism/optimism/op-program/io"
)

func MiddlewarePipes() (pClientRW, pHostRW, hClientRW, hHostRW oppio.FileChannel, stop Stoppable, err error) {
	// Setup client I/O for preimage oracle interaction
	pClientRW, pHostRW, err = oppio.CreateBidirectionalChannel()
	if err != nil {
		err = fmt.Errorf("failed to create preimage pipe: %w", err)
		return
	}

	// Setup client I/O for hint comms
	hClientRW, hHostRW, err = oppio.CreateBidirectionalChannel()
	if err != nil {
		err = fmt.Errorf("failed to create hints pipe: %w", err)
		return
	}

	stop = StopFn(func(ctx context.Context) error {
		var result error
		if err := pClientRW.Close(); err != nil && !errors.Is(err, fs.ErrClosed) {
			result = errors.Join(result, fmt.Errorf("failed to close preimage client channels: %w", err))
		}
		if err := pHostRW.Close(); err != nil && !errors.Is(err, fs.ErrClosed) {
			result = errors.Join(result, fmt.Errorf("failed to close preimage host channels: %w", err))
		}
		if err := hClientRW.Close(); err != nil && !errors.Is(err, fs.ErrClosed) {
			result = errors.Join(result, fmt.Errorf("failed to close hint client channels: %w", err))
		}
		if err := hHostRW.Close(); err != nil && !errors.Is(err, fs.ErrClosed) {
			result = errors.Join(result, fmt.Errorf("failed to close hint host channels: %w", err))
		}
		return result
	})

	return
}
