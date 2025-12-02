package cliapp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
)

var mockInterruptErr = errors.New("mock interrupt")

type fakeLifecycle struct {
	startCh, stopCh chan error
	stopped         bool
	selfClose       context.CancelCauseFunc
}

func (f *fakeLifecycle) Start(ctx context.Context) error {
	select {
	case err := <-f.startCh:
		f.stopped = true
		return err
	case <-ctx.Done():
		f.stopped = true
		return ctx.Err()
	}
}

func (f *fakeLifecycle) Stop(ctx context.Context) error {
	select {
	case err := <-f.stopCh:
		f.stopped = true
		return err
	case <-ctx.Done():
		f.stopped = true
		return ctx.Err()
	}
}

func (f *fakeLifecycle) Stopped() bool {
	return f.stopped
}

var _ Lifecycle = (*fakeLifecycle)(nil)

func TestLifecycleCmd(t *testing.T) {

	appSetup := func(t *testing.T) (signalCh chan struct{}, initCh, startCh, stopCh, resultCh chan error, appCh chan *fakeLifecycle) {
		signalCh = make(chan struct{})
		initCh = make(chan error)
		startCh = make(chan error)
		stopCh = make(chan error)
		resultCh = make(chan error)
		// optional channel to retrieve the fakeLifecycle from, available some time after init, before start.
		appCh = make(chan *fakeLifecycle, 1)

		// mock an application that may fail at different stages of its lifecycle
		mockAppFn := func(ctx *cli.Context, close context.CancelCauseFunc) (Lifecycle, error) {
			select {
			case <-ctx.Context.Done():
				return nil, ctx.Context.Err()
			case err := <-initCh:
				if err != nil {
					return nil, err
				}
			}

			app := &fakeLifecycle{
				startCh:   startCh,
				stopCh:    stopCh,
				stopped:   false,
				selfClose: close,
			}
			appCh <- app
			return app, nil
		}

		// turn our mock app and system signal into a lifecycle-managed command
		actionFn := LifecycleCmd(mockAppFn)

		// try to shut the test down after being locked more than a minute
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)

		// puppeteer system signal interrupts by hooking up the test signal channel as "blocker" for the app to use.
		ctx = ctxinterrupt.WithWaiterFunc(ctx, func(ctx context.Context) (interrupt, ctxErr error) {
			select {
			case <-ctx.Done():
				ctxErr = context.Cause(ctx)
			case <-signalCh:
				interrupt = mockInterruptErr
			}
			return
		})
		t.Cleanup(cancel)

		// create a fake CLI context to run our command with
		cliCtx := &cli.Context{
			Context: ctx,
			App: &cli.App{
				Name:   "test-app",
				Action: actionFn,
			},
			Command: nil,
		}
		// run the command async, it may block etc. The result will be sent back to the tester.
		go func() {
			result := actionFn(cliCtx)
			require.NoError(t, ctx.Err(), "expecting test context to be alive after end still")
			// collect the result
			resultCh <- result
		}()
		t.Cleanup(func() {
			close(signalCh)
			close(initCh)
			close(startCh)
			close(stopCh)
			close(resultCh)
			close(appCh)
		})
		return
	}

	t.Run("interrupt int", func(t *testing.T) {
		signalCh, _, _, _, resultCh, _ := appSetup(t)
		signalCh <- struct{}{}
		res := <-resultCh
		require.ErrorIs(t, res, mockInterruptErr)
		require.ErrorContains(t, res, "failed to setup")
	})
	t.Run("failed init", func(t *testing.T) {
		_, initCh, _, _, resultCh, _ := appSetup(t)
		v := errors.New("TEST INIT ERROR")
		initCh <- v
		res := <-resultCh
		require.ErrorIs(t, res, v)
		require.ErrorContains(t, res, "failed to setup")
	})
	t.Run("interrupt start", func(t *testing.T) {
		signalCh, initCh, _, _, resultCh, appCh := appSetup(t)
		initCh <- nil
		app := <-appCh
		require.False(t, app.Stopped())
		signalCh <- struct{}{}
		res := <-resultCh
		require.ErrorIs(t, res, mockInterruptErr)
		require.ErrorContains(t, res, "failed to start")
		require.True(t, app.Stopped())
	})
	t.Run("failed start", func(t *testing.T) {
		_, initCh, startCh, _, resultCh, appCh := appSetup(t)
		initCh <- nil
		app := <-appCh
		require.False(t, app.Stopped())
		v := errors.New("TEST START ERROR")
		startCh <- v
		res := <-resultCh
		require.ErrorIs(t, res, v)
		require.ErrorContains(t, res, "failed to start")
		require.True(t, app.Stopped())
	})
	t.Run("graceful shutdown", func(t *testing.T) {
		signalCh, initCh, startCh, stopCh, resultCh, appCh := appSetup(t)
		initCh <- nil
		app := <-appCh
		require.False(t, app.Stopped())
		startCh <- nil
		signalCh <- struct{}{} // interrupt, but at an expected time
		stopCh <- nil          // graceful shutdown after interrupt
		require.NoError(t, <-resultCh, nil)
		require.True(t, app.Stopped())
	})
	t.Run("interrupted shutdown", func(t *testing.T) {
		signalCh, initCh, startCh, _, resultCh, appCh := appSetup(t)
		initCh <- nil
		app := <-appCh
		require.False(t, app.Stopped())
		startCh <- nil
		signalCh <- struct{}{} // start graceful shutdown
		signalCh <- struct{}{} // interrupt before the shutdown process is allowed to complete
		res := <-resultCh
		require.ErrorIs(t, res, mockInterruptErr)
		require.ErrorContains(t, res, "failed to stop")
		require.True(t, app.Stopped()) // still fully closes, interrupts only accelerate shutdown where possible.
	})
	t.Run("failed shutdown", func(t *testing.T) {
		signalCh, initCh, startCh, stopCh, resultCh, appCh := appSetup(t)
		initCh <- nil
		app := <-appCh
		require.False(t, app.Stopped())
		startCh <- nil
		signalCh <- struct{}{} // start graceful shutdown
		v := errors.New("TEST STOP ERROR")
		stopCh <- v
		res := <-resultCh
		require.ErrorIs(t, res, v)
		require.ErrorContains(t, res, "failed to stop")
		require.True(t, app.Stopped())
	})
	t.Run("app self-close", func(t *testing.T) {
		_, initCh, startCh, stopCh, resultCh, appCh := appSetup(t)
		initCh <- nil
		app := <-appCh
		require.False(t, app.Stopped())
		startCh <- nil
		v := errors.New("TEST SELF CLOSE ERROR")
		app.selfClose(v)
		stopCh <- nil
		require.NoError(t, <-resultCh, "self-close is not considered an error")
		require.True(t, app.Stopped())
	})
}
