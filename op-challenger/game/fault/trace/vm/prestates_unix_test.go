//go:build unix

package vm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestRunCmdKillsGrandchildOnCancel reproduces the orphaned-grandchild deadlock: the VM
// spawns an oracle-server child that inherits the VM's stdout/stderr pipes and sleeps
// forever. Without process-group teardown, killing only the direct child leaves the
// grandchild holding the pipes open and cmd.Wait blocks indefinitely. RunCmd must return
// promptly when the context is cancelled and the grandchild must be killed.
func TestRunCmdKillsGrandchildOnCancel(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	pidFile := filepath.Join(t.TempDir(), "grandchild.pid")

	// The parent (the "VM") forks a grandchild (the "oracle server") that records its pid,
	// keeps the inherited stdout open, and sleeps far longer than the test. The parent then
	// sleeps too. Neither writes EOF, mirroring the production deadlock.
	script := fmt.Sprintf(`( echo $$ > %q; sleep 600 ) & sleep 600`, pidFile)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- RunCmd(ctx, logger, "sh", "-c", script)
	}()

	// Wait until the grandchild has recorded its pid before cancelling. This guarantees the
	// full process tree is up, so the test exercises the teardown deadlock rather than racing
	// cancellation against process startup (which could spuriously pass on an overloaded box).
	grandchildPid := readPid(t, pidFile)
	start := time.Now()
	cancel()

	select {
	case <-done:
		require.Less(t, time.Since(start), 5*time.Second, "RunCmd should return promptly after cancellation")
	case <-time.After(20 * time.Second):
		t.Fatal("RunCmd did not return after context cancellation - process tree teardown hung")
	}

	require.Eventually(t, func() bool {
		return !processAlive(grandchildPid)
	}, 5*time.Second, 50*time.Millisecond, "grandchild oracle-server process should be killed with the group")
}

func readPid(t *testing.T, pidFile string) int {
	var data []byte
	require.Eventually(t, func() bool {
		var err error
		data, err = os.ReadFile(pidFile)
		return err == nil && len(strings.TrimSpace(string(data))) > 0
	}, 5*time.Second, 50*time.Millisecond, "grandchild should record its pid")
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	require.NoError(t, err)
	return pid
}

func processAlive(pid int) bool {
	// Signal 0 performs error checking without sending a signal.
	return syscall.Kill(pid, 0) == nil
}
