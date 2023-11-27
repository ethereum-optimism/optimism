package stack

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	oppio "github.com/ethereum-optimism/optimism/op-program/io"
)

func ExecSource(name string, stdOut, stdErr io.Writer, onComplete context.CancelCauseFunc) Source {
	return func() (preimageRW oppio.FileChannel, hintRW oppio.FileChannel, stop Stoppable, err error) {
		pClientRW, pHostRW, hClientRW, hHostRW, stopPipe, err := MiddlewarePipes()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create pipes: %w", err)
		}
		host := ExecSink(name, stdOut, stdErr, onComplete)
		stopExec, err := host(pClientRW, hClientRW)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to exec source: %w", err)
		}
		stop = StopFn(func(ctx context.Context) error {
			return errors.Join(stopExec.Stop(ctx), stopPipe.Stop(ctx))
		})
		return pHostRW, hHostRW, stop, nil
	}
}

func ExecSink(name string, stdOut, stdErr io.Writer, onComplete context.CancelCauseFunc) Sink {
	return func(preimageRW oppio.FileChannel, hintRW oppio.FileChannel) (Stoppable, error) {
		cmd := exec.Command(name)
		cmd.ExtraFiles = make([]*os.File, preimage.MaxFd-3) // not including stdin, stdout and stderr
		cmd.ExtraFiles[preimage.HClientRFd-3] = hintRW.Reader()
		cmd.ExtraFiles[preimage.HClientWFd-3] = hintRW.Writer()
		cmd.ExtraFiles[preimage.PClientRFd-3] = preimageRW.Reader()
		cmd.ExtraFiles[preimage.PClientWFd-3] = preimageRW.Writer()
		cmd.Stdout = stdOut // for debugging
		cmd.Stderr = stdErr // for debugging

		cmd.WaitDelay = time.Second

		err := cmd.Start()
		if err != nil {
			return nil, fmt.Errorf("program cmd failed to start: %w", err)
		}

		// now wait for it to stop, and kill if it doesn't stop before ctx cancellation.
		waitErr := make(chan error, 1)
		go func() {
			err := cmd.Wait()
			waitErr <- err
			onComplete(err) // signal when the sub-process has stopped (it may decide to stop by its own)
		}()

		return StopFn(func(ctx context.Context) error {
			// see if we stopped already
			if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
				// No stop-error if it already exited.
				// The final process-state is communicated through onComplete.
				return nil
			}
			// If we already cancelled the context, don't even try to interrupt, just kill the process.
			if ctx.Err() != nil {
				if err := cmd.Process.Kill(); err != nil {
					err = errors.Join(err, ctx.Err(), cmd.Process.Release())
					return fmt.Errorf("ctx cancelled and failed to kill process: %w", err)
				}
				return errors.Join(ctx.Err(), cmd.Wait())
			}
			// First signal nicely that we like to stop it
			_ = cmd.Process.Signal(os.Interrupt) // ignore error, process may be unable to handle interrupt.

			// Now gracefully wait for the process to stop, or kill the process if the user cancels waiting.
			select {
			case <-ctx.Done():
				killErr := cmd.Process.Kill()
				// wait should complete after above kill, and release the resources etc.
				waitErr := <-waitErr
				return errors.Join(waitErr, killErr)
			case err := <-waitErr:
				// Wait releases the cmd resources already
				return err
			}
		}), nil
	}
}
