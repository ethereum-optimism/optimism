package sysgo

import (
	"context"
	"fmt"
	"time"
)

const (
	controlledInterruptWait = 10 * time.Second
	controlledKillWait      = 5 * time.Second
)

func controlCall(fn func()) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if recoveredErr, ok := recovered.(error); ok {
				err = recoveredErr
				return
			}
			err = fmt.Errorf("control operation failed: %v", recovered)
		}
	}()
	fn()
	return nil
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
