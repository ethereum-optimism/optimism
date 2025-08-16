package dsl

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/retry"
)

// RetryWithBackoff performs an operation with exponential backoff and increased attempts
func RetryWithBackoff(ctx context.Context, operation func() error) error {
	return retry.Do0(ctx, 5, retry.Exponential(), func() error {
		return operation()
	})
}

// WaitForServiceReady waits for a service to be ready with a more generous timeout
func WaitForServiceReady(ctx context.Context, check func() error) error {
	return retry.Do0(ctx, 10, &retry.FixedStrategy{Dur: 2 * time.Second}, func() error {
		return check()
	})
}
