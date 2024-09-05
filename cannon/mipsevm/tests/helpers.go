package tests

import (
	"io"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	mttestutil "github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded/testutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/singlethreaded"
	sttestutil "github.com/ethereum-optimism/optimism/cannon/mipsevm/singlethreaded/testutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

type VMFactory func(po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger, opts ...testutil.StateOption) mipsevm.FPVM

func singleThreadedVmFactory(po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger, opts ...testutil.StateOption) mipsevm.FPVM {
	state := singlethreaded.CreateEmptyState()
	mutator := sttestutil.NewStateMutatorSingleThreaded(state)
	for _, opt := range opts {
		opt(mutator)
	}
	return singlethreaded.NewInstrumentedState(state, po, stdOut, stdErr, nil)
}

func MultiThreadedVmFactory(po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger, opts ...testutil.StateOption) mipsevm.FPVM {
	state := multithreaded.CreateEmptyState()
	mutator := mttestutil.NewStateMutatorMultiThreaded(state)
	for _, opt := range opts {
		opt(mutator)
	}
	return multithreaded.NewInstrumentedState(state, po, stdOut, stdErr, log)
}

type ElfVMFactory func(t require.TestingT, elfFile string, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger) mipsevm.FPVM

func SingleThreadElfVmFactory(t require.TestingT, elfFile string, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger) mipsevm.FPVM {
	state, meta := testutil.LoadELFProgram(t, elfFile, singlethreaded.CreateInitialState, true)
	fpvm := singlethreaded.NewInstrumentedState(state, po, stdOut, stdErr, meta)
	require.NoError(t, fpvm.InitDebug())
	return fpvm
}

func MultiThreadElfVmFactory(t require.TestingT, elfFile string, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger) mipsevm.FPVM {
	state, meta := testutil.LoadELFProgram(t, elfFile, multithreaded.CreateInitialState, false)
	fpvm := multithreaded.NewInstrumentedState(state, po, stdOut, stdErr, log)
	require.NoError(t, fpvm.InitDebug(meta))
	return fpvm
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
		ElfVMFactory: SingleThreadElfVmFactory,
	}
}

func GetMultiThreadedTestCase(t require.TestingT) VersionedVMTestCase {
	return VersionedVMTestCase{
		Name:         "multi-threaded",
		Contracts:    testutil.TestContractsSetup(t, testutil.MipsMultithreaded),
		StateHashFn:  multithreaded.GetStateHashFn(),
		VMFactory:    MultiThreadedVmFactory,
		ElfVMFactory: MultiThreadElfVmFactory,
	}
}

func GetMipsVersionTestCases(t require.TestingT) []VersionedVMTestCase {
	return []VersionedVMTestCase{
		GetSingleThreadedTestCase(t),
		GetMultiThreadedTestCase(t),
	}
}
