package tests

import (
	"fmt"
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

type TestCase interface {
	Name() string
}

type InitializeStateFn[T TestCase] func(testCase T, state *multithreaded.State, vm VersionedVMTestCase)
type SetExpectationsFn[T TestCase] func(testCase T, expect *mtutil.ExpectedState)

type DiffTester[T TestCase] struct {
	stateOpts       []mtutil.StateOption
	initState       InitializeStateFn[T]
	setExpectations SetExpectationsFn[T]
}

func NewDiffTester[T TestCase]() *DiffTester[T] {
	return &DiffTester[T]{}
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

func (d *DiffTester[T]) Run(t *testing.T, testCases []T, randSeed int64, opts ...TestOption) {
	if !d.isConfigValid(t) {
		t.Fatalf("DiffTester is misconfigured")
	}

	for _, vm := range GetMipsVersionTestCases(t) {
		for i, testCase := range testCases {
			cfg := newTestConfig(randSeed*int64(i), opts...)
			mods := d.generateTestModifiers(t, testCase, vm, d.setExpectations, cfg)
			for _, mod := range mods {
				testName := fmt.Sprintf("%v%v (%v)", testCase.Name(), mod.name, vm.Name)
				t.Run(testName, func(t *testing.T) {
					stateOpts := []mtutil.StateOption{mtutil.WithRandomization(cfg.randSeed)}
					stateOpts = append(stateOpts, d.stateOpts...)
					goVm := vm.VMFactory(cfg.po(), cfg.stdOut(), cfg.stdErr(), cfg.logger, stateOpts...)
					state := mtutil.GetMtState(t, goVm)

					// Set up state
					d.initState(testCase, state, vm)
					mod.stateMod(state)

					// Set up expectations
					expect := mtutil.NewExpectedState(t, state)
					d.setExpectations(testCase, expect)
					mod.expectMod(expect)

					// Step the VM
					step := state.GetStep()
					stepWitness, err := goVm.Step(true)
					require.NoError(t, err)

					// Validate
					expect.Validate(t, state)
					testutil.ValidateEVM(t, stepWitness, step, goVm, vm.StateHashFn, vm.Contracts)
				})
			}
		}
	}
}

func (d *DiffTester[T]) isConfigValid(t *testing.T) bool {
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

func (d *DiffTester[T]) generateTestModifiers(t require.TestingT, testCase T, vm VersionedVMTestCase, setExpectations SetExpectationsFn[T], cfg *TestConfig) []*testModifier {
	modifiers := []*testModifier{
		newTestModifier(""), // Always return a noop
	}

	// Process expectations
	goVm := vm.VMFactory(nil, nil, nil, nil)
	state := mtutil.GetMtState(t, goVm)
	expect := mtutil.NewExpectedState(t, state)
	setExpectations(testCase, expect)

	// Generate test modifiers based on expectations
	modifiers = append(modifiers, d.memReservationTestModifier(cfg, expect)...)

	return modifiers
}

// memReservationTestModifier updates tests that write to memory, to ensure any overlapping memory reservation
// is cleared
func (d *DiffTester[T]) memReservationTestModifier(cfg *TestConfig, expect *mtutil.ExpectedState) []*testModifier {
	var modifiers []*testModifier

	memTargets := expect.ExpectedMemoryWrites()
	if cfg.skipAutomaticMemoryReservationTests || len(memTargets) == 0 {
		// If we are explicitly skipping these mods, or memory is not written to at all, there is nothing to do
		return modifiers
	}

	modifiers = append(modifiers, &testModifier{
		name: " [mod:overlappingMemReservation]",
		stateMod: func(state *multithreaded.State) {
			// Set up a memory reservation that overlaps with the effective address of the target memory word
			r := testutil.NewRandHelper(cfg.randSeed + 10000)
			targetMemAddr := memTargets[r.Intn(len(memTargets))]
			effAddr := targetMemAddr & arch.AddressMask

			state.LLReservationStatus = multithreaded.LLReservationStatus(r.Intn(2) + 1)
			state.LLAddress = effAddr + arch.Word(r.Intn(arch.WordSizeBytes))
			state.LLOwnerThread = arch.Word(r.Intn(10))
		},
		expectMod: func(expect *mtutil.ExpectedState) {
			expect.ExpectMemoryReservationCleared()
		},
	})

	return modifiers
}

type TestConfig struct {
	randSeed int64
	po       func() mipsevm.PreimageOracle
	stdOut   func() io.Writer
	stdErr   func() io.Writer
	logger   log.Logger
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

func newTestConfig(randSeed int64, opts ...TestOption) *TestConfig {
	testConfig := &TestConfig{
		randSeed: randSeed,
		po:       func() mipsevm.PreimageOracle { return nil },
		stdOut:   func() io.Writer { return os.Stdout },
		stdErr:   func() io.Writer { return os.Stderr },
		logger:   testutil.CreateLogger(),
	}

	for _, opt := range opts {
		opt(testConfig)
	}
	return testConfig
}
