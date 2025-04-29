package tests

import (
	"io"
	"os"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	mttestutil "github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded/testutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

type VMFactory func(po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger, opts ...testutil.StateOption) mipsevm.FPVM

func multiThreadedVmFactory(po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger, features mipsevm.FeatureToggles, opts ...testutil.StateOption) mipsevm.FPVM {
	state := multithreaded.CreateEmptyState()
	mutator := mttestutil.NewStateMutatorMultiThreaded(state)
	for _, opt := range opts {
		opt(mutator)
	}
	return multithreaded.NewInstrumentedState(state, po, stdOut, stdErr, log, nil, features)
}

type ElfVMFactory func(t require.TestingT, elfFile string, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger) mipsevm.FPVM

func multiThreadElfVmFactory(t require.TestingT, elfFile string, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger, features mipsevm.FeatureToggles) mipsevm.FPVM {
	state, meta := testutil.LoadELFProgram(t, elfFile, multithreaded.CreateInitialState)
	fpvm := multithreaded.NewInstrumentedState(state, po, stdOut, stdErr, log, meta, features)
	require.NoError(t, fpvm.InitDebug())
	return fpvm
}

type ProofGenerator func(t require.TestingT, state mipsevm.FPVMState, memoryProofAddresses ...arch.Word) []byte

func multiThreadedProofGenerator(t require.TestingT, state mipsevm.FPVMState, memoryProofAddresses ...arch.Word) []byte {
	mtState, ok := state.(*multithreaded.State)
	if !ok {
		require.Fail(t, "Failed to cast FPVMState to multithreaded State type")
	}

	proofData := mtState.EncodeThreadProof()
	insnProof := mtState.GetMemory().MerkleProof(mtState.GetPC())
	proofData = append(proofData, insnProof[:]...)

	for _, addr := range memoryProofAddresses {
		memProof := mtState.GetMemory().MerkleProof(addr)
		proofData = append(proofData, memProof[:]...)
	}

	return proofData
}

type VersionedVMTestCase struct {
	Name           string
	Contracts      *testutil.ContractMetadata
	StateHashFn    mipsevm.HashFn
	VMFactory      VMFactory
	ElfVMFactory   ElfVMFactory
	ProofGenerator ProofGenerator
	Version        versions.StateVersion
}

func GetMultiThreadedTestCase(t require.TestingT, version versions.StateVersion) VersionedVMTestCase {
	features := versions.FeaturesForVersion(version)
	return VersionedVMTestCase{
		Name:        version.String(),
		Contracts:   testutil.TestContractsSetup(t, testutil.MipsMultithreaded, uint8(version)),
		StateHashFn: multithreaded.GetStateHashFn(),
		VMFactory: func(po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger, opts ...testutil.StateOption) mipsevm.FPVM {
			return multiThreadedVmFactory(po, stdOut, stdErr, log, features, opts...)
		},
		ElfVMFactory: func(t require.TestingT, elfFile string, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger) mipsevm.FPVM {
			return multiThreadElfVmFactory(t, elfFile, po, stdOut, stdErr, log, features)
		},
		ProofGenerator: multiThreadedProofGenerator,
		Version:        version,
	}
}

func GetMipsVersionTestCases(t require.TestingT) []VersionedVMTestCase {
	var cases []VersionedVMTestCase
	for _, version := range versions.StateVersionTypes {
		if !arch.IsMips32 && versions.IsSupportedMultiThreaded64(version) {
			cases = append(cases, GetMultiThreadedTestCase(t, version))
		}
	}
	return cases
}

type threadProofTestcase struct {
	Name  string
	Proof []byte
}

func GenerateEmptyThreadProofVariations(t require.TestingT) []threadProofTestcase {
	defaultThreadProof := multiThreadedProofGenerator(t, multithreaded.CreateEmptyState())
	zeroBytesThreadProof := make([]byte, multithreaded.THREAD_WITNESS_SIZE)
	copy(zeroBytesThreadProof[multithreaded.SERIALIZED_THREAD_SIZE:], defaultThreadProof[multithreaded.SERIALIZED_THREAD_SIZE:])
	nilBytesThreadProof := defaultThreadProof[multithreaded.SERIALIZED_THREAD_SIZE:]
	return []threadProofTestcase{
		{Name: "default thread proof", Proof: defaultThreadProof},
		{Name: "zeroed thread bytes proof", Proof: zeroBytesThreadProof},
		{Name: "nil thread bytes proof", Proof: nilBytesThreadProof},
	}
}

func setupWithTestCase(t require.TestingT, v VersionedVMTestCase, randomSeed int, preimageOracle mipsevm.PreimageOracle, opts ...testutil.StateOption) (mipsevm.FPVM, *multithreaded.State, *testutil.ContractMetadata) {
	allOpts := append([]testutil.StateOption{testutil.WithRandomization(int64(randomSeed))}, opts...)
	vm := v.VMFactory(preimageOracle, os.Stdout, os.Stderr, testutil.CreateLogger(), allOpts...)
	state := mttestutil.GetMtState(t, vm)
	return vm, state, v.Contracts
}
