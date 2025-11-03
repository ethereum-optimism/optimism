// Package retry provides retry utilities with exponential backoff for L1/L2 operations.
package retry

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// RetryConfig configures retry behavior.
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// DefaultRetryConfig returns a default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}
}

// RetryFunc is a function that may be retried.
type RetryFunc func(ctx context.Context) error

// RetryWithBackoff retries a function with exponential backoff.
func RetryWithBackoff(ctx context.Context, logger log.Logger, config RetryConfig, fn RetryFunc) error {
	var lastErr error
	
	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff
			delay := time.Duration(float64(config.InitialDelay) * pow(config.Multiplier, float64(attempt-1)))
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
			
			logger.Debug("Retrying operation", 
				"attempt", attempt+1, 
				"max_attempts", config.MaxAttempts,
				"delay", delay,
				"last_error", lastErr)
			
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled: %w", ctx.Err())
			case <-time.After(delay):
			}
		}
		
		err := fn(ctx)
		if err == nil {
			if attempt > 0 {
				logger.Info("Operation succeeded after retry", "attempt", attempt+1)
			}
			return nil
		}
		
		lastErr = err
		
		if attempt < config.MaxAttempts-1 {
			logger.Warn("Operation failed, will retry", 
				"attempt", attempt+1,
				"error", err)
		}
	}
	
	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// RetryL1Operation retries an L1 operation with exponential backoff.
func RetryL1Operation(ctx context.Context, logger log.Logger, fn RetryFunc) error {
	config := DefaultRetryConfig()
	config.MaxAttempts = 5
	config.InitialDelay = 200 * time.Millisecond
	return RetryWithBackoff(ctx, logger, config, fn)
}

// RetryL2Operation retries an L2 operation with exponential backoff.
func RetryL2Operation(ctx context.Context, logger log.Logger, fn RetryFunc) error {
	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.InitialDelay = 100 * time.Millisecond
	return RetryWithBackoff(ctx, logger, config, fn)
}

// pow calculates base^exp efficiently.
func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

