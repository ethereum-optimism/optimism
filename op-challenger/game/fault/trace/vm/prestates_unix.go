//go:build unix

package vm

import (
	"os/exec"
	"syscall"
)

// setProcessGroup runs cmd in its own process group and, on context cancellation,
// kills the entire group (negative pid) rather than just the leader. This ensures
// the VM's orphaned oracle-server grandchild is terminated along with the VM.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
