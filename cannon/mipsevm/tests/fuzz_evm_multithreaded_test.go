package tests

import (
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

// TODO
func FuzzStateSyscallCloneMT(f *testing.F) {
	v := GetMultiThreadedTestCase(f)
	// t.Skip is causing linting check to fail, disable for now
	//nolint:staticcheck
	f.Fuzz(func(t *testing.T, pc uint32, step uint64, preimageOffset uint32) {
		// TODO(cp-903) Customize test for multi-threaded vm
		t.Skip("TODO - customize this test for MTCannon")
		pc = pc & 0xFF_FF_FF_FC // align PC
		nextPC := pc + 4
		goVm := v.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(),
			WithPC(pc), WithNextPC(nextPC), WithStep(step), WithPreimageOffset(preimageOffset))
		state := goVm.GetState()
		state.GetRegistersRef()[2] = exec.SysClone

		state.GetMemory().SetMemory(pc, syscallInsn)
		preStateRoot := state.GetMemory().MerkleRoot()
		expectedRegisters := testutil.CopyRegisters(state)
		expectedRegisters[2] = 0x1

		stepWitness, err := goVm.Step(true)
		require.NoError(t, err)
		require.False(t, stepWitness.HasPreimage())

		require.Equal(t, pc+4, state.GetCpu().PC)
		require.Equal(t, nextPC+4, state.GetCpu().NextPC)
		require.Equal(t, uint32(0), state.GetCpu().LO)
		require.Equal(t, uint32(0), state.GetCpu().HI)
		require.Equal(t, uint32(0), state.GetHeap())
		require.Equal(t, uint8(0), state.GetExitCode())
		require.Equal(t, false, state.GetExited())
		require.Equal(t, preStateRoot, state.GetMemory().MerkleRoot())
		require.Equal(t, expectedRegisters, state.GetRegistersRef())
		require.Equal(t, step+1, state.GetStep())
		require.Equal(t, common.Hash{}, state.GetPreimageKey())
		require.Equal(t, preimageOffset, state.GetPreimageOffset())

		evm := testutil.NewMIPSEVM(v.Contracts)
		evmPost := evm.Step(t, stepWitness, step, v.StateHashFn)
		goPost, _ := goVm.GetState().EncodeWitness()
		require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}
