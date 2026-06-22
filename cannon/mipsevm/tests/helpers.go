package tests

import (
	"io"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	mtutil "github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded/testutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

type VMFactory func(po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger, opts ...mtutil.StateOption) mipsevm.FPVM

func multiThreadedVmFactory(po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger, features mipsevm.FeatureToggles, opts ...mtutil.StateOption) mipsevm.FPVM {
	state := multithreaded.CreateEmptyState()
	mutator := mtutil.NewStateMutator(state)
	for _, opt := range opts {
		opt(mutator)
	}
	return multithreaded.NewInstrumentedState(state, po, stdOut, stdErr, log, nil, features)
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
	ProofGenerator ProofGenerator
	Version        versions.StateVersion
}

func GetMultiThreadedTestCase(t require.TestingT, version versions.StateVersion) VersionedVMTestCase {
	features := versions.FeaturesForVersion(version)
	return VersionedVMTestCase{
		Name:        version.String(),
		Contracts:   testutil.TestContractsSetup(t, testutil.MipsMultithreaded, uint8(version)),
		StateHashFn: multithreaded.GetStateHashFn(),
		VMFactory: func(po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger, opts ...mtutil.StateOption) mipsevm.FPVM {
			return multiThreadedVmFactory(po, stdOut, stdErr, log, features, opts...)
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
