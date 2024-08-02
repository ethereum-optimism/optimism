package tests

import (
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/singlethreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

type StateMutator interface {
	SetPC(pc uint32)
	SetNextPC(nextPC uint32)
	SetHeap(addr uint32)
	SetLastHint(lastHint hexutil.Bytes)
	SetPreimageKey(key common.Hash)
	SetPreimageOffset(offset uint32)
	SetStep(step uint64)
}

type singlethreadedMutator struct {
	state *singlethreaded.State
}

var _ StateMutator = (*singlethreadedMutator)(nil)

func (m *singlethreadedMutator) SetPC(pc uint32) {
	m.state.Cpu.PC = pc
}

func (m *singlethreadedMutator) SetNextPC(nextPC uint32) {
	m.state.Cpu.NextPC = nextPC
}

func (m *singlethreadedMutator) SetHeap(addr uint32) {
	m.state.Heap = addr
}

func (m *singlethreadedMutator) SetLastHint(lastHint hexutil.Bytes) {
	m.state.LastHint = lastHint
}

func (m *singlethreadedMutator) SetPreimageKey(key common.Hash) {
	m.state.PreimageKey = key
}

func (m *singlethreadedMutator) SetPreimageOffset(offset uint32) {
	m.state.PreimageOffset = offset
}

func (m *singlethreadedMutator) SetStep(step uint64) {
	m.state.Step = step
}

type multithreadedMutator struct {
	state *multithreaded.State
}

var _ StateMutator = (*multithreadedMutator)(nil)

func (m *multithreadedMutator) SetPC(pc uint32) {
	thread := m.state.GetCurrentThread()
	thread.Cpu.PC = pc
}

func (m *multithreadedMutator) SetHeap(addr uint32) {
	m.state.Heap = addr
}

func (m *multithreadedMutator) SetNextPC(nextPC uint32) {
	thread := m.state.GetCurrentThread()
	thread.Cpu.NextPC = nextPC
}

func (m *multithreadedMutator) SetLastHint(lastHint hexutil.Bytes) {
	m.state.LastHint = lastHint
}

func (m *multithreadedMutator) SetPreimageKey(key common.Hash) {
	m.state.PreimageKey = key
}

func (m *multithreadedMutator) SetPreimageOffset(offset uint32) {
	m.state.PreimageOffset = offset
}

func (m *multithreadedMutator) SetStep(step uint64) {
	m.state.Step = step
}

type VMOption func(vm StateMutator)

func WithPC(pc uint32) VMOption {
	return func(state StateMutator) {
		state.SetPC(pc)
	}
}

func WithNextPC(nextPC uint32) VMOption {
	return func(state StateMutator) {
		state.SetNextPC(nextPC)
	}
}

func WithHeap(addr uint32) VMOption {
	return func(state StateMutator) {
		state.SetHeap(addr)
	}
}

func WithLastHint(lastHint hexutil.Bytes) VMOption {
	return func(state StateMutator) {
		state.SetLastHint(lastHint)
	}
}

func WithPreimageKey(key common.Hash) VMOption {
	return func(state StateMutator) {
		state.SetPreimageKey(key)
	}
}

func WithPreimageOffset(offset uint32) VMOption {
	return func(state StateMutator) {
		state.SetPreimageOffset(offset)
	}
}

func WithStep(step uint64) VMOption {
	return func(state StateMutator) {
		state.SetStep(step)
	}
}

type VMFactory func(po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger, opts ...VMOption) mipsevm.FPVM

func singleThreadedVmFactory(po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger, opts ...VMOption) mipsevm.FPVM {
	state := singlethreaded.CreateEmptyState()
	mutator := &singlethreadedMutator{state: state}
	for _, opt := range opts {
		opt(mutator)
	}
	return singlethreaded.NewInstrumentedState(state, po, stdOut, stdErr, nil)
}

func multiThreadedVmFactory(po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger, opts ...VMOption) mipsevm.FPVM {
	state := multithreaded.CreateEmptyState()
	mutator := &multithreadedMutator{state: state}
	for _, opt := range opts {
		opt(mutator)
	}
	return multithreaded.NewInstrumentedState(state, po, stdOut, stdErr, log)
}

type ElfVMFactory func(t require.TestingT, elfFile string, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger) mipsevm.FPVM

func singleThreadElfVmFactory(t require.TestingT, elfFile string, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger) mipsevm.FPVM {
	state := testutil.LoadELFProgram(t, elfFile, singlethreaded.CreateInitialState, true)
	return singlethreaded.NewInstrumentedState(state, po, stdOut, stdErr, nil)
}

func multiThreadElfVmFactory(t require.TestingT, elfFile string, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger) mipsevm.FPVM {
	state := testutil.LoadELFProgram(t, elfFile, multithreaded.CreateInitialState, false)
	return multithreaded.NewInstrumentedState(state, po, stdOut, stdErr, log)
}

type VersionedVMTestCase struct {
	Name         string
	Contracts    *testutil.ContractMetadata
	StateHashFn  mipsevm.HashFn
	VMFactory    VMFactory
	ElfVMFactory ElfVMFactory
}

func GetSingleThreadedTestCase(t require.TestingT) VersionedVMTestCase {
	return VersionedVMTestCase{
		Name:         "single-threaded",
		Contracts:    testutil.TestContractsSetup(t, testutil.MipsSingleThreaded),
		StateHashFn:  singlethreaded.GetStateHashFn(),
		VMFactory:    singleThreadedVmFactory,
		ElfVMFactory: singleThreadElfVmFactory,
	}
}

func GetMultiThreadedTestCase(t require.TestingT) VersionedVMTestCase {
	return VersionedVMTestCase{
		Name:         "multi-threaded",
		Contracts:    testutil.TestContractsSetup(t, testutil.MipsMultithreaded),
		StateHashFn:  multithreaded.GetStateHashFn(),
		VMFactory:    multiThreadedVmFactory,
		ElfVMFactory: multiThreadElfVmFactory,
	}
}

func GetMipsVersionTestCases(t require.TestingT) []VersionedVMTestCase {
	return []VersionedVMTestCase{
		GetSingleThreadedTestCase(t),
		GetMultiThreadedTestCase(t),
	}
}
