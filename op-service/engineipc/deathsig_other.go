//go:build !linux

package engineipc

import "os/exec"

// setDeathSignal is a no-op on non-Linux platforms (Pdeathsig is Linux-specific). Callers still
// Close() the engine explicitly; this only guards against a hard-killed parent.
func setDeathSignal(_ *exec.Cmd) {}
