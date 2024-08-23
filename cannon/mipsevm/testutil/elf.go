package testutil

import (
	"debug/elf"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/program"
)

func LoadELFProgram[T mipsevm.FPVMState](t require.TestingT, name string, initState program.CreateInitialFPVMState[T], doPatchGo bool) T {
	state, _ := loadELFProgram(t, name, initState, doPatchGo, false)
	return state
}

func LoadELFProgramWithMetadata[T mipsevm.FPVMState](t require.TestingT, name string, initState program.CreateInitialFPVMState[T], doPatchGo bool) (T, *program.Metadata) {
	return loadELFProgram(t, name, initState, doPatchGo, true)
}

func loadELFProgram[T mipsevm.FPVMState](t require.TestingT, name string, initState program.CreateInitialFPVMState[T], doPatchGo bool, loadMetadata bool) (T, *program.Metadata) {
	elfProgram, err := elf.Open(name)
	require.NoError(t, err, "open ELF file")
	var meta *program.Metadata
	if loadMetadata {
		meta, err = program.MakeMetadata(elfProgram)
		require.NoError(t, err, "load metadata")
	}

	state, err := program.LoadELF(elfProgram, initState)
	require.NoError(t, err, "load ELF into state")

	if doPatchGo {
		err = program.PatchGo(elfProgram, state)
		require.NoError(t, err, "apply Go runtime patches")
	}

	require.NoError(t, program.PatchStack(state), "add initial stack")
	return state, meta
}
