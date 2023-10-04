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

// LifecycleCmd turns a LifecycleAction into an CLI action,
// by instrumenting it with CLI context and signal based termination.
// The app may continue to run post-processing until fully shutting down.
// The user can force an early shut-down during post-processing by sending a second interruption signal.
func LifecycleCmd(fn LifecycleAction) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		hostCtx := ctx.Context
		appCtx, appCancel := context.WithCancelCause(hostCtx)
		ctx.Context = appCtx

		go func() {
			opio.BlockOnInterruptsContext(appCtx)
			appCancel(errors.New("interrupt signal"))
		}()

		appLifecycle, err := fn(ctx, appCancel)
		if err != nil {
			return fmt.Errorf("failed to setup: %w", err)
		}

		if err := appLifecycle.Start(appCtx); err != nil {
			return fmt.Errorf("failed to start: %w", err)
		}

		// wait for app to be closed (through interrupt, or app requests to be stopped by closing the context)
		<-appCtx.Done()

		// Graceful stop context.
		// This allows the service to idle before shutdown, if halted. User may interrupt.
		stopCtx, stopCancel := context.WithCancel(hostCtx)
		go func() {
			opio.BlockOnInterruptsContext(stopCtx)
			stopCancel()
		}()

		// Execute graceful stop.
		stopErr := appLifecycle.Stop(stopCtx)
		stopCancel()
		if stopErr != nil {
			return fmt.Errorf("failed to stop app: %w", stopErr)
		}

		return nil
	}
}
