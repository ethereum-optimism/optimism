package op_e2e

import (
	"crypto/md5"
	"os"
	"regexp"
	"strconv"
	"sync"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/stretchr/testify/require"
)

var enableParallelTesting bool = os.Getenv("OP_E2E_DISABLE_PARALLEL") != "true"

type testopts struct {
	executor uint64
}

var (
	testBaseExp      = regexp.MustCompile("^Test[^/]+")
	testRegistryInst = testRegistry{
		tests: map[string]uint64{},
	}
)

type testRegistry struct {
	tests map[string]uint64
	mutex sync.Mutex
}

// registerTest ensures that if a test is assigned to an executor, that its
// parent test is assigned to the same executor, or not assigned to an executor
// at all
func (tr *testRegistry) registerTest(t e2eutils.TestingBase, name string, executor uint64) {
	baseName := testBaseExp.FindString(name)
	tr.mutex.Lock()
	defer tr.mutex.Unlock()
	if baseExecutor, ok := tr.tests[baseName]; ok {
		require.Equal(
			t, baseExecutor, executor,
			"base test for %s executes only on %d but %s requested to execute on %d",
			baseName, baseExecutor, name, executor,
		)
	} else if name == baseName {
		tr.tests[name] = executor
	}
}

func InitParallel(t e2eutils.TestingBase, args ...func(t e2eutils.TestingBase, opts *testopts)) {
	t.Helper()
	if enableParallelTesting {
		t.Parallel()
	}

	info := getExecutorInfo(t)
	tName := t.Name()
	tHash := md5.Sum([]byte(tName))
	executor := uint64(tHash[0]) % info.total
	opts := &testopts{
		executor: executor,
	}
	for _, arg := range args {
		arg(t, opts)
	}
	testRegistryInst.registerTest(t, tName, opts.executor)
	checkExecutor(t, info, opts.executor)
}

func UsesCannon(t e2eutils.TestingBase, opts *testopts) {
	if os.Getenv("OP_E2E_CANNON_ENABLED") == "false" {
		t.Skip("Skipping cannon test")
	}
}

func SkipOnFPAC(t e2eutils.TestingBase, opts *testopts) {
	if e2eutils.UseFPAC() {
		t.Skip("Skipping test for FPAC")
	}
}

func SkipOnNotFPAC(t e2eutils.TestingBase, opts *testopts) {
	if !e2eutils.UseFPAC() {
		t.Skip("Skipping test for non-FPAC")
	}
}

//	UseExecutor allows manually splitting tests between circleci executors
//
// Tests default to run on the first executor but can be moved to the second with:
// InitParallel(t, UseExecutor(1))
// Any tests assigned to an executor greater than the number available automatically use the last executor.
// Executor indexes start from 0
func UseExecutor(assignedIdx uint64) func(t e2eutils.TestingBase, opts *testopts) {
	return func(t e2eutils.TestingBase, opts *testopts) {
		opts.executor = assignedIdx
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
