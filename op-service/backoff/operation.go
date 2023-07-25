package backoff

import (
	"context"
	"fmt"
	"time"
)

// Operation represents an operation that will be retried
// based on some backoff strategy if it fails.
type Operation[T any] func() (T, error)

// ErrFailedPermanently is an error raised by Do when the
// underlying Operation has been retried maxAttempts times.
type ErrFailedPermanently struct {
	attempts int
	LastErr  error
}

func (e *ErrFailedPermanently) Error() string {
	return fmt.Sprintf("operation failed permanently after %d attempts: %v", e.attempts, e.LastErr)
}

// Do performs the provided Operation up to maxAttempts times
// with delays in between each retry according to the provided
// Strategy.
func Do[T any](maxAttempts int, strategy Strategy, op Operation[T]) error {
	return DoCtx(context.Background(), maxAttempts, strategy, op)
}

// DoResult performs the provided Operation up to maxAttempts times
// with delays in between each retry according to the provided
// Strategy. It returns the result of the operation if it succeeds,
// or an error if it fails permanently.
func DoResult[T any](maxAttempts int, strategy Strategy, op Operation[T]) (T, error) {
	return DoResultCtx(context.Background(), maxAttempts, strategy, op)
}

// DoCtx executes the provided Operation and throws away
// the result, returning just the error. As opposed to [Do],
// this function allows the caller to provide a context.
func DoCtx[T interface{}](ctx context.Context, maxAttempts int, strategy Strategy, op Operation[T]) error {
	_, err := doCtx(ctx, maxAttempts, strategy, op)
	return err
}

// DoResultCtx executes the provided Operation and throws away
// the result, returning just the error. As opposed to [DoResult],
// this function allows the caller to provide a context.
func DoResultCtx[T interface{}](ctx context.Context, maxAttempts int, strategy Strategy, op Operation[T]) (T, error) {
	return doCtx(ctx, maxAttempts, strategy, op)
}

func doCtx[T interface{}](ctx context.Context, maxAttempts int, strategy Strategy, op Operation[T]) (T, error) {
	var emptyClient T
	if maxAttempts < 1 {
		return emptyClient, fmt.Errorf("need at least 1 attempt to run op, but have %d max attempts", maxAttempts)
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
			return emptyClient, ctx.Err()
		case <-reattemptCh:
			attempt++
			t, err := op()
			if err == nil {
				return t, nil
			}
			if attempt == maxAttempts {
				return emptyClient, &ErrFailedPermanently{
					attempts: maxAttempts,
					LastErr:  err,
				}
			}
			time.AfterFunc(strategy.Duration(attempt-1), doReattempt)
		}
	}
}
