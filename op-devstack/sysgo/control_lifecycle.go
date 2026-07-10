package sysgo

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	controlledInterruptWait = 10 * time.Second
	controlledKillWait      = 5 * time.Second
)

func controlCall(fn func()) error {
	result := make(chan error, 1)
	// Run fn in its own goroutine so runtime.Goexit, e.g. from FailNow, can be reported.
	go func() {
		completed := false
		defer func() {
			if recovered := recover(); recovered != nil {
				if recoveredErr, ok := recovered.(error); ok {
					result <- recoveredErr
					return
				}
				result <- fmt.Errorf("control operation failed: %v", recovered)
				return
			}
			if !completed {
				result <- errors.New("control operation failed: function exited without returning; did it call runtime.Goexit or FailNow?")
			}
		}()
		fn()
		completed = true
		result <- nil
	}()
	return <-result
}

func runControlStart(ctx context.Context, running func() bool, start func()) error {
	if running() {
		return fmt.Errorf("service is already running")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return controlCall(start)
}
