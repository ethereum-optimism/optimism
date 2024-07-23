package testutil

import (
	"debug/elf"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/program"
)

func LoadELFProgram[T mipsevm.FPVMState](t *testing.T, name string, initState program.CreateInitialFPVMState[T], doPatchGo bool) T {
	elfProgram, err := elf.Open(name)
	require.NoError(t, err, "open ELF file")

	state, err := program.LoadELF(elfProgram, initState)
	require.NoError(t, err, "load ELF into state")

	if doPatchGo {
		err = program.PatchGo(elfProgram, state)
		require.NoError(t, err, "apply Go runtime patches")
	}

	require.NoError(t, program.PatchStack(state), "add initial stack")
	return state
}
