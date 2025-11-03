// Package context provides context and timeout utilities for op-node.
package context

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// TimeoutConfig configures timeout behavior.
type TimeoutConfig struct {
	L1RequestTimeout time.Duration
	L2RequestTimeout time.Duration
	ConfigReloadTimeout time.Duration
	DefaultTimeout time.Duration
}

// DefaultTimeoutConfig returns default timeout configuration.
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		L1RequestTimeout: 30 * time.Second,
		L2RequestTimeout: 10 * time.Second,
		ConfigReloadTimeout: 5 * time.Second,
		DefaultTimeout: 15 * time.Second,
	}
}

// WithL1Timeout creates a context with L1 request timeout.
func WithL1Timeout(ctx context.Context, logger log.Logger, config TimeoutConfig) (context.Context, context.CancelFunc) {
	return WithTimeout(ctx, logger, config.L1RequestTimeout, "L1 request")
}

// WithL2Timeout creates a context with L2 request timeout.
func WithL2Timeout(ctx context.Context, logger log.Logger, config TimeoutConfig) (context.Context, context.CancelFunc) {
	return WithTimeout(ctx, logger, config.L2RequestTimeout, "L2 request")
}

// WithConfigReloadTimeout creates a context with config reload timeout.
func WithConfigReloadTimeout(ctx context.Context, logger log.Logger, config TimeoutConfig) (context.Context, context.CancelFunc) {
	return WithTimeout(ctx, logger, config.ConfigReloadTimeout, "config reload")
}

// WithTimeout creates a context with a specific timeout.
func WithTimeout(ctx context.Context, logger log.Logger, timeout time.Duration, operation string) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	
	newCtx, cancel := context.WithTimeout(ctx, timeout)
	
	go func() {
		<-newCtx.Done()
		if newCtx.Err() == context.DeadlineExceeded {
			logger.Warn("Operation timeout exceeded", "operation", operation, "timeout", timeout)
		}
	}()
	
	return newCtx, cancel
}

// WithDefaultTimeout creates a context with default timeout.
func WithDefaultTimeout(ctx context.Context, logger log.Logger, config TimeoutConfig) (context.Context, context.CancelFunc) {
	return WithTimeout(ctx, logger, config.DefaultTimeout, "default operation")
}

