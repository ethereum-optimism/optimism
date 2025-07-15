package tests

import (
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	mtutil "github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded/testutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

type TestNamer[T any] func(testCase T) string

type InitializeStateFn[T any] func(testCase T, state *multithreaded.State, vm VersionedVMTestCase)
type SetExpectationsFn[T any] func(testCase T, expect *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult

type DiffTester[T any] struct {
	testNamer       TestNamer[T]
	stateOpts       []mtutil.StateOption
	initState       InitializeStateFn[T]
	setExpectations SetExpectationsFn[T]
}

func NewDiffTester[T any](testNamer TestNamer[T]) *DiffTester[T] {
	return &DiffTester[T]{
		testNamer: testNamer,
	}
}

func (d *DiffTester[T]) InitState(initStateFn InitializeStateFn[T], opts ...mtutil.StateOption) *DiffTester[T] {
	d.initState = initStateFn
	d.stateOpts = opts

	return d
}

func (d *DiffTester[T]) SetExpectations(setExpectationsFn SetExpectationsFn[T]) *DiffTester[T] {
	d.setExpectations = setExpectationsFn

	return d
}

func (d *DiffTester[T]) Run(t *testing.T, testCases []T, opts ...TestOption) {
	// Encapsulate core logic in run() for easier unit testing with the testRunner interface
	d.run(wrapT(t), testCases, opts...)
}

func (d *DiffTester[T]) run(t testRunner, testCases []T, opts ...TestOption) {
	if !d.isConfigValid(t) {
		t.Fatalf("DiffTester is misconfigured")
	}

	cfg := newTestConfig(t, opts...)
	for _, vm := range cfg.vms {
		for i, testCase := range testCases {
			randSeed := randomSeed(t, d.testNamer(testCase), i)
			mods := d.generateTestModifiers(t, testCase, vm, d.setExpectations, cfg, randSeed)
			for _, mod := range mods {
				testName := fmt.Sprintf("%v%v (%v)", d.testNamer(testCase), mod.name, vm.Name)
				t.Run(testName, func(t testcaseT) {
					t.Parallel()
					stateOpts := []mtutil.StateOption{mtutil.WithRandomization(randSeed)}
					stateOpts = append(stateOpts, d.stateOpts...)
					goVm := vm.VMFactory(cfg.po(), cfg.stdOut(), cfg.stdErr(), cfg.logger, stateOpts...)
					state := mtutil.GetMtState(t, goVm)

					// Set up state
					d.initState(testCase, state, vm)
					mod.stateMod(state)

					// Set up expectations
					expect := mtutil.NewExpectedState(t, state)
					execExpectation := d.setExpectations(testCase, expect, vm)
					mod.expectMod(expect)

					execExpectation.assertExpectedResult(t, goVm, vm, expect)
				})
			}
		}
	}
}

func (d *DiffTester[T]) isConfigValid(t testRunner) bool {
	isValid := true
	if d.initState == nil {
		t.Errorf("Must configure initial state via InitState()")
		isValid = false
	}
	if d.setExpectations == nil {
		t.Errorf("Must configure expectations via SetExpectations()")
		isValid = false
	}
	return isValid
}

type testModifier struct {
	name      string
	stateMod  func(state *multithreaded.State)
	expectMod func(expect *mtutil.ExpectedState)
}

func newTestModifier(name string) *testModifier {
	return &testModifier{
		name:      name,
		stateMod:  func(state *multithreaded.State) {},
		expectMod: func(expect *mtutil.ExpectedState) {},
	}
}

func (d *DiffTester[T]) generateTestModifiers(t require.TestingT, testCase T, vm VersionedVMTestCase, setExpectations SetExpectationsFn[T], cfg *TestConfig, randSeed int64) []*testModifier {
	modifiers := []*testModifier{
		newTestModifier(""), // Always return a noop
	}

	// Process expectations
	goVm := vm.VMFactory(nil, nil, nil, nil)
	state := mtutil.GetMtState(t, goVm)
	expect := mtutil.NewExpectedState(t, state)
	setExpectations(testCase, expect, vm)

	// Generate test modifiers based on expectations
	modifiers = append(modifiers, d.memReservationTestModifier(cfg, randSeed, expect)...)

	return modifiers
}

// memReservationTestModifier updates tests that write to memory, to ensure that memory reservations are handled correctly
func (d *DiffTester[T]) memReservationTestModifier(cfg *TestConfig, randSeed int64, expect *mtutil.ExpectedState) []*testModifier {
	var modifiers []*testModifier

	memTargets := expect.ExpectedMemoryWrites()
	if cfg.skipAutomaticMemoryReservationTests || len(memTargets) == 0 {
		// If we are explicitly skipping these mods, or memory is not written to at all, there is nothing to do
		return modifiers
	}

	for i, testCase := range memReservationTestCases {
		modifiers = append(modifiers, &testModifier{
			name: fmt.Sprintf(" [mod:%v]", testCase.name),
			stateMod: func(state *multithreaded.State) {
				r := testutil.NewRandHelper(randSeed*int64(i) + 10000)
				targetMemAddr := memTargets[r.Intn(len(memTargets))]
				effAddr := targetMemAddr & arch.AddressMask

				llAddress := effAddr + testCase.effAddrOffset
				llOwnerThread := state.GetCurrentThread().ThreadId
				if !testCase.matchThreadId {
					llOwnerThread += 1
				}

				state.LLReservationStatus = testCase.llReservationStatus
				state.LLAddress = llAddress
				state.LLOwnerThread = llOwnerThread
			},
			expectMod: func(expect *mtutil.ExpectedState) {
				if testCase.shouldClearReservation {
					expect.ExpectMemoryReservationCleared()
				}
			},
		})
	}

	return modifiers
}

type memReservationTestCase struct {
	name                   string
	llReservationStatus    multithreaded.LLReservationStatus
	matchThreadId          bool
	effAddrOffset          arch.Word
	shouldClearReservation bool
}

var memReservationTestCases []memReservationTestCase = []memReservationTestCase{
	{name: "matching reservation", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: true, shouldClearReservation: true},
	{name: "matching reservation, 64-bit", llReservationStatus: multithreaded.LLStatusActive64bit, matchThreadId: true, shouldClearReservation: true},
	{name: "matching reservation, unaligned", llReservationStatus: multithreaded.LLStatusActive32bit, effAddrOffset: 1, matchThreadId: true, shouldClearReservation: true},
	{name: "matching reservation, 64-bit, unaligned", llReservationStatus: multithreaded.LLStatusActive64bit, effAddrOffset: 5, matchThreadId: true, shouldClearReservation: true},
	{name: "matching reservation, diff thread", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: false, shouldClearReservation: true},
	{name: "matching reservation, diff thread, 64-bit", llReservationStatus: multithreaded.LLStatusActive64bit, matchThreadId: false, shouldClearReservation: true},
	{name: "mismatched reservation", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: true, effAddrOffset: 8, shouldClearReservation: false},
	{name: "mismatched reservation, 64-bit", llReservationStatus: multithreaded.LLStatusActive64bit, matchThreadId: true, effAddrOffset: 8, shouldClearReservation: false},
	{name: "mismatched reservation, diff thread", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: false, effAddrOffset: 8, shouldClearReservation: false},
	{name: "mismatched reservation, diff thread, 64-bit", llReservationStatus: multithreaded.LLStatusActive64bit, matchThreadId: false, effAddrOffset: 8, shouldClearReservation: false},
	{name: "no reservation, matching addr", llReservationStatus: multithreaded.LLStatusNone, matchThreadId: true, shouldClearReservation: true},
	{name: "no reservation, mismatched addr", llReservationStatus: multithreaded.LLStatusNone, matchThreadId: true, effAddrOffset: 8, shouldClearReservation: false},
}

func randomSeed(t require.TestingT, s string, extraData ...int) int64 {
	h := fnv.New64a()

	_, err := h.Write([]byte(s))
	require.NoError(t, err)
	for _, extra := range extraData {
		extraBytes := []byte(fmt.Sprintf("%d", extra))
		_, err := h.Write(extraBytes)
		require.NoError(t, err)
	}

	return int64(h.Sum64())
}

type TestConfig struct {
	vms    []VersionedVMTestCase
	po     func() mipsevm.PreimageOracle
	stdOut func() io.Writer
	stdErr func() io.Writer
	logger log.Logger
	// Allow consumer to control automated test generation
	skipAutomaticMemoryReservationTests bool
}

type TestOption func(*TestConfig)

func WithPreimageOracle(po func() mipsevm.PreimageOracle) TestOption {
	return func(tc *TestConfig) {
		tc.po = po
	}
}

func SkipAutomaticMemoryReservationTests() TestOption {
	return func(tc *TestConfig) {
		tc.skipAutomaticMemoryReservationTests = true
	}
}

func WithVm(vm VersionedVMTestCase) TestOption {
	return func(tc *TestConfig) {
		tc.vms = []VersionedVMTestCase{vm}
	}
}

func newTestConfig(t require.TestingT, opts ...TestOption) *TestConfig {
	testConfig := &TestConfig{
		vms:    GetMipsVersionTestCases(t),
		po:     func() mipsevm.PreimageOracle { return nil },
		stdOut: func() io.Writer { return os.Stdout },
		stdErr: func() io.Writer { return os.Stderr },
		logger: testutil.CreateLogger(),
	}

	for _, opt := range opts {
		opt(testConfig)
	}
	return testConfig
}

type ExpectedExecResult interface {
	assertExpectedResult(t testing.TB, vm mipsevm.FPVM, vmType VersionedVMTestCase, expect *mtutil.ExpectedState)
}

type normalExecResult struct{}

func ExpectNormalExecution() ExpectedExecResult {
	return normalExecResult{}
}

func (e normalExecResult) assertExpectedResult(t testing.TB, goVm mipsevm.FPVM, vmVersion VersionedVMTestCase, expect *mtutil.ExpectedState) {
	// Step the VM
	state := goVm.GetState()
	step := state.GetStep()
	stepWitness, err := goVm.Step(true)
	require.NoError(t, err)

	// Validate
	expect.Validate(t, state)
	testutil.ValidateEVM(t, stepWitness, step, goVm, vmVersion.StateHashFn, vmVersion.Contracts)
}

type vmPanicResult struct {
	panicMsg string
	evmError string
}

func ExpectVmPanic(goPanicMsg, evmRevertMsg string) ExpectedExecResult {
	return vmPanicResult{
		panicMsg: goPanicMsg,
		evmError: evmRevertMsg,
	}
}

func (e vmPanicResult) assertExpectedResult(t testing.TB, goVm mipsevm.FPVM, vmVersion VersionedVMTestCase, expect *mtutil.ExpectedState) {
	state := goVm.GetState()
	proofData := vmVersion.ProofGenerator(t, state)
	errMsg := testutil.CreateErrorStringMatcher(e.evmError)
	testutil.AssertEVMReverts(t, state, vmVersion.Contracts, nil, proofData, errMsg)
	require.PanicsWithValue(t, e.panicMsg, func() { _, _ = goVm.Step(false) })
}

type preimageOracleRevertResult struct {
	panicMsg       string
	preimageKey    [32]byte
	preimageValue  []byte
	preimageOffset arch.Word
}

func ExpectPreimageOraclePanic(preimageKey [32]byte, preimageValue []byte, preimageOffset arch.Word, panicMsg string) ExpectedExecResult {
	return preimageOracleRevertResult{
		panicMsg:       panicMsg,
		preimageKey:    preimageKey,
		preimageValue:  preimageValue,
		preimageOffset: preimageOffset,
	}
}

func (e preimageOracleRevertResult) assertExpectedResult(t testing.TB, goVm mipsevm.FPVM, vmVersion VersionedVMTestCase, expect *mtutil.ExpectedState) {
	require.PanicsWithValue(t, e.panicMsg, func() { _, _ = goVm.Step(true) })
	testutil.AssertPreimageOracleReverts(t, e.preimageKey, e.preimageValue, e.preimageOffset, vmVersion.Contracts)
}

type testcaseT interface {
	testing.TB
	Parallel()
}
type testFn func(testcaseT)

type testRunner interface {
	testing.TB
	Run(name string, fn testFn) bool
	Parallel()
}

// Adapt *testing.T to internal testRunner interface
type wrappedT struct{ *testing.T }

func (tr *wrappedT) Run(name string, fn testFn) bool {
	return tr.T.Run(name, func(t *testing.T) {
		fn(t)
	})
}

func (tr *wrappedT) Parallel() {
	tr.T.Parallel()
}

func wrapT(t *testing.T) testRunner { return &wrappedT{t} }
