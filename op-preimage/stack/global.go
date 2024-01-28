package stack

import (
	"context"
	"errors"
	"fmt"
	"io"

	"golang.org/x/sync/errgroup"

	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	oppio "github.com/ethereum-optimism/optimism/op-program/io"
)

func GlobalSource() Source {
	return Source(func() (preimageRW oppio.FileChannel, hintRW oppio.FileChannel, stop Stoppable, err error) {
		preimageChan := preimage.CreatePreimageChannel()
		hinterChan := preimage.CreateHinterChannel()
		stopper := StopFn(func(ctx context.Context) error {
			var result error
			if err := preimageChan.Close(); err != nil {
				result = errors.Join(result, fmt.Errorf("failed to close preimage channels: %w", err))
			}
			if err := hinterChan.Close(); err != nil {
				result = errors.Join(result, fmt.Errorf("failed to close hint channels: %w", err))
			}
			return result
		})
		return preimageChan, hinterChan, stopper, nil
	})
}

func GlobalSink() Sink {
	return Sink(func(preimageRW oppio.FileChannel, hintRW oppio.FileChannel) (Stoppable, error) {
		globalPreimageChan := preimage.CreatePreimageChannel()
		globalHinterChan := preimage.CreateHinterChannel()
		var errGrp errgroup.Group
		errGrp.Go(func() error {
			_, err := io.Copy(globalPreimageChan, preimageRW)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return fmt.Errorf("failed to pipe preimage data into global channel: %w", err)
			}
			return nil
		})
		errGrp.Go(func() error {
			_, err := io.Copy(globalHinterChan, hintRW)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return fmt.Errorf("failed to pipe hint data into global channel: %w", err)
			}
			return nil
		})
		errGrp.Go(func() error {
			_, err := io.Copy(preimageRW, globalPreimageChan)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return fmt.Errorf("failed to pipe preimage data out of global channel: %w", err)
			}
			return nil
		})
		errGrp.Go(func() error {
			_, err := io.Copy(hintRW, globalHinterChan)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return fmt.Errorf("failed to pipe hint data out of global channel: %w", err)
			}
			return nil
		})
		return StopFn(func(ctx context.Context) error {
			var result error
			if err := globalPreimageChan.Close(); err != nil {
				result = errors.Join(result, fmt.Errorf("failed to close global preimage channel: %w", err))
			}
			if err := globalHinterChan.Close(); err != nil {
				result = errors.Join(result, fmt.Errorf("failed to close global hint channel: %w", err))
			}
			if err := errGrp.Wait(); err != nil {
				result = errors.Join(result, fmt.Errorf("failed to await exit of copy processes: %w", err))
			}
			return result
		}), nil
	})
}
