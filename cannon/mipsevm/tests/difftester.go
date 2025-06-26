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

type DiffTester[T TestCase] struct {
	testCases []T
	stateOpts []mtutil.StateOption
}

func NewDiffTester[T TestCase](testCases []T, opts ...mtutil.StateOption) *DiffTester[T] {
	return &DiffTester[T]{
		testCases: testCases,
		stateOpts: opts,
	}
}

type TestCase interface {
	Name() string
}
type InitializeStateFn[T TestCase] func(testCase T, state *StateInitializer, vm VersionedVMTestCase)
type SetExpectationsFn[T TestCase] func(testCase T, expected *StateExpectations)

func (d *DiffTester[T]) Run(t *testing.T, stateInit InitializeStateFn[T], expectations SetExpectationsFn[T], opts ...TestOption) {
	cfg := newTestConfig(opts...)
	for _, vm := range GetMipsVersionTestCases(t) {
		for i, testCase := range d.testCases {
			testName := fmt.Sprintf("%v (%v)", testCase.Name(), vm.Name)
			t.Run(testName, func(t *testing.T) {

				// TODO - Figure out better random seed method
				stateOpts := []mtutil.StateOption{mtutil.WithRandomization(int64(i))}
				stateOpts = append(stateOpts, d.stateOpts...)
				goVm := vm.VMFactory(cfg.po(), cfg.stdOut(), cfg.stdErr(), cfg.logger, stateOpts...)
				state := mtutil.GetMtState(t, goVm)
				step := state.GetStep()

				// Set up state
				stateInitializer := &StateInitializer{state: state}
				stateInit(testCase, stateInitializer, vm)
				// Set up expectations
				stateExpectations := &StateExpectations{e: mtutil.NewExpectedState(t, state)}
				expectations(testCase, stateExpectations)

				// Apply standard expectations
				// TODO - these automatic expectations should be on be default, but with the ability to disable
				// Or maybe run with and without modifications
				d.applyStandardTestModifications(t, stateInitializer, stateExpectations)

				// Step the VM
				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)

				// Validate
				stateExpectations.Validate(t, state)
				testutil.ValidateEVM(t, stepWitness, step, goVm, vm.StateHashFn, vm.Contracts)
			})
		}
	}
}

func (d *DiffTester[T]) applyStandardTestModifications(t *testing.T, init *StateInitializer, expect *StateExpectations) {
	d.testMemoryReservationClearing(t, init, expect)
}

func (d *DiffTester[T]) testMemoryReservationClearing(t *testing.T, init *StateInitializer, expect *StateExpectations) {
	// If we are testing a memory write, force create a conflicting memory reservation so we can be sure that
	// all writes to memory will clear any existing reservation
	d.maybeInitMemoryReservation(t, init, expect)

	for _, memTarget := range expect.memoryWrites {
		effAddr := memTarget & arch.AddressMask
		reservedAddr := init.state.LLAddress & arch.AddressMask
		if effAddr == reservedAddr {
			t.Logf("Automatically set expectation that memory reservation at 0x%x will be cleared", init.state.LLAddress)
			expect.ExpectMemoryReservationCleared()
			return
		}
	}
}

func (d *DiffTester[T]) maybeInitMemoryReservation(t *testing.T, init *StateInitializer, expect *StateExpectations) {
	memTargets := expect.memoryWrites
	if len(memTargets) == 0 || init.memReservationSet {
		// If a memory reservation was explicitly set, or we don't expect any memory writes,
		// there is no need to initialize a memory reservation
		return
	}

	// Set up a memory reservation that overlaps with the effective address of the target memory word
	r := testutil.NewRandHelperFromState(init.state)
	targetMemAddr := memTargets[r.Intn(len(memTargets))]
	effAddr := targetMemAddr & arch.AddressMask

	t.Logf("Automatically set up memory reservation on initial state targeting memory address 0x%x", targetMemAddr)

	init.state.LLReservationStatus = multithreaded.LLReservationStatus(r.Intn(2) + 1)
	init.state.LLAddress = effAddr + arch.Word(r.Intn(arch.WordSizeBytes))
	init.state.LLOwnerThread = arch.Word(r.Intn(10))
}

type StateExpectations struct {
	e *mtutil.ExpectedState
	// Remember some special expectations
	memoryWrites []arch.Word
}

func (s *StateExpectations) Validate(t require.TestingT, state mipsevm.FPVMState) {
	s.e.Validate(t, state)
}

func (s *StateExpectations) Step() {
	s.e.ExpectStep()
}

func (s *StateExpectations) MemoryWrite(addr arch.Word, val arch.Word) {
	// TODO: Update regular ExpectMemoryWrite method
	s.e.ExpectMemoryWrite(addr, val)
	s.memoryWrites = append(s.memoryWrites, addr)
}

func (s *StateExpectations) ActiveRegister(reg int, val arch.Word) {
	s.e.ActiveThread().Registers[reg] = val
}

func (s *StateExpectations) ExpectMemoryReservationCleared() {
	s.e.ExpectMemoryReservationCleared()
}

type StateInitializer struct {
	state *multithreaded.State
	// Remember some special initializations
	memReservationSet bool
}

func (s *StateInitializer) SetRegister(register int, value arch.Word) {
	s.state.GetRegistersRef()[register] = value
}

func (s *StateInitializer) StoreInstruction(pc arch.Word, insn uint32) {
	testutil.StoreInstruction(s.state.GetMemory(), pc, insn)
}

func (s *StateInitializer) SetMemory(addr arch.Word, value arch.Word) {
	s.state.GetMemory().SetWord(addr, value)
}

func (s *StateInitializer) SetMemoryReservation(status multithreaded.LLReservationStatus, addr arch.Word, owner arch.Word) {
	s.memReservationSet = true
	s.state.LLReservationStatus = status
	s.state.LLAddress = addr
	s.state.LLOwnerThread = owner
}

type TestConfig struct {
	po     func() mipsevm.PreimageOracle
	stdOut func() io.Writer
	stdErr func() io.Writer
	logger log.Logger
}

type TestOption func(*TestConfig)

func WithPreimageOracle(po func() mipsevm.PreimageOracle) TestOption {
	return func(tc *TestConfig) {
		tc.po = po
	}
}

func newTestConfig(opts ...TestOption) *TestConfig {
	testConfig := &TestConfig{
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
