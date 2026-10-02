package ipc_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/ipc"
)

// TestSpawnFailureDrainsStderr reads the forwarded stderr right after a failed Spawn whose child is
// still writing. Spawn (and Close, which it calls) must not return before the child's stderr is
// fully forwarded, or the read races the forwarder (run with -race).
func TestSpawnFailureDrainsStderr(t *testing.T) {
	defer ipc.SetReadyTimeout(time.Second)()
	sh, err := exec.LookPath("sh")
	require.NoError(t, err)

	var logs bytes.Buffer
	_, err = ipc.Spawn(sh, "--socket", []string{"-c", "while :; do echo spam >&2; done"}, &logs)
	require.ErrorContains(t, err, "never became ready")
	require.Contains(t, logs.String(), "[sh] spam\n")
}

// TestSpawnFailureWithStderrHeldByDescendant spawns a wrapper whose background child inherits the
// wrapper's stderr and outlives it. Spawn's cleanup must not wait for that child to close the pipe
// once the wrapper is killed.
func TestSpawnFailureWithStderrHeldByDescendant(t *testing.T) {
	defer ipc.SetReadyTimeout(time.Second)()
	sh, err := exec.LookPath("sh")
	require.NoError(t, err)
	pidFile := filepath.Join(t.TempDir(), "descendant.pid")
	t.Cleanup(func() {
		raw, err := os.ReadFile(pidFile)
		if err != nil {
			return
		}
		if pid, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil {
			if proc, err := os.FindProcess(pid); err == nil {
				_ = proc.Kill()
			}
		}
	})

	script := "sleep 60 & echo $! > " + pidFile + "; while :; do sleep 1; done"
	done := make(chan error, 1)
	go func() {
		_, err := ipc.Spawn(sh, "--socket", []string{"-c", script}, nil)
		done <- err
	}()
	select {
	case err := <-done:
		require.ErrorContains(t, err, "never became ready")
	case <-time.After(15 * time.Second):
		t.Fatal("Spawn did not return while a descendant of the child held its stderr open")
	}
}

// TestSpawnReportsEarlyExit spawns a child that exits at once. Spawn must fail as soon as the child
// exits, not after the ready timeout, and report its exit status and stderr.
func TestSpawnReportsEarlyExit(t *testing.T) {
	sh, err := exec.LookPath("sh")
	require.NoError(t, err)

	start := time.Now()
	_, err = ipc.Spawn(sh, "--socket", []string{"-c", "echo 'cannot load config' >&2; exit 3"}, nil)
	require.Less(t, time.Since(start), 10*time.Second, "spawn failure must not wait out the ready timeout")
	require.ErrorContains(t, err, "exit status 3")
	require.ErrorContains(t, err, "cannot load config")
}

// TestSpawnPassesSocketArg checks that the child receives the socket path under socketArg, after
// the caller's arguments.
func TestSpawnPassesSocketArg(t *testing.T) {
	sh, err := exec.LookPath("sh")
	require.NoError(t, err)

	// sh -c binds the arguments after the script to $0, $1, ...
	_, err = ipc.Spawn(sh, "--listen-on", []string{"-c", `echo "got $0 $1" >&2; exit 1`}, nil)
	require.ErrorContains(t, err, "got --listen-on /")
	require.ErrorContains(t, err, "rpc.sock")
}
