//go:build linux

package ipc

import (
	"os/exec"
	"syscall"
)

// setDeathSignal makes the OS send SIGKILL to the child when the parent dies, so a hard-killed
// parent can't leak long-lived children. Linux-only.
//
// Pdeathsig is tied to the OS thread that started the child, not to the process: the child is
// killed when that thread exits. The Go runtime keeps its threads alive, except one whose goroutine
// exits while locked to it (runtime.LockOSThread) — so Spawn must not run on such a goroutine.
func setDeathSignal(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL
}
