//go:build !unix

package vm

import "os/exec"

// setProcessGroup is a no-op on non-Unix platforms. Process-group teardown relies on
// Unix-only syscalls; op-challenger runs on Linux in production. cmd.WaitDelay still
// bounds the wait as a backstop.
func setProcessGroup(cmd *exec.Cmd) {}
