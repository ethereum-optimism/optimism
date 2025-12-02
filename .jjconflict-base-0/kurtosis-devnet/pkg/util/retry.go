package util

import (
	"context"
	"fmt"
	"log"
	"time"
)

// WithRetry executes a function with exponential backoff retry logic
// This is specifically designed for handling gRPC connection timeouts with Kurtosis
func WithRetry[T any](ctx context.Context, operation string, fn func() (T, error)) (T, error) {
	var result T
	var err error

	for attempt := 1; attempt <= 3; attempt++ {
		result, err = fn()
		if err == nil {
			if attempt > 1 {
				log.Printf("✅ Successfully completed %s on attempt %d", operation, attempt)
			}
			return result, nil
		}

		log.Printf("❌ Attempt %d failed for %s: %v", attempt, operation, err)

		if attempt < 3 {
			sleepDuration := time.Duration(attempt*2) * time.Second
			log.Printf("⏳ Retrying %s in %v...", operation, sleepDuration)
			time.Sleep(sleepDuration)
		}
	}

	return result, fmt.Errorf("%s failed after 3 attempts: %w", operation, err)
}
