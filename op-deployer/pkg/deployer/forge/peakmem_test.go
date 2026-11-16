package forge

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// helperMemory is the memory that the touch process of TestExecWithEnvReturnsPeakMemory touches.
const helperMemory = 64 << 20

// peakMemoryRole names the role of a process of TestExecWithEnvReturnsPeakMemory.
const peakMemoryRole = "FORGE_TEST_PEAK_MEMORY_ROLE"

// TestExecWithEnvReturnsPeakMemory checks the peak memory of a child that touches helperMemory: in
// the right unit, KiB on Linux and bytes on macOS, and not the higher peak of its parent, which
// Linux's exec carries into the child's. Each measure runs in a new process, whose peak is known.
func TestExecWithEnvReturnsPeakMemory(t *testing.T) {
	switch os.Getenv(peakMemoryRole) {
	case "touch":
		runtime.KeepAlive(touch(helperMemory))
		return
	case "measure":
		fmt.Printf("peak=%d\n", runTouch(t))
		return
	case "measure-above-ballast":
		ballast := touch(4 * helperMemory)
		peak := runTouch(t)
		runtime.KeepAlive(ballast)
		fmt.Printf("peak=%d\n", peak)
		return
	}
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("the peak memory of a process is known only on Linux and macOS")
	}
	peak := measureInNewProcess(t, "measure")
	require.GreaterOrEqual(t, peak, uint64(helperMemory))
	require.Less(t, peak, uint64(1<<30))
	require.Less(t, measureInNewProcess(t, "measure-above-ballast"), uint64(4*helperMemory))
}

// touch allocates size bytes and writes a byte of each page, so that the pages are resident.
func touch(size int) []byte {
	data := make([]byte, size)
	for i := 0; i < len(data); i += os.Getpagesize() {
		data[i] = 1
	}
	return data
}

// runTouch runs a touch process with execWithEnv and returns its peak memory.
func runTouch(t *testing.T) uint64 {
	exe, err := os.Executable()
	require.NoError(t, err)
	client := NewClient(StaticBinary(exe))
	client.Stdout, client.Stderr = io.Discard, io.Discard
	env := append(os.Environ(), peakMemoryRole+"=touch")
	peak, err := client.execWithEnv(context.Background(), t.TempDir(), env, io.Discard, io.Discard,
		"-test.run=^TestExecWithEnvReturnsPeakMemory$")
	require.NoError(t, err)
	return peak
}

// measureInNewProcess runs the test as a process of role, and returns the peak that it prints.
func measureInNewProcess(t *testing.T, role string) uint64 {
	exe, err := os.Executable()
	require.NoError(t, err)
	cmd := exec.Command(exe, "-test.run=^TestExecWithEnvReturnsPeakMemory$")
	cmd.Env = append(os.Environ(), peakMemoryRole+"="+role)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	match := regexp.MustCompile(`peak=(\d+)`).FindSubmatch(out)
	require.NotNil(t, match, string(out))
	peak, err := strconv.ParseUint(string(match[1]), 10, 64)
	require.NoError(t, err)
	return peak
}

func TestRecordRunMemory(t *testing.T) {
	before := PeakRunMemory()
	recordRunMemory(before + 2)
	recordRunMemory(before + 1)
	require.Equal(t, before+2, PeakRunMemory())
}
