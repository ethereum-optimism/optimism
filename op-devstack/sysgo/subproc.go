package sysgo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
)

// SubProcess is a process that can be started and stopped.
type SubProcess struct {
	p   devtest.CommonT
	cmd *exec.Cmd

	stdOutCallback logpipe.LogCallback
	stdErrCallback logpipe.LogCallback

	stdOutProc *logpipe.LineBuffer
	stdErrProc *logpipe.LineBuffer
	waitCh     chan error

	mu sync.Mutex
}

func NewSubProcess(p devtest.CommonT, stdOutCallback, stdErrCallback logpipe.LogCallback) *SubProcess {
	return &SubProcess{
		p:              p,
		stdOutCallback: stdOutCallback,
		stdErrCallback: stdErrCallback,
	}
}

func (sp *SubProcess) Start(cmdPath string, args []string, env []string) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if sp.cmd != nil {
		return fmt.Errorf("process is still running (PID: %d)", sp.cmd.Process.Pid)
	}
	sp.p.Logger().Info("Starting subprocess", "cmd", cmdPath, "args", args)

	stdOutProc := logpipe.NewLineBuffer(sp.stdOutCallback)
	stdErrProc := logpipe.NewLineBuffer(sp.stdErrCallback)

	cmd := exec.Command(cmdPath, args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = stdOutProc
	cmd.Stderr = stdErrProc
	if err := cmd.Start(); err != nil {
		return err
	}
	sp.cmd = cmd
	sp.waitCh = nil
	sp.stdOutProc = stdOutProc
	sp.stdErrProc = stdErrProc
	sp.p.Cleanup(func() {
		err := sp.Stop(true)
		if err != nil {
			sp.p.Logger().Error("Shutdown error", "err", err)
		}
	})
	return nil
}

// Stop waits for the process to stop, interrupting the process if it has not completed and
// interrupt is true.
func (sp *SubProcess) Stop(interrupt bool) error {
	return sp.stop(context.Background(), interrupt, 0, 0)
}

func (sp *SubProcess) StopControlled(ctx context.Context, interruptWait time.Duration, killWait time.Duration) error {
	return sp.stop(ctx, true, interruptWait, killWait)
}

func (sp *SubProcess) stop(ctx context.Context, interrupt bool, interruptWait time.Duration, killWait time.Duration) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if sp.cmd == nil {
		return nil // already stopped gracefully
	}
	cmd := sp.cmd
	waitCh := sp.waitCh
	if waitCh == nil {
		waitCh = make(chan error, 1)
		sp.waitCh = waitCh
		go func() {
			waitCh <- cmd.Wait()
		}()
	}
	select {
	case waitErr := <-waitCh:
		return sp.completeStopLocked(interrupt, waitErr)
	default:
	}

	// If not already done, then try an interrupt first as requested.
	if cmd.ProcessState == nil && interrupt {
		sp.p.Logger().Info("Sending interrupt")
		if err := cmd.Process.Signal(os.Interrupt); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return err
		}
	}

	// Use cmd.Wait() instead of cmd.Process.Wait() to ensure all stdout/stderr
	// data is fully flushed before returning. Process.Wait() only waits for the
	// process to exit but does not guarantee I/O completion, which causes races
	// where log output hasn't been written to the LineBuffer yet.
	waitErr, ok := waitProcess(ctx, waitCh, interruptWait)
	if !ok {
		if err := cmd.Process.Kill(); err != nil {
			waitErr, ok = waitProcess(ctx, waitCh, killWait)
			if !ok {
				return fmt.Errorf("interrupt timed out and kill failed: %w", err)
			}
			return sp.completeStopLocked(interrupt, waitErr)
		}
		waitErr, ok = waitProcess(ctx, waitCh, killWait)
		if !ok {
			return fmt.Errorf("process did not stop after interrupt and kill")
		}
	}

	return sp.completeStopLocked(interrupt, waitErr)
}

func (sp *SubProcess) completeStopLocked(interrupt bool, waitErr error) error {
	var exitErr *exec.ExitError
	if waitErr != nil && !(interrupt && errors.As(waitErr, &exitErr)) {
		sp.p.Logger().Warn("Sub-process exited with error", "err", waitErr)
	} else if interrupt && waitErr != nil {
		sp.p.Logger().Info("Sub-process stopped")
	} else {
		sp.p.Logger().Info("Sub-process gracefully exited")
	}

	if sp.stdOutProc != nil {
		_ = sp.stdOutProc.Close()
		sp.stdOutProc = nil
	}
	if sp.stdErrProc != nil {
		_ = sp.stdErrProc.Close()
		sp.stdErrProc = nil
	}
	sp.cmd = nil
	sp.waitCh = nil
	return nil
}

func waitProcess(ctx context.Context, waitCh <-chan error, timeout time.Duration) (error, bool) {
	if timeout <= 0 {
		select {
		case err := <-waitCh:
			return err, true
		case <-ctx.Done():
			return ctx.Err(), false
		}
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case err := <-waitCh:
		return err, true
	case <-timer.C:
		return nil, false
	case <-ctx.Done():
		return ctx.Err(), false
	}
}
