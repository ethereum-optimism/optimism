package mipsevm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"
)

type MIPSEVM struct {
	env      *vm.EVM
	evmState *state.StateDB
	addrs    *Addresses
}

func NewMIPSEVM(contracts *Contracts, addrs *Addresses) *MIPSEVM {
	env, evmState := NewEVMEnv(contracts, addrs)
	return &MIPSEVM{env, evmState, addrs}
}

func (m *MIPSEVM) SetTracer(tracer vm.EVMLogger) {
	m.env.Config.Tracer = tracer
}

// Step is a pure function that computes the poststate from the VM state encoded in the StepWitness.
func (m *MIPSEVM) Step(t *testing.T, stepWitness *StepWitness) []byte {
	sender := common.Address{0x13, 0x37}
	startingGas := uint64(30_000_000)

	// we take a snapshot so we can clean up the state, and isolate the logs of this instruction run.
	snap := m.env.StateDB.Snapshot()

	if stepWitness.HasPreimage() {
		t.Logf("reading preimage key %x at offset %d", stepWitness.PreimageKey, stepWitness.PreimageOffset)
		poInput, err := stepWitness.EncodePreimageOracleInput()
		require.NoError(t, err, "encode preimage oracle input")
		_, leftOverGas, err := m.env.Call(vm.AccountRef(sender), m.addrs.Oracle, poInput, startingGas, big.NewInt(0))
		require.NoErrorf(t, err, "evm should not fail, took %d gas", startingGas-leftOverGas)
	}

	input := stepWitness.EncodeStepInput()
	ret, leftOverGas, err := m.env.Call(vm.AccountRef(sender), m.addrs.MIPS, input, startingGas, big.NewInt(0))
	require.NoError(t, err, "evm should not fail")
	require.Len(t, ret, 32, "expecting 32-byte state hash")
	// remember state hash, to check it against state
	postHash := common.Hash(*(*[32]byte)(ret))
	logs := m.evmState.Logs()
	require.Equal(t, 1, len(logs), "expecting a log with post-state")
	evmPost := logs[0].Data

	stateHash, err := StateWitness(evmPost).StateHash()
	require.NoError(t, err, "state hash could not be computed")
	require.Equal(t, stateHash, postHash, "logged state must be accurate")

	m.env.StateDB.RevertToSnapshot(snap)
	t.Logf("EVM step took %d gas, and returned stateHash %s", startingGas-leftOverGas, postHash)
	return evmPost
}

func TestContractsSetup(t require.TestingT) (*Contracts, *Addresses) {
	contracts, err := LoadContracts()
	require.NoError(t, err)

	addrs := &Addresses{
		MIPS:         common.Address{0: 0xff, 19: 1},
		Oracle:       common.Address{0: 0xff, 19: 2},
		Sender:       common.Address{0x13, 0x37},
		FeeRecipient: common.Address{0xaa},
	}

	return contracts, addrs
}
