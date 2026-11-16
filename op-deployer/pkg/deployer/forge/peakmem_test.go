package forge

import (
	"context"
	"io"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// helperMemory is what the helper process of TestExecWithEnvReturnsPeakMemory touches.
const helperMemory = 64 << 20

// TestExecWithEnvReturnsPeakMemory checks the unit of the peak: in KiB on Linux and in bytes on
// macOS, either would be off by a factor of 1024 if read as the other.
func TestExecWithEnvReturnsPeakMemory(t *testing.T) {
	if os.Getenv("FORGE_TEST_PEAK_MEMORY_HELPER") != "" {
		data := make([]byte, helperMemory)
		for i := 0; i < len(data); i += os.Getpagesize() {
			data[i] = 1
		}
		runtime.KeepAlive(data)
		return
	}
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("the peak memory of a process is known only on Linux and macOS")
	}
	client := NewClient(StaticBinary(os.Args[0]))
	client.Stdout, client.Stderr = io.Discard, io.Discard
	env := append(os.Environ(), "FORGE_TEST_PEAK_MEMORY_HELPER=1")
	peak, err := client.execWithEnv(context.Background(), t.TempDir(), env, io.Discard, io.Discard,
		"-test.run=^TestExecWithEnvReturnsPeakMemory$")
	require.NoError(t, err)
	require.GreaterOrEqual(t, peak, uint64(helperMemory))
	require.Less(t, peak, uint64(1<<30))
}

func TestRecordRunMemory(t *testing.T) {
	before := PeakRunMemory()
	recordRunMemory(before + 2)
	recordRunMemory(before + 1)
	require.Equal(t, before+2, PeakRunMemory())
}
