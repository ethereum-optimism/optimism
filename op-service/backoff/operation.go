package backoff

import (
	"context"
	"fmt"
	"time"
)

// Operation represents an operation that will be retried
// based on some backoff strategy if it fails.
type Operation func() error

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
func Do(maxAttempts int, strategy Strategy, op Operation) error {
	return DoCtx(context.Background(), maxAttempts, strategy, op)
}

func DoCtx(ctx context.Context, maxAttempts int, strategy Strategy, op Operation) error {
	if maxAttempts < 1 {
		return fmt.Errorf("need at least 1 attempt to run op, but have %d max attempts", maxAttempts)
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
			return ctx.Err()
		case <-reattemptCh:
			attempt++
			err := op()
			if err == nil {
				return nil
			}

			if attempt == maxAttempts {
				return &ErrFailedPermanently{
					attempts: maxAttempts,
					LastErr:  err,
				}
			}
			time.AfterFunc(strategy.Duration(attempt-1), doReattempt)
		}

	}
}
