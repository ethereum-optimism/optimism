package backoff

import (
	"context"
	"fmt"
	"time"
)

// ErrFailedPermanently is an error raised by Do when the
// underlying Operation has been retried maxAttempts times.
type ErrFailedPermanently struct {
	attempts int
	LastErr  error
}

func (e *ErrFailedPermanently) Error() string {
	return fmt.Sprintf("operation failed permanently after %d attempts: %v", e.attempts, e.LastErr)
}

type pair[T, U any] struct {
	a T
	b U
}

func Do2[T, U any](ctx context.Context, maxAttempts int, strategy Strategy, op func() (T, U, error)) (T, U, error) {
	f := func() (pair[T, U], error) {
		a, b, err := op()
		return pair[T, U]{a, b}, err
	}
	res, err := Do(ctx, maxAttempts, strategy, f)
	return res.a, res.b, err
}

// Do performs the provided Operation up to maxAttempts times
// with delays in between each retry according to the provided
// Strategy.
func Do[T any](ctx context.Context, maxAttempts int, strategy Strategy, op func() (T, error)) (T, error) {
	var empty T
	if maxAttempts < 1 {
		return empty, fmt.Errorf("need at least 1 attempt to run op, but have %d max attempts", maxAttempts)
	}
	var attempt int

	reattemptCh := make(chan struct{}, 1)
	doReattempt := func() {
		reattemptCh <- struct{}{}
	}
	doReattempt()

	for {
		select {
		case <-ctx.Done():
			return empty, ctx.Err()
		case <-reattemptCh:
			attempt++
			ret, err := op()
			if err == nil {
				return ret, nil
			}

			if attempt == maxAttempts {
				return empty, &ErrFailedPermanently{
					attempts: maxAttempts,
					LastErr:  err,
				}
			}
			time.AfterFunc(strategy.Duration(attempt-1), doReattempt)
		}

	}
}
