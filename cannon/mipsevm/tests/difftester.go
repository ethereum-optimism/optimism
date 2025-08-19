package tests

import (
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	mtutil "github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded/testutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

type TestNamer[T any] func(testCase T) string

func NoopTestNamer[T any](c T) string {
	return ""
}

type SimpleInitializeStateFn func(t require.TestingT, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM)
type SimpleSetExpectationsFn func(t require.TestingT, expect *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult
type SimplePostStepCheckFn func(t require.TestingT, vm VersionedVMTestCase, deps *TestDependencies, witness *mipsevm.StepWitness)

type soloTestCase struct {
	name string
}

type SimpleDiffTester struct {
	diffTester DiffTester[soloTestCase]
}

// NewSimpleDiffTester returns a DiffTester designed to run only a single default test case
func NewSimpleDiffTester() *SimpleDiffTester {
	return &SimpleDiffTester{
		diffTester: *NewDiffTester(func(t soloTestCase) string {
			return t.name
		}),
	}
}

func (d *SimpleDiffTester) InitState(initStateFn SimpleInitializeStateFn, opts ...mtutil.StateOption) *SimpleDiffTester {
	wrappedFn := func(t require.TestingT, _ soloTestCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		initStateFn(t, state, vm, r, goVm)
	}
	d.diffTester.InitState(wrappedFn, opts...)
	return d
}

func (d *SimpleDiffTester) SetExpectations(setExpectationsFn SimpleSetExpectationsFn) *SimpleDiffTester {
	wrappedFn := func(t require.TestingT, testCase soloTestCase, expect *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		return setExpectationsFn(t, expect, vm)
	}
	d.diffTester.SetExpectations(wrappedFn)

	return d
}

func (d *SimpleDiffTester) PostCheck(postStepCheckFn SimplePostStepCheckFn) *SimpleDiffTester {
	wrappedFn := func(t require.TestingT, testCase soloTestCase, vm VersionedVMTestCase, deps *TestDependencies, wit *mipsevm.StepWitness) {
		postStepCheckFn(t, vm, deps, wit)
	}
	d.diffTester.PostCheck(wrappedFn)

	return d
}

func (d *SimpleDiffTester) Run(t *testing.T, opts ...TestOption) {
	singleTestCase := []soloTestCase{
		{name: "solo test case"},
	}
	d.diffTester.run(wrapT(t), singleTestCase, opts...)
}

type InitializeStateFn[T any] func(t require.TestingT, testCase T, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM)
type SetExpectationsFn[T any] func(t require.TestingT, testCase T, expect *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult
type PostStepCheckFn[T any] func(t require.TestingT, testCase T, vm VersionedVMTestCase, deps *TestDependencies, witness *mipsevm.StepWitness)

type DiffTester[T any] struct {
	testNamer       TestNamer[T]
	stateOpts       []mtutil.StateOption
	initState       InitializeStateFn[T]
	setExpectations SetExpectationsFn[T]
	postStepCheck   PostStepCheckFn[T]
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

func (d *DiffTester[T]) PostCheck(postStepCheckFn PostStepCheckFn[T]) *DiffTester[T] {
	d.postStepCheck = postStepCheckFn

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
			randSeed := cfg.randomSeed
			if randSeed == 0 {
				randSeed = randomSeed(t, d.testNamer(testCase), i)
			}
			mods := d.generateTestModifiers(t, testCase, vm, cfg, randSeed)
			for _, mod := range mods {
				testName := fmt.Sprintf("%v%v (%v)", d.testNamer(testCase), mod.name, vm.Name)
				t.Run(testName, func(t testcaseT) {
					t.Parallel()

					setup := mod.cachedSetup
					if setup == nil {
						setup = d.newTestSetup(t, testCase, vm, cfg, randSeed, mod)
					}

					expect := setup.expect
					execExpectation := setup.expectedResult
					var witness *mipsevm.StepWitness
					for i := 0; i < cfg.steps; i++ {
						if i > 0 {
							// After the initial step, we need to set up our expectations again
							expect = d.expectedState(t, setup.state)
							execExpectation = d.setExpectations(t, testCase, expect, vm)
						}

						witness = execExpectation.assertExpectedResult(t, setup.goVm, vm, expect, cfg)
					}

					// Run post-step checks
					if d.postStepCheck != nil {
						d.postStepCheck(t, testCase, vm, setup.deps, witness)
					}
				})
			}
		}
	}
}

func (d *DiffTester[T]) newTestSetup(t require.TestingT, testCase T, vm VersionedVMTestCase, cfg *TestConfig, randSeed int64, mod *testModifier) *testSetup {
	testDeps := cfg.testDependencies()

	stateOpts := []mtutil.StateOption{mtutil.WithRandomization(randSeed)}
	stateOpts = append(stateOpts, d.stateOpts...)
	goVm := vm.VMFactory(testDeps.po, testDeps.stdOut, testDeps.stdErr, testDeps.logger, stateOpts...)

	state := mtutil.GetMtState(t, goVm)
	d.initState(t, testCase, state, vm, testutil.NewRandHelper(randSeed*2), goVm)
	if mod != nil {
		mod.stateMod(state)
	}

	expect := d.expectedState(t, state)
	if mod != nil {
		mod.expectMod(expect)
	}
	expectedResult := d.setExpectations(t, testCase, expect, vm)

	return &testSetup{
		deps:           testDeps,
		goVm:           goVm,
		state:          state,
		expect:         expect,
		expectedResult: expectedResult,
	}
}

func (d *DiffTester[T]) expectedState(t require.TestingT, state *multithreaded.State) *mtutil.ExpectedState {
	if mtutil.ActiveThreadCount(state) == 0 {
		// State is invalid, just return an empty expectation
		// We expect some tests to set up invalid states
		return &mtutil.ExpectedState{}
	}
	return mtutil.NewExpectedState(t, state)
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
	name        string
	stateMod    func(state *multithreaded.State)
	expectMod   func(expect *mtutil.ExpectedState)
	cachedSetup *testSetup
}

func newTestModifier(name string, cachedSetup *testSetup) *testModifier {
	return &testModifier{
		name:        name,
		stateMod:    func(state *multithreaded.State) {},
		expectMod:   func(expect *mtutil.ExpectedState) {},
		cachedSetup: cachedSetup,
	}
}

func (d *DiffTester[T]) generateTestModifiers(t require.TestingT, testCase T, vm VersionedVMTestCase, cfg *TestConfig, randSeed int64) []*testModifier {
	// Set up state
	setup := d.newTestSetup(t, testCase, vm, cfg, randSeed, nil)

	// Build modifiers array, start with the original case (noop modification)
	modifiers := []*testModifier{
		newTestModifier("", setup), // Always return a noop
	}

	// Generate test modifiers based on expectations
	modifiers = append(modifiers, d.memReservationTestModifier(cfg, randSeed, setup.expect)...)

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

type testSetup struct {
	deps           *TestDependencies
	goVm           mipsevm.FPVM
	state          *multithreaded.State
	expect         *mtutil.ExpectedState
	expectedResult ExpectedExecResult
}

type TestDependencies struct {
	po     mipsevm.PreimageOracle
	stdOut io.Writer
	stdErr io.Writer
	logger log.Logger
}

type TestConfig struct {
	vms   []VersionedVMTestCase
	steps int
	// Dependencies
	po     func() mipsevm.PreimageOracle
	stdOut func() io.Writer
	stdErr func() io.Writer
	logger log.Logger
	// no-tracer by default, but see test_util.MarkdownTracer
	tracingHooks *tracing.Hooks
	// Allow consumer to control automated test generation
	skipAutomaticMemoryReservationTests bool
	// Allow consumer to configure a random seed, if not configured (equal to 0) one will be generated
	randomSeed int64
}

func (c *TestConfig) testDependencies() *TestDependencies {
	return &TestDependencies{
		po:     c.po(),
		stdOut: c.stdOut(),
		stdErr: c.stdErr(),
		logger: c.logger,
	}
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

func WithVms(vms []VersionedVMTestCase) TestOption {
	return func(tc *TestConfig) {
		tc.vms = vms
	}
}

func WithRandomSeed(seed int64) TestOption {
	return func(tc *TestConfig) {
		tc.randomSeed = seed
	}
}

// WithTracingHooks Sets tracing hooks - see: testutil.MarkdownTracer
func WithTracingHooks(hooks *tracing.Hooks) TestOption {
	return func(tc *TestConfig) {
		tc.tracingHooks = hooks
	}
}

func WithSteps(steps int) TestOption {
	return func(tc *TestConfig) {
		if steps < 1 {
			steps = 1
		}
		tc.steps = steps
	}
}

func newTestConfig(t require.TestingT, opts ...TestOption) *TestConfig {
	testConfig := &TestConfig{
		po:     func() mipsevm.PreimageOracle { return nil },
		stdOut: func() io.Writer { return os.Stdout },
		stdErr: func() io.Writer { return os.Stderr },
		logger: testutil.CreateLogger(),
		steps:  1,
	}

	for _, opt := range opts {
		opt(testConfig)
	}

	// Generating vm versions is expensive, only do it if necessary
	if testConfig.vms == nil {
		testConfig.vms = GetMipsVersionTestCases(t)
	}

	return testConfig
}

type ExpectedExecResult interface {
	assertExpectedResult(t testing.TB, vm mipsevm.FPVM, vmType VersionedVMTestCase, expect *mtutil.ExpectedState, cfg *TestConfig) *mipsevm.StepWitness
}

type normalExecResult struct{}

func ExpectNormalExecution() ExpectedExecResult {
	return normalExecResult{}
}

func (e normalExecResult) assertExpectedResult(t testing.TB, goVm mipsevm.FPVM, vmVersion VersionedVMTestCase, expect *mtutil.ExpectedState, cfg *TestConfig) *mipsevm.StepWitness {
	// Step the VM
	state := goVm.GetState()
	step := state.GetStep()
	stepWitness, err := goVm.Step(true)
	require.NoError(t, err)

	// Validate
	expect.Validate(t, state)
	testutil.ValidateEVM(t, stepWitness, step, goVm, vmVersion.StateHashFn, vmVersion.Contracts)

	return stepWitness
}

type vmPanicResult struct {
	panicValue           interface{}
	evmErrorMatcher      testutil.ErrMatcher
	memoryProofAddresses []arch.Word
	proofData            []byte
}

type VMPanicTestOption func(*vmPanicResult)

func WithProofData(proofData []byte) VMPanicTestOption {
	return func(vmPanicResult *vmPanicResult) {
		vmPanicResult.proofData = proofData
	}
}

func WithMemoryProofAddr(addr arch.Word) VMPanicTestOption {
	return func(vmPanicResult *vmPanicResult) {
		vmPanicResult.memoryProofAddresses = append(vmPanicResult.memoryProofAddresses, addr)
	}
}

func ExpectVmPanic(goPanicValue interface{}, evmRevertMsg string, options ...VMPanicTestOption) ExpectedExecResult {
	result := vmPanicResult{
		panicValue:      goPanicValue,
		evmErrorMatcher: testutil.StringErrorMatcher(evmRevertMsg),
	}
	for _, opt := range options {
		opt(&result)
	}
	return result
}

func ExpectVmPanicWithCustomErr(goPanicMsg interface{}, customErrSignature string, options ...VMPanicTestOption) ExpectedExecResult {
	result := vmPanicResult{
		panicValue:      goPanicMsg,
		evmErrorMatcher: testutil.CustomErrorMatcher(customErrSignature),
	}
	for _, opt := range options {
		opt(&result)
	}
	return result
}

func (e vmPanicResult) assertExpectedResult(t testing.TB, goVm mipsevm.FPVM, vmVersion VersionedVMTestCase, expect *mtutil.ExpectedState, cfg *TestConfig) *mipsevm.StepWitness {
	state := goVm.GetState()
	proofData := e.proofData
	if proofData == nil {
		proofData = vmVersion.ProofGenerator(t, state, e.memoryProofAddresses...)
	}
	testutil.AssertEVMReverts(t, state, vmVersion.Contracts, cfg.tracingHooks, proofData, e.evmErrorMatcher)

	if panicErr, ok := e.panicValue.(error); ok {
		require.PanicsWithError(t, panicErr.Error(), func() { _, _ = goVm.Step(false) })
	} else if panicStr, ok := e.panicValue.(string); ok {
		require.PanicsWithValue(t, panicStr, func() { _, _ = goVm.Step(false) })
	} else {
		t.Fatalf("Invalid panic value provided.  Go panic value must be a string or error.  Got: %v", e.panicValue)
	}

	return nil
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

func (e preimageOracleRevertResult) assertExpectedResult(t testing.TB, goVm mipsevm.FPVM, vmVersion VersionedVMTestCase, expect *mtutil.ExpectedState, cfg *TestConfig) *mipsevm.StepWitness {
	require.PanicsWithValue(t, e.panicMsg, func() { _, _ = goVm.Step(true) })
	testutil.AssertPreimageOracleReverts(t, e.preimageKey, e.preimageValue, e.preimageOffset, vmVersion.Contracts)
	return nil
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
