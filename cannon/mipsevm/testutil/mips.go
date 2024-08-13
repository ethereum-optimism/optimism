package testutil

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
)

type MIPSEVM struct {
	env         *vm.EVM
	evmState    *state.StateDB
	addrs       *Addresses
	localOracle mipsevm.PreimageOracle
	artifacts   *Artifacts
	// Track step execution for logging purposes
	lastStep      uint64
	lastStepInput []byte
}

func NewMIPSEVM(artifacts *Artifacts, addrs *Addresses) *MIPSEVM {
	env, evmState := NewEVMEnv(artifacts, addrs)
	return &MIPSEVM{env, evmState, addrs, nil, artifacts, math.MaxUint64, nil}
}

func (m *MIPSEVM) SetTracer(tracer *tracing.Hooks) {
	m.env.Config.Tracer = tracer
}

func (m *MIPSEVM) SetLocalOracle(oracle mipsevm.PreimageOracle) {
	m.localOracle = oracle
}

// Step is a pure function that computes the poststate from the VM state encoded in the StepWitness.
func (m *MIPSEVM) Step(t *testing.T, stepWitness *mipsevm.StepWitness, step uint64, stateHashFn mipsevm.HashFn) []byte {
	m.lastStep = step
	m.lastStepInput = nil
	sender := common.Address{0x13, 0x37}
	startingGas := uint64(30_000_000)

	// we take a snapshot so we can clean up the state, and isolate the logs of this instruction run.
	snap := m.env.StateDB.Snapshot()

	if stepWitness.HasPreimage() {
		t.Logf("reading preimage key %x at offset %d", stepWitness.PreimageKey, stepWitness.PreimageOffset)
		poInput, err := EncodePreimageOracleInput(t, stepWitness, mipsevm.LocalContext{}, m.localOracle, m.artifacts.Oracle)
		require.NoError(t, err, "encode preimage oracle input")
		_, leftOverGas, err := m.env.Call(vm.AccountRef(sender), m.addrs.Oracle, poInput, startingGas, common.U2560)
		require.NoErrorf(t, err, "evm should not fail, took %d gas", startingGas-leftOverGas)
	}

	input := EncodeStepInput(t, stepWitness, mipsevm.LocalContext{}, m.artifacts.MIPS)
	m.lastStepInput = input
	ret, leftOverGas, err := m.env.Call(vm.AccountRef(sender), m.addrs.MIPS, input, startingGas, common.U2560)
	require.NoError(t, err, "evm should not fail")
	require.Len(t, ret, 32, "expecting 32-byte state hash")
	// remember state hash, to check it against state
	postHash := common.Hash(*(*[32]byte)(ret))
	logs := m.evmState.Logs()
	require.Equal(t, 1, len(logs), "expecting a log with post-state")
	evmPost := logs[0].Data

	stateHash, err := stateHashFn(evmPost)
	require.NoError(t, err, "state hash could not be computed")
	require.Equal(t, stateHash, postHash, "logged state must be accurate")

	m.env.StateDB.RevertToSnapshot(snap)
	t.Logf("EVM step %d took %d gas, and returned stateHash %s", step, startingGas-leftOverGas, postHash)
	return evmPost
}

func EncodeStepInput(t *testing.T, wit *mipsevm.StepWitness, localContext mipsevm.LocalContext, mips *foundry.Artifact) []byte {
	input, err := mips.ABI.Pack("step", wit.State, wit.ProofData, localContext)
	require.NoError(t, err)
	return input
}

func EncodePreimageOracleInput(t *testing.T, wit *mipsevm.StepWitness, localContext mipsevm.LocalContext, localOracle mipsevm.PreimageOracle, oracle *foundry.Artifact) ([]byte, error) {
	if wit.PreimageKey == ([32]byte{}) {
		return nil, errors.New("cannot encode pre-image oracle input, witness has no pre-image to proof")
	}

	switch preimage.KeyType(wit.PreimageKey[0]) {
	case preimage.LocalKeyType:
		if len(wit.PreimageValue) > 32+8 {
			return nil, fmt.Errorf("local pre-image exceeds maximum size of 32 bytes with key 0x%x", wit.PreimageKey)
		}
		preimagePart := wit.PreimageValue[8:]
		var tmp [32]byte
		copy(tmp[:], preimagePart)
		input, err := oracle.ABI.Pack("loadLocalData",
			new(big.Int).SetBytes(wit.PreimageKey[1:]),
			localContext,
			tmp,
			new(big.Int).SetUint64(uint64(len(preimagePart))),
			new(big.Int).SetUint64(uint64(wit.PreimageOffset)),
		)
		require.NoError(t, err)
		return input, nil
	case preimage.Keccak256KeyType:
		input, err := oracle.ABI.Pack(
			"loadKeccak256PreimagePart",
			new(big.Int).SetUint64(uint64(wit.PreimageOffset)),
			wit.PreimageValue[8:])
		require.NoError(t, err)
		return input, nil
	case preimage.PrecompileKeyType:
		if localOracle == nil {
			return nil, fmt.Errorf("local oracle is required for precompile preimages")
		}
		preimage := localOracle.GetPreimage(preimage.Keccak256Key(wit.PreimageKey).PreimageKey())
		precompile := common.BytesToAddress(preimage[:20])
		callInput := preimage[20:]
		input, err := oracle.ABI.Pack(
			"loadPrecompilePreimagePart",
			new(big.Int).SetUint64(uint64(wit.PreimageOffset)),
			precompile,
			callInput,
		)
		require.NoError(t, err)
		return input, nil
	default:
		return nil, fmt.Errorf("unsupported pre-image type %d, cannot prepare preimage with key %x offset %d for oracle",
			wit.PreimageKey[0], wit.PreimageKey, wit.PreimageOffset)
	}
}

func LogStepFailureAtCleanup(t *testing.T, mipsEvm *MIPSEVM) {
	t.Cleanup(func() {
		if t.Failed() {
			// Note: For easier debugging of a failing step, see MIPS.t.sol#test_step_debug_succeeds()
			t.Logf("Failed while executing step %d with input: %x", mipsEvm.lastStep, mipsEvm.lastStepInput)
		}
	})
}
