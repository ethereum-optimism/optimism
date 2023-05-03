package unicorntest

import (
	"bytes"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/cannon/mipsevm"
)

// baseAddrStart - baseAddrEnd is used in tests to write the results to
const baseAddrEnd = 0xbf_ff_ff_f0
const baseAddrStart = 0xbf_c0_00_00

// endAddr is used as return-address for tests
const endAddr = 0xa7ef00d0

func TestState(t *testing.T) {
	testFiles, err := os.ReadDir("../../mipsevm/open_mips_tests/test/bin")
	require.NoError(t, err)

	for _, f := range testFiles {
		t.Run(f.Name(), func(t *testing.T) {
			if f.Name() == "oracle.bin" {
				t.Skip("oracle test needs to be updated to use syscall pre-image oracle")
			}
			fn := path.Join("../../mipsevm/open_mips_tests/test/bin", f.Name())

			programMem, err := os.ReadFile(fn)
			require.NoError(t, err)
			state := &mipsevm.State{PC: 0, NextPC: 4, Memory: mipsevm.NewMemory()}
			err = state.Memory.SetMemoryRange(0, bytes.NewReader(programMem))
			require.NoError(t, err, "load program into state")

			// set the return address ($ra) to jump into when test completes
			state.Registers[31] = endAddr

			mu, err := NewUnicorn()
			require.NoError(t, err, "load unicorn")
			defer mu.Close()

			require.NoError(t, mu.MemMap(baseAddrStart, ((baseAddrEnd-baseAddrStart)&^mipsevm.PageAddrMask)+mipsevm.PageSize))
			require.NoError(t, mu.MemMap(endAddr&^mipsevm.PageAddrMask, mipsevm.PageSize))

			err = LoadUnicorn(state, mu)
			require.NoError(t, err, "load state into unicorn")

			us, err := NewUnicornState(mu, state, nil, os.Stdout, os.Stderr)
			require.NoError(t, err, "hook unicorn to state")

			for i := 0; i < 1000; i++ {
				if us.state.PC == endAddr {
					break
				}
				_, err := us.Step(false)
				require.NoError(t, err)
			}
			require.Equal(t, uint32(endAddr), us.state.PC, "must reach end")
			// inspect test result
			done, result := state.Memory.GetMemory(baseAddrEnd+4), state.Memory.GetMemory(baseAddrEnd+8)
			require.Equal(t, done, uint32(1), "must be done")
			require.Equal(t, result, uint32(1), "must have success result")
		})
	}
}
