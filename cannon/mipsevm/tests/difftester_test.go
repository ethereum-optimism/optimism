package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	mtutil "github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded/testutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

func TestDiffTester_Run_SimpleTest(t *testing.T) {
	// Run simple noop instruction (0x0)
	testCases := []simpleTestCase{
		{name: "a"},
		{name: "b"},
	}

	for _, useCorrectReturnExpectation := range []bool{true, false} {
		testName := fmt.Sprintf("useCorrectReturnExpectation=%v", useCorrectReturnExpectation)
		t.Run(testName, func(t *testing.T) {
			initStateCalled := make(map[string]int)
			initState := func(t require.TestingT, testCase simpleTestCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
				initStateCalled[testCase.name] += 1
				testutil.StoreInstruction(state.GetMemory(), state.GetPC(), testCase.insn)
			}

			expectationsCalled := make(map[string]int)
			setExpectations := func(t require.TestingT, testCase simpleTestCase, expect *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
				expectationsCalled[testCase.name] += 1
				expect.ExpectStep()

				if useCorrectReturnExpectation {
					return ExpectNormalExecution()
				} else {
					return ExpectVmPanic("oops", "oops")
				}
			}

			versions := GetMipsVersionTestCases(t)
			expectedTestCases := generateExpectedTestCases(testCases, versions)

			// Run tests
			tRunner := newMockTestRunner(t)
			NewDiffTester(testNamer).
				InitState(initState).
				SetExpectations(setExpectations).
				run(tRunner, testCases)

			// Validate that we invoked initState and setExpectations as expected
			for _, c := range testCases {
				expectedCalls := len(versions)
				require.Equal(t, expectedCalls, initStateCalled[c.name])
				require.Equal(t, expectedCalls, expectationsCalled[c.name])
			}

			// Validate that tests ran and passed as expected
			require.Equal(t, len(tRunner.childTestMocks), len(expectedTestCases))
			for _, testCase := range expectedTestCases {
				failed, err := tRunner.testFailedOrPanicked(testCase)
				require.NoError(t, err)
				require.Equal(t, failed, !useCorrectReturnExpectation, "Expected test '%v' status failed = %v", testCase, !useCorrectReturnExpectation)
			}
		})
	}
}

func TestDiffTester_Run_WithSteps(t *testing.T) {
	outterCases := []struct {
		name          string
		steps         int
		expectedSteps int
	}{
		{name: "0 steps", steps: 0, expectedSteps: 1},
		{name: "negative steps", steps: -1, expectedSteps: 1},
		{name: "1 step", steps: 1, expectedSteps: 1},
		{name: "2 step", steps: 2, expectedSteps: 2},
		{name: "3 step", steps: 3, expectedSteps: 3},
	}

	// Run simple noop instruction (0x0)
	cases := []simpleTestCase{
		{name: "a", insn: 0x0},
	}

	for _, oc := range outterCases {
		t.Run(oc.name, func(t *testing.T) {
			initStateCalled := make(map[string]int)
			initState := func(t require.TestingT, testCase simpleTestCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
				initStateCalled[testCase.name] += 1
				testutil.StoreInstruction(state.GetMemory(), state.GetPC(), testCase.insn)
			}

			expectationsCalled := make(map[string]int)
			setExpectations := func(t require.TestingT, testCase simpleTestCase, expect *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
				expectationsCalled[testCase.name] += 1
				expect.ExpectStep()
				return ExpectNormalExecution()
			}

			versions := GetMipsVersionTestCases(t)
			expectedTestCases := generateExpectedTestCases(cases, versions)

			// Run tests
			tRunner := newMockTestRunner(t)
			NewDiffTester(testNamer).
				InitState(initState).
				SetExpectations(setExpectations).
				run(tRunner, cases, WithSteps(oc.steps))

			// Validate that we invoked initState and setExpectations as expected
			for _, c := range cases {
				initCalls := len(versions)
				expectCalls := oc.expectedSteps * len(versions)
				require.Equal(t, initCalls, initStateCalled[c.name])
				require.Equal(t, expectCalls, expectationsCalled[c.name])
			}

			// Validate that tests ran and passed as expected
			require.Equal(t, len(tRunner.childTestMocks), len(expectedTestCases))
			for _, testCase := range expectedTestCases {
				failed, err := tRunner.testFailedOrPanicked(testCase)
				require.NoError(t, err)
				require.Equal(t, false, failed)
			}
		})
	}
}

func TestDiffTester_Run_WithMemModifications(t *testing.T) {
	// Test store word (sw), which modifies memory
	baseReg := uint32(9)
	rtReg := uint32(8)
	opcode := uint32(0x2b)
	base := arch.Word(0x1000)
	imm := uint32(8)
	effAddr := base + arch.Word(imm)
	insn := opcode<<26 | baseReg<<21 | rtReg<<16 | imm
	pc := arch.Word(0)

	testCases := []simpleTestCase{
		{name: "a", insn: insn},
		{name: "b", insn: insn},
	}

	for _, skipAutomaticMemTests := range []bool{true, false} {
		testName := fmt.Sprintf("skipAutomaticMemTests=%v", skipAutomaticMemTests)
		t.Run(testName, func(t *testing.T) {

			initStateCalled := make(map[string]int)
			initState := func(t require.TestingT, tt simpleTestCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
				initStateCalled[tt.name] += 1
				storeInsnWithCache(state, goVm, pc, tt.insn)
				state.GetMemory().SetWord(effAddr, 0xAA_BB_CC_DD_A1_B1_C1_D1)
				state.GetRegistersRef()[rtReg] = 0x11_22_33_44_55_66_77_88
				state.GetRegistersRef()[baseReg] = base
			}

			expectationsCalled := make(map[string]int)
			setExpectations := func(t require.TestingT, tt simpleTestCase, expect *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
				expectationsCalled[tt.name] += 1
				expect.ExpectStep()
				expect.ExpectMemoryWrite(effAddr, 0x55_66_77_88_A1_B1_C1_D1)
				return ExpectNormalExecution()
			}

			versions := GetMipsVersionTestCases(t)
			var mods []string
			if !skipAutomaticMemTests {
				for _, memTestCase := range memReservationTestCases {
					modName := fmt.Sprintf(" [mod:%v]", memTestCase.name)
					mods = append(mods, modName)
				}
			}
			expectedTestCases := generateExpectedTestCases(testCases, versions, mods...)

			// Run tests
			var opts []TestOption
			if skipAutomaticMemTests {
				opts = append(opts, SkipAutomaticMemoryReservationTests())
			}

			tRunner := newMockTestRunner(t)
			NewDiffTester(testNamer).
				InitState(initState, mtutil.WithPCAndNextPC(pc)).
				SetExpectations(setExpectations).
				run(tRunner, testCases, opts...)

			// Validate that we invoked initState and setExpectations as expected
			for _, c := range testCases {
				expectedCalls := len(versions) * (len(mods) + 1)
				require.Equal(t, expectedCalls, initStateCalled[c.name])
				require.Equal(t, expectedCalls, expectationsCalled[c.name])
			}

			// Validate that tests ran and passed
			require.Equal(t, len(tRunner.childTestMocks), len(expectedTestCases))
			for _, testCase := range expectedTestCases {
				failed, err := tRunner.testFailedOrPanicked(testCase)
				require.NoError(t, err)
				require.False(t, failed, "Test '%v' should pass", testCase)
			}
		})
	}
}

func TestDiffTester_Run_WithPanic(t *testing.T) {
	// Set up test to panic - invoke syscall with invalid syscallNum 0
	testCases := []simpleTestCase{
		{name: "a", insn: syscallInsn},
	}
	syscallNum := arch.Word(0)

	for _, useCorrectReturnExpectation := range []bool{true, false} {
		testName := fmt.Sprintf("useCorrectReturnExpectation=%v", useCorrectReturnExpectation)
		t.Run(testName, func(t *testing.T) {
			initStateCalled := make(map[string]int)
			initState := func(t require.TestingT, testCase simpleTestCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
				initStateCalled[testCase.name] += 1
				testutil.StoreInstruction(state.GetMemory(), state.GetPC(), testCase.insn)
				state.GetRegistersRef()[2] = syscallNum
			}

			expectationsCalled := make(map[string]int)
			setExpectations := func(t require.TestingT, testCase simpleTestCase, expect *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
				expectationsCalled[testCase.name] += 1
				expect.ExpectStep()

				if useCorrectReturnExpectation {
					return ExpectVmPanic("unrecognized syscall: 0", "unimplemented syscall")
				} else {
					return ExpectNormalExecution()
				}
			}

			versions := GetMipsVersionTestCases(t)
			expectedTestCases := generateExpectedTestCases(testCases, versions)

			// Run tests
			tRunner := newMockTestRunner(t)
			NewDiffTester(testNamer).
				InitState(initState).
				SetExpectations(setExpectations).
				run(tRunner, testCases)

			// Validate that we invoked initState and setExpectations as expected
			for _, c := range testCases {
				expectedCalls := len(versions)
				require.Equal(t, expectedCalls, initStateCalled[c.name])
				require.Equal(t, expectedCalls, expectationsCalled[c.name])
			}

			// Validate that tests ran and passed as expected
			require.Equal(t, len(tRunner.childTestMocks), len(expectedTestCases))
			for _, testCase := range expectedTestCases {
				if useCorrectReturnExpectation {
					failed, err := tRunner.testFailedOrPanicked(testCase)
					require.NoError(t, err)
					require.False(t, failed, "Test '%v' should pass", testCase)
				} else {
					panicked, err := tRunner.testPanicked(testCase)
					require.NoError(t, err)
					require.True(t, panicked, "Test '%v' should panic", testCase)
				}
			}
		})
	}
}

func TestDiffTester_Run_WithVm(t *testing.T) {
	// Run simple noop instruction (0x0)
	testCases := []simpleTestCase{
		{name: "a"},
		{name: "b"},
	}

	initStateCalled := make(map[string]int)
	initState := func(t require.TestingT, testCase simpleTestCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		initStateCalled[testCase.name] += 1
		testutil.StoreInstruction(state.GetMemory(), state.GetPC(), testCase.insn)
	}

	expectationsCalled := make(map[string]int)
	setExpectations := func(t require.TestingT, testCase simpleTestCase, expect *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expectationsCalled[testCase.name] += 1
		expect.ExpectStep()

		return ExpectNormalExecution()
	}

	vm := GetMipsVersionTestCases(t)[0]
	versions := []VersionedVMTestCase{vm}
	expectedTestCases := generateExpectedTestCases(testCases, versions)

	// Run tests
	tRunner := newMockTestRunner(t)
	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		run(tRunner, testCases, WithVm(vm))

	// Validate that we invoked initState and setExpectations as expected
	for _, c := range testCases {
		require.Equal(t, 1, initStateCalled[c.name])
		require.Equal(t, 1, expectationsCalled[c.name])
	}

	// Validate that we ran the expected tests
	require.Equal(t, len(tRunner.childTestMocks), len(expectedTestCases))
}

// Test case struct for simple test scenarios
type simpleTestCase struct {
	name string
	insn uint32
}

// Test helper to create a test namer
func testNamer(testCase simpleTestCase) string {
	return testCase.name
}

// generateExpectedTestCases Generates expected test cases that are derived from the provided `cases`
func generateExpectedTestCases(cases []simpleTestCase, versions []VersionedVMTestCase, expectedMods ...string) []string {
	var expectedTestRuns []string
	expectedMods = append(expectedMods, "")
	for _, vm := range versions {
		for _, testCase := range cases {
			for _, mod := range expectedMods {
				testName := fmt.Sprintf("%v%v (%v)", testCase.name, mod, vm.Name)
				expectedTestRuns = append(expectedTestRuns, testName)
			}
		}
	}
	return expectedTestRuns
}

type mockTestRunner struct {
	*mockT
	childTestMocks map[string]*mockT
}

var _ testRunner = (*mockTestRunner)(nil)

func newMockTestRunner(tb testing.TB) *mockTestRunner {
	t := &mockT{name: "MockTestRunner", t: tb, debugTestFailures: false}
	childMocks := make(map[string]*mockT)
	return &mockTestRunner{mockT: t, childTestMocks: childMocks}
}

func (m *mockTestRunner) Run(name string, fn testFn) bool {
	t := &mockT{name: name, t: m.t, debugTestFailures: m.debugTestFailures}
	defer func() {
		if err := recover(); err != nil {
			m.t.Logf("Test panicked: %v", err)
			t.panicked = true
		}
	}()

	m.childTestMocks[name] = t

	fn(t)
	return t.failed
}

func (m *mockTestRunner) Parallel() {}

func (m *mockTestRunner) testFailedOrPanicked(testName string) (bool, error) {
	runner, ok := m.childTestMocks[testName]
	if !ok {
		return false, fmt.Errorf("test '%v' not found", testName)
	}
	return runner.failed || runner.panicked, nil
}

func (m *mockTestRunner) testPanicked(testName string) (bool, error) {
	runner, ok := m.childTestMocks[testName]
	if !ok {
		return false, fmt.Errorf("test '%v' not found", testName)
	}
	return runner.panicked, nil
}

type mockT struct {
	testing.TB
	t                 testing.TB
	name              string
	failed            bool
	panicked          bool
	debugTestFailures bool
}

func (m *mockT) Error(args ...any) {
	m.failed = true
}

func (m *mockT) Errorf(format string, args ...any) {
	if m.debugTestFailures {
		m.t.Logf("[TEST ERROR]"+format, args...)
	}
	m.failed = true
}

func (m *mockT) Fail() {
	m.failed = true
}

func (m *mockT) FailNow() {
	m.failed = true
}

func (m *mockT) Failed() bool {
	return m.failed
}

func (m *mockT) Fatal(args ...any) {
	m.failed = true
}

func (m *mockT) Fatalf(format string, args ...any) {
	if m.debugTestFailures {
		m.t.Logf("[TEST FATAL]"+format, args...)
	}
	m.failed = true
}

func (m *mockT) Cleanup(f func()) {}

func (m *mockT) Helper() {}

func (m *mockT) Log(args ...any) {}

func (m *mockT) Logf(format string, args ...any) {}

func (m *mockT) Name() string {
	return m.name
}

func (m *mockT) Setenv(key, value string) {}

func (m *mockT) Chdir(dir string) {}

func (m *mockT) Skip(args ...any) {}

func (m *mockT) SkipNow() {}

func (m *mockT) Skipf(format string, args ...any) {}

func (m *mockT) Skipped() bool {
	return false
}

func (m *mockT) TempDir() string {
	return ""
}

func (m *mockT) Context() context.Context {
	return context.Background()
}

func (m *mockT) Parallel() {}

var _ testing.TB = (*mockT)(nil)
