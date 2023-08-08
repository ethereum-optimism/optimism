package utils

import (
	"os"
	"os/exec"
	"syscall"
)

// ProcessGroup contains an [exec.Cmd] and a channel.
type ProcessGroup struct {
	sigChan chan os.Signal
	done    chan bool
	*exec.Cmd
}

// NewProcessGroup creates a new ProcessGroup to monitor a signal channel.
func NewProcessGroup(c chan os.Signal, cmd *exec.Cmd) *ProcessGroup {
	return &ProcessGroup{
		sigChan: c,
		Cmd:     cmd,
		done:    make(chan bool, 1),
	}
}

// Run performs the equivalent of running the underlying Cmd.
//
// When spawning the command, it will be put into its own OS process proup.
//
// If a value is received on the signal channel, the command _and_ all child
// processes will be killed (SIGKILL) by signalling the command's process
// group id.
func (p *ProcessGroup) Run() error {
	p.Cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	p.Cmd.Stdout = os.Stdout
    p.Cmd.Stderr = os.Stderr
	if err := p.Cmd.Start(); err != nil {
		return err
	}
	defer func() {
		close(p.done)
	}()
	go func() {
		select {
		case <-p.sigChan:
			syscall.Kill(-p.Cmd.Process.Pid, syscall.SIGKILL)
			return
		case <-p.done:
			return
		}
	}()
	return nil
}

// Terminate sends a signal on its signal channel, killing the command
// and all child processes with a [syscall.SIGKILL].
func (p *ProcessGroup) Terminate() {
	p.sigChan <- syscall.Signal(syscall.SIGTERM)
}
