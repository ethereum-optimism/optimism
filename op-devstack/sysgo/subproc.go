package sysgo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
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

	// exited is closed once cmd.Wait() returns, i.e. the process has exited and its
	// stdout/stderr have been fully flushed. waitErr holds that Wait() result and is
	// safe to read only after exited is closed.
	exited  chan struct{}
	waitErr error

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
	sp.stdOutProc = stdOutProc
	sp.stdErrProc = stdErrProc
	sp.exited = make(chan struct{})
	// Own the single cmd.Wait() here so callers can observe an early exit via Exited()
	// without racing a second Wait() in Stop(). cmd.Wait() also blocks until stdout/stderr
	// have been fully copied, so all log output is flushed by the time exited is closed.
	go func() {
		sp.waitErr = cmd.Wait()
		close(sp.exited)
	}()
	sp.p.Cleanup(func() {
		err := sp.Stop(true)
		if err != nil {
			sp.p.Logger().Error("Shutdown error", "err", err)
		}
	})
	return nil
}

// Exited returns a channel that is closed once the process has exited and its
// stdout/stderr have been fully flushed. It must only be called after Start.
func (sp *SubProcess) Exited() <-chan struct{} {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	return sp.exited
}

// ExitErr returns what waiting on the process reported, nil for a clean exit. Exited must be
// closed before it is called; that close is what publishes the value.
func (sp *SubProcess) ExitErr() error {
	return sp.waitErr
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

	// If the process is still running, request an interrupt as requested. We avoid
	// reading sp.cmd.ProcessState here since the Wait() goroutine writes it; instead
	// check the exited channel without blocking.
	select {
	case <-sp.exited:
		// already exited; nothing to interrupt
	default:
		if interrupt {
			sp.p.Logger().Info("Sending interrupt")
			if err := sp.cmd.Process.Signal(os.Interrupt); err != nil {
				// The process may have exited between the check and the signal; log
				// rather than fail, then fall through to wait for the exit result.
				sp.p.Logger().Warn("Failed to interrupt sub-process", "err", err)
			}
		}
	}

	// Wait for the Wait() goroutine to report the exit. cmd.Wait() (run there) blocks
	// until all stdout/stderr data is flushed, so log output is complete before we return.
	ok := waitProcess(ctx, sp.exited, interruptWait)
	if !ok {
		if err := cmd.Process.Kill(); err != nil {
			ok = waitProcess(ctx, sp.exited, killWait)
			if !ok {
				return fmt.Errorf("interrupt timed out and kill failed: %w", err)
			}
			return sp.completeStopLocked(interrupt, sp.waitErr)
		}
		ok = waitProcess(ctx, sp.exited, killWait)
		if !ok {
			return fmt.Errorf("process did not stop after interrupt and kill")
		}
	}

	return sp.completeStopLocked(interrupt, sp.waitErr)
}

// Kill ends the process with SIGKILL, so none of its shutdown logic runs: a crash, not a power
// loss. A process that had already exited on its own is released, and Kill returns that exit as
// an error.
func (sp *SubProcess) Kill() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if sp.cmd == nil {
		return nil // already stopped
	}

	select {
	case <-sp.exited:
		return sp.releaseExitedBeforeKillLocked()
	default:
	}

	sp.p.Logger().Info("Killing sub-process", "pid", sp.cmd.Process.Pid)
	killErr := sp.cmd.Process.Kill()
	if killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
		sp.p.Logger().Error("Failed to kill sub-process", "pid", sp.cmd.Process.Pid, "err", killErr)
		return fmt.Errorf("failed to kill sub-process: %w", killErr)
	}
	if !waitProcess(context.Background(), sp.exited, controlledKillWait) {
		sp.p.Logger().Error("Sub-process did not exit after SIGKILL", "pid", sp.cmd.Process.Pid, "wait", controlledKillWait)
		return fmt.Errorf("sub-process did not exit within %s of SIGKILL", controlledKillWait)
	}
	// The process can also exit on its own after the check above: Wait either reaps it before the
	// signal (ErrProcessDone) or reaps it after the signal landed on the zombie, changing nothing.
	if errors.Is(killErr, os.ErrProcessDone) || !terminatedBySIGKILL(sp.waitErr) {
		return sp.releaseExitedBeforeKillLocked()
	}
	sp.p.Logger().Info("Sub-process killed", "exit", sp.waitErr)
	sp.releaseLocked()
	return nil
}

// releaseExitedBeforeKillLocked releases a process that exited on its own and returns the error
// naming that exit.
func (sp *SubProcess) releaseExitedBeforeKillLocked() error {
	sp.p.Logger().Error("Sub-process exited before it could be killed", "exit", sp.waitErr)
	sp.releaseLocked()
	if sp.waitErr == nil {
		return errors.New("sub-process exited with status 0 before it could be killed")
	}
	return fmt.Errorf("sub-process exited before it could be killed: %w", sp.waitErr)
}

func terminatedBySIGKILL(waitErr error) bool {
	var exitErr *exec.ExitError
	if !errors.As(waitErr, &exitErr) {
		return false
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	return ok && status.Signaled() && status.Signal() == syscall.SIGKILL
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
	sp.releaseLocked()
	return nil
}

// releaseLocked closes the log pipes and forgets the exited process so Start can run a new one.
func (sp *SubProcess) releaseLocked() {
	if sp.stdOutProc != nil {
		_ = sp.stdOutProc.Close()
		sp.stdOutProc = nil
	}
	if sp.stdErrProc != nil {
		_ = sp.stdErrProc.Close()
		sp.stdErrProc = nil
	}
	sp.cmd = nil
}

func waitProcess(ctx context.Context, exited <-chan struct{}, timeout time.Duration) bool {
	if timeout <= 0 {
		select {
		case <-exited:
			return true
		case <-ctx.Done():
			return false
		}
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-exited:
		return true
	case <-timer.C:
		return false
	case <-ctx.Done():
		return false
	}
}
