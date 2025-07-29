package testutil

import (
	"debug/elf"
	"fmt"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/program"
)

type GoTarget string

const (
	Go1_23 GoTarget = "go-1-23"
	Go1_24 GoTarget = "go-1-24"
)

func LoadELFProgram[T mipsevm.FPVMState](t require.TestingT, name string, initState program.CreateInitialFPVMState[T]) (T, *program.Metadata) {
	elfProgram, err := elf.Open(name)
	require.NoError(t, err, "open ELF file")
	meta, err := program.MakeMetadata(elfProgram)
	require.NoError(t, err, "load metadata")

	state, err := program.LoadELF(elfProgram, initState)
	require.NoError(t, err, "load ELF into state")

	require.NoError(t, program.PatchStack(state), "add initial stack")
	return state, meta
}

// ProgramPath returns the appropriate ELF test program for the current architecture
func ProgramPath(programName string, goTarget GoTarget) string {
	return fmt.Sprintf("../../testdata/%s/bin/%s.64.elf", goTarget, programName)
}
