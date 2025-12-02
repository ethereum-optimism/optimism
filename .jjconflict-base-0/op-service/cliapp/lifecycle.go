package cliapp

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
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

// LifecycleCmd turns a LifecycleAction into an CLI action,
// by instrumenting it with CLI context and signal based cancellation.
// The signals are caught with the ctxinterrupt.waiter attached to the context, or default
// interrupt signal handling if not already provided.
// The app may continue to run post-processing until fully shutting down.
// The user can force an early shut-down during post-processing by sending a second interruption signal.
func LifecycleCmd(fn LifecycleAction) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		hostCtx, stop := ctxinterrupt.WithSignalWaiter(ctx.Context)
		defer stop()
		appCtx, appCancel := context.WithCancelCause(ctxinterrupt.WithCancelOnInterrupt(hostCtx))
		// This is updated so the fn callback cli.Context uses the appCtx we just made.
		ctx.Context = appCtx

		appLifecycle, err := fn(ctx, appCancel)
		if err != nil {
			// join errors to include context cause (nil errors are dropped)
			return errors.Join(
				fmt.Errorf("failed to setup: %w", err),
				context.Cause(ctx.Context),
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
		stopCtx := ctxinterrupt.WithCancelOnInterrupt(hostCtx)

		// Execute graceful stop.
		stopErr := appLifecycle.Stop(stopCtx)
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
