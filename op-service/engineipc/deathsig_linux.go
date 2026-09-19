//go:build linux

package engineipc

import (
	"os/exec"
	"syscall"
)

// setDeathSignal makes the OS send SIGKILL to the spawned engine when the parent (this Go process)
// dies, so a hard-killed test can't leak long-lived engine subprocesses. Linux-only.
func setDeathSignal(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL
}
