package cliapp

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-service/opio"
)

type Lifecycle interface {
	// Start starts a service. A service only fully starts once. Subsequent starts may return an error.
	// A context is provided to end the service during setup.
	// The caller should call Stop to clean up after failing to start.
	Start(ctx context.Context) error
	// Stop stops a service gracefully.
	// The provided ctx can force an accelerated shutdown,
	// but the node still has to completely stop.
	Stop(ctx context.Context) error
	// Stopped determines if the service was stopped with Stop.
	Stopped() bool
}

// LifecycleAction instantiates a Lifecycle based on a CLI context.
// With the close argument a lifecycle may choose to shut itself down.
// A service may choose to idle, dump debug information or otherwise delay
// a shutdown when the Stop context is not expired.
type LifecycleAction func(ctx *cli.Context, close context.CancelCauseFunc) (Lifecycle, error)

var interruptErr = errors.New("interrupt signal")

// LifecycleCmd turns a LifecycleAction into an CLI action,
// by instrumenting it with CLI context and signal based termination.
// The signals are caught with the opio.BlockFn attached to the context, if any.
// If no block function is provided, it adds default interrupt handling.
// The app may continue to run post-processing until fully shutting down.
// The user can force an early shut-down during post-processing by sending a second interruption signal.
func LifecycleCmd(fn LifecycleAction) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		hostCtx := ctx.Context
		blockOnInterrupt := opio.BlockerFromContext(hostCtx)
		if blockOnInterrupt == nil { // add default interrupt blocker to context if none is set.
			hostCtx = opio.WithInterruptBlocker(hostCtx)
			blockOnInterrupt = opio.BlockerFromContext(hostCtx)
		}
		appCtx, appCancel := context.WithCancelCause(hostCtx)
		ctx.Context = appCtx

		go func() {
			blockOnInterrupt(appCtx)
			appCancel(interruptErr)
		}()

		appLifecycle, err := fn(ctx, appCancel)
		if err != nil {
			// join errors to include context cause (nil errors are dropped)
			return errors.Join(
				fmt.Errorf("failed to setup: %w", err),
				context.Cause(appCtx),
			)
		}

		if err := appLifecycle.Start(appCtx); err != nil {
			// join errors to include context cause (nil errors are dropped)
			return errors.Join(
				fmt.Errorf("failed to start: %w", err),
				context.Cause(appCtx),
			)
		}

		// wait for app to be closed (through interrupt, or app requests to be stopped by closing the context)
		<-appCtx.Done()

		// Graceful stop context.
		// This allows the service to idle before shutdown, if halted. User may interrupt.
		stopCtx, stopCancel := context.WithCancelCause(hostCtx)
		go func() {
			blockOnInterrupt(stopCtx)
			stopCancel(interruptErr)
		}()

		// Execute graceful stop.
		stopErr := appLifecycle.Stop(stopCtx)
		stopCancel(nil)
		// note: Stop implementation may choose to suppress a context error,
		// if it handles it well (e.g. stop idling after a halt).
		if stopErr != nil {
			// join errors to include context cause (nil errors are dropped)
			return errors.Join(
				fmt.Errorf("failed to stop: %w", stopErr),
				context.Cause(stopCtx),
			)
		}

		return nil
	}
}
