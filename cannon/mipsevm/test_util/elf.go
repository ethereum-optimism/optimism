package test_util

import (
	"debug/elf"
	"testing"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/core"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/patch"
	"github.com/stretchr/testify/require"
)

func LoadELFProgram[T core.FPVMState](t *testing.T, name string, initState patch.CreateFPVMState[T]) T {
	elfProgram, err := elf.Open(name)
	require.NoError(t, err, "open ELF file")

	state, err := patch.LoadELF(elfProgram, initState)
	require.NoError(t, err, "load ELF into state")

	err = patch.PatchGo(elfProgram, state)
	require.NoError(t, err, "apply Go runtime patches")
	require.NoError(t, patch.PatchStack(state), "add initial stack")
	return state
}
