package op_e2e

import (
	"crypto/md5"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/config"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
)

func RunMain(m *testing.M) {
	if config.ExternalL2Shim != "" {
		fmt.Println("Running tests with external L2 process adapter at ", config.ExternalL2Shim)
		// As these are integration tests which launch many other processes, the
		// default parallelism makes the tests flaky.  This change aims to
		// reduce the flakiness of these tests.
		maxProcs := runtime.NumCPU() / 4
		if maxProcs == 0 {
			maxProcs = 1
		}
		runtime.GOMAXPROCS(maxProcs)
	}

	os.Exit(m.Run())
}

var enableParallelTesting bool = os.Getenv("OP_E2E_DISABLE_PARALLEL") != "true"

func InitParallel(t e2eutils.TestingBase, args ...func(t e2eutils.TestingBase)) {
	t.Helper()
	if enableParallelTesting {
		t.Parallel()
	}
	for _, arg := range args {
		arg(t)
	}
	autoAllocateExecutor(t)
}

// isSubTest determines if the test is a sub-test or top level test.
// It does this by checking if the test name contains /
// This is not a particularly great way check, but appears to be the only option currently.
func isSubTest(t e2eutils.TestingBase) bool {
	return strings.Contains(t.Name(), "/")
}

func autoAllocateExecutor(t e2eutils.TestingBase) {
	if isSubTest(t) {
		// Always run subtests, they only start on the same executor as their parent.
		return
	}
	info := getExecutorInfo(t)
	tName := t.Name()
	tHash := md5.Sum([]byte(tName))
	executor := uint64(tHash[0]) % info.total
	checkExecutor(t, info, executor)
}

func UsesCannon(t e2eutils.TestingBase) {
	if os.Getenv("OP_E2E_CANNON_ENABLED") == "false" {
		t.Skip("Skipping cannon test")
	}
}

func SkipOnFaultProofs(t e2eutils.TestingBase) {
	if e2eutils.UseFaultProofs() {
		t.Skip("Skipping test for fault proofs")
	}
}

func SkipOnL2OO(t e2eutils.TestingBase) {
	if e2eutils.UseL2OO() {
		t.Skip("Skipping test for L2OO")
	}
}

type executorInfo struct {
	total      uint64
	idx        uint64
	splitInUse bool
}

func getExecutorInfo(t e2eutils.TestingBase) executorInfo {
	var info executorInfo
	envTotal := os.Getenv("CIRCLE_NODE_TOTAL")
	envIdx := os.Getenv("CIRCLE_NODE_INDEX")
	if envTotal == "" || envIdx == "" {
		// Not using test splitting, so ignore assigned executor
		t.Logf("Test splitting not in use.")
		info.total = 1
		return info
	}
	total, err := strconv.ParseUint(envTotal, 10, 0)
	if err != nil {
		t.Fatalf("Could not parse CIRCLE_NODE_TOTAL env var %v: %v", envTotal, err)
	}
	idx, err := strconv.ParseUint(envIdx, 10, 0)
	if err != nil {
		t.Fatalf("Could not parse CIRCLE_NODE_INDEX env var %v: %v", envIdx, err)
	}

	info.total = total
	info.idx = idx
	info.splitInUse = true
	return info
}

func checkExecutor(t e2eutils.TestingBase, info executorInfo, assignedIdx uint64) {
	if !info.splitInUse {
		t.Logf("Test splitting not in use.")
		return
	}

	if assignedIdx >= info.total && info.idx == info.total-1 {
		t.Logf("Running test. Current executor (%v) is the last executor and assigned executor (%v) >= total executors (%v).", info.idx, assignedIdx, info.total)
		return
	}
	if info.idx == assignedIdx {
		t.Logf("Running test. Assigned executor (%v) matches current executor (%v) of total (%v)", assignedIdx, info.idx, info.total)
		return
	}
	t.Skipf("Skipping test. Assigned executor %v, current executor %v of total %v", assignedIdx, info.idx, info.total)
}
