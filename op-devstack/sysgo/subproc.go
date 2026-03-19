package sysgo

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"

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
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if sp.cmd == nil {
		return nil // already stopped gracefully
	}

	// If not already done, then try an interrupt first as requested.
	if sp.cmd.ProcessState == nil && interrupt {
		sp.p.Logger().Info("Sending interrupt")
		if err := sp.cmd.Process.Signal(os.Interrupt); err != nil {
			return err
		}
	}

	// Use cmd.Wait() instead of cmd.Process.Wait() to ensure all stdout/stderr
	// data is fully flushed before returning. Process.Wait() only waits for the
	// process to exit but does not guarantee I/O completion, which causes races
	// where log output hasn't been written to the LineBuffer yet.
	waitErr := sp.cmd.Wait()
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
	return nil
}
