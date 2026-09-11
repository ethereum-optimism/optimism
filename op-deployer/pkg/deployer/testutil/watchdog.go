package testutil

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"
)

// StartWatchdog crashes the test binary with a full goroutine dump if the
// calling test is still running after timeout. Use it in live-RPC tests whose
// blocking paths ignore context deadlines (e.g. a forked anvil waiting on a
// dead upstream): without it a hang stalls the whole CI shard until the
// no-output watchdog kills the job with no diagnostics.
func StartWatchdog(t *testing.T, timeout time.Duration) {
	timer := time.AfterFunc(timeout, func() {
		buf := make([]byte, 4<<20)
		n := runtime.Stack(buf, true)
		fmt.Fprintf(os.Stderr, "test %s exceeded %v watchdog timeout\n\n%s\n", t.Name(), timeout, buf[:n])
		panic(fmt.Sprintf("test %s exceeded %v watchdog timeout", t.Name(), timeout))
	})
	t.Cleanup(func() { timer.Stop() })
}
