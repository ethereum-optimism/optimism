package op_e2e

import (
	"os"
	"strconv"
	"testing"
)

var enableParallelTesting bool = os.Getenv("OP_E2E_DISABLE_PARALLEL") != "true"

type testopts struct {
	executor uint64
}

func InitParallel(t *testing.T, args ...func(t *testing.T, opts *testopts)) {
	t.Helper()
	if enableParallelTesting {
		t.Parallel()
	}

	opts := &testopts{}
	for _, arg := range args {
		arg(t, opts)
	}
	checkExecutor(t, opts.executor)
}

func UsesCannon(t *testing.T, opts *testopts) {
	if os.Getenv("OP_E2E_CANNON_ENABLED") == "false" {
		t.Skip("Skipping cannon test")
	}
}

//	UseExecutor allows manually splitting tests between circleci executors
//
// Tests default to run on the first executor but can be moved to the second with:
// InitParallel(t, UseExecutor(1))
// Any tests assigned to an executor greater than the number available automatically use the last executor.
// Executor indexes start from 0
func UseExecutor(assignedIdx uint64) func(t *testing.T, opts *testopts) {
	return func(t *testing.T, opts *testopts) {
		opts.executor = assignedIdx
	}
}

func checkExecutor(t *testing.T, assignedIdx uint64) {
	envTotal := os.Getenv("CIRCLE_NODE_TOTAL")
	envIdx := os.Getenv("CIRCLE_NODE_INDEX")
	if envTotal == "" || envIdx == "" {
		// Not using test splitting, so ignore assigned executor
		t.Logf("Running test. Test splitting not in use.")
		return
	}
	total, err := strconv.ParseUint(envTotal, 10, 0)
	if err != nil {
		t.Fatalf("Could not parse CIRCLE_NODE_TOTAL env var %v: %v", envTotal, err)
	}
	idx, err := strconv.ParseUint(envIdx, 10, 0)
	if err != nil {
		t.Fatalf("Could not parse CIRCLE_NODE_INDEX env var %v: %v", envIdx, err)
	}
	if assignedIdx >= total && idx == total-1 {
		t.Logf("Running test. Current executor (%v) is the last executor and assigned executor (%v) >= total executors (%v).", idx, assignedIdx, total)
		return
	}
	if idx == assignedIdx {
		t.Logf("Running test. Assigned executor (%v) matches current executor (%v) of total (%v)", assignedIdx, idx, total)
		return
	}
	t.Skipf("Skipping test. Assigned executor %v, current executor %v of total %v", assignedIdx, idx, total)
}
