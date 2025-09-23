package sysgo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
)

// SubProcess is a process that can be started, and stopped, and restarted.
//
// If at any point the process fails to start or exit successfully,
// the failure is reported to the devtest.P.
//
// If the sub-process exits by itself, the exit is detected,
// and if not successful (non-zero exit code on unix) it also reports failure to the devtest.P.
//
// Sub-process logs are assumed to be structured JSON logs, and are piped to the logger.
type SubProcess struct {
	p   devtest.P
	cmd *exec.Cmd

	stdOutLogs logpipe.LogProcessor
	stdErrLogs logpipe.LogProcessor

	waitCtx context.Context // closed when process-Wait completes

	mu sync.Mutex
}

func NewSubProcess(p devtest.P, stdOutLogs, stdErrLogs logpipe.LogProcessor) *SubProcess {
	return &SubProcess{
		p:          p,
		stdOutLogs: stdOutLogs,
		stdErrLogs: stdErrLogs,
	}
}

func (sp *SubProcess) Start(cmdPath string, args []string, env []string) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if sp.cmd != nil {
		return fmt.Errorf("process is still running (PID: %d)", sp.cmd.Process.Pid)
	}
	cmd := exec.Command(cmdPath, args...)
	cmd.Env = append(os.Environ(), env...)
	stdout, err := cmd.StdoutPipe()
	sp.p.Require().NoError(err, "stdout err")
	stderr, err := cmd.StderrPipe()
	sp.p.Require().NoError(err, "stderr err")
	go func() {
		err := logpipe.PipeLogs(stdout, sp.stdOutLogs)
		sp.p.Require().NoError(err, "stdout logging error")
	}()
	go func() {
		err := logpipe.PipeLogs(stderr, sp.stdErrLogs)
		sp.p.Require().NoError(err, "stderr logging error")
	}()
	if err := cmd.Start(); err != nil {
		return err
	}
	sp.cmd = cmd

	subCtx, subCancel := context.WithCancelCause(context.Background())
	go func() {
		state, err := cmd.Process.Wait()
		subCancel(err)
		sp.p.Require().NoError(err, "Sub-process failed to be closed")
		sp.p.Logger().Info("Sub-process stopped", "exitCode", state.ExitCode(), "pid", state.Pid())
		// if it exited on its own, then we care about the error. If not, we (or the user) signaled it.
		if state.Exited() {
			sp.p.Require().True(state.Success(), "Sub-process closed with error status: %s", state.String())
		}
	}()
	sp.waitCtx = subCtx

	sp.p.Cleanup(func() {
		err := sp.Stop()
		if err != nil {
			sp.p.Logger().Error("Shutdown error", "err", err)
		}
	})
	return nil
}

// Kill stops the process, and does not wait for it to complete.
func (sp *SubProcess) Kill() error {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // don't wait, just force it to stop immediately
	return sp.GracefulStop(ctx)
}

// Stop implements the default control-panel interface,
// and gracefully stops with a 10-second timeout.
func (sp *SubProcess) Stop() error {
	// by default, for control-panel, use an interrupt and a 10-second grace
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	return sp.GracefulStop(ctx)
}

// GracefulStop sends an interrupt and waits for the process to stop.
// If the given ctx is closed, a forced shutdown (process kill) is pursued.
func (sp *SubProcess) GracefulStop(ctx context.Context) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if sp.cmd == nil {
		return nil // already stopped gracefully
	}

	if ctx.Err() == nil && sp.waitCtx.Err() == nil {
		// if not force-closing, and not already done, then try an interrupt first.
		sp.p.Logger().Info("Sending interrupt")
		if err := sp.cmd.Process.Signal(os.Interrupt); err != nil {
			return err
		}
	}
	select {
	case <-ctx.Done():
		sp.p.Logger().Warn("Sub-process did not respond to interrupt, force-closing now")
		err := sp.cmd.Process.Kill()
		if err != nil {
			return fmt.Errorf("failed to force-close sub-process: %w", err)
		}
		sp.p.Logger().Info("Successfully force-closed sub-process")
		// resources of cmd.Process will be cleaned up by the Process.Wait
	case <-sp.waitCtx.Done():
		if err := context.Cause(sp.waitCtx); err != nil && err != context.Canceled {
			sp.p.Logger().Warn("Sub-process exited with error", "err", err)
		} else {
			sp.p.Logger().Info("Sub-process gracefully exited")
		}
	}
	sp.cmd = nil
	sp.waitCtx = nil
	return nil
}

// Wait waits for the process to complete.
func (sp *SubProcess) Wait(ctx context.Context) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if sp.waitCtx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-sp.waitCtx.Done():
		if err := context.Cause(sp.waitCtx); err != nil && err != context.Canceled {
			sp.p.Logger().Warn("Sub-process exited with error", "err", err)
			return err
		} else {
			sp.p.Logger().Info("Sub-process gracefully exited")
			return nil
		}
	}
}
