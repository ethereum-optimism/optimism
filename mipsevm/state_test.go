package mipsevm

import (
	"bytes"
	"debug/elf"
	"io"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

// baseAddrStart - baseAddrEnd is used in tests to write the results to
const baseAddrEnd = 0xbf_ff_ff_f0
const baseAddrStart = 0xbf_c0_00_00

// endAddr is used as return-address for tests
const endAddr = 0xa7ef00d0

func TestState(t *testing.T) {
	testFiles, err := os.ReadDir("test/bin")
	require.NoError(t, err)

	for _, f := range testFiles {
		t.Run(f.Name(), func(t *testing.T) {
			if f.Name() == "oracle.bin" {
				t.Skip("oracle test needs to be updated to use syscall pre-image oracle")
			}
			// TODO: currently tests are compiled as flat binary objects
			// We can use more standard tooling to compile them to ELF files and get remove maketests.py
			fn := path.Join("test/bin", f.Name())
			//elfProgram, err := elf.Open()
			//require.NoError(t, err, "must load test ELF binary")
			//state, err := LoadELF(elfProgram)
			//require.NoError(t, err, "must load ELF into state")
			programMem, err := os.ReadFile(fn)
			state := &State{PC: 0, NextPC: 4, Memory: NewMemory()}
			err = state.Memory.SetMemoryRange(0, bytes.NewReader(programMem))
			require.NoError(t, err, "load program into state")

			// set the return address ($ra) to jump into when test completes
			state.Registers[31] = endAddr

			//err = state.SetMemoryRange(baseAddr&^pageAddrMask, bytes.NewReader(make([]byte, pageSize)))
			//require.NoError(t, err, "must allocate page for the result data")
			//
			//err = state.SetMemoryRange(endAddr&^pageAddrMask, bytes.NewReader(make([]byte, pageSize)))
			//require.NoError(t, err, "must allocate page to return to")

			mu, err := NewUnicorn()
			require.NoError(t, err, "load unicorn")
			defer mu.Close()

			require.NoError(t, mu.MemMap(baseAddrStart, ((baseAddrEnd-baseAddrStart)&^pageAddrMask)+pageSize))
			require.NoError(t, mu.MemMap(endAddr&^pageAddrMask, pageSize))

			err = LoadUnicorn(state, mu)
			require.NoError(t, err, "load state into unicorn")

			us, err := NewUnicornState(mu, state, nil, os.Stdout, os.Stderr)
			require.NoError(t, err, "hook unicorn to state")

			for i := 0; i < 1000; i++ {
				if us.state.PC == endAddr {
					break
				}
				us.Step(false)
			}
			require.Equal(t, uint32(endAddr), us.state.PC, "must reach end")
			// inspect test result
			done, result := state.Memory.GetMemory(baseAddrEnd+4), state.Memory.GetMemory(baseAddrEnd+8)
			require.Equal(t, done, uint32(1), "must be done")
			require.Equal(t, result, uint32(1), "must have success result")
		})
	}
}

func TestHello(t *testing.T) {
	elfProgram, err := elf.Open("../example/bin/hello.elf")
	require.NoError(t, err, "open ELF file")

	state, err := LoadELF(elfProgram)
	require.NoError(t, err, "load ELF into state")

	err = patchVM(elfProgram, state)
	require.NoError(t, err, "apply Go runtime patches")

	mu, err := NewUnicorn()
	require.NoError(t, err, "load unicorn")
	defer mu.Close()
	err = LoadUnicorn(state, mu)
	require.NoError(t, err, "load state into unicorn")
	var stdOutBuf, stdErrBuf bytes.Buffer
	us, err := NewUnicornState(mu, state, nil, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr))
	require.NoError(t, err, "hook unicorn to state")

	for i := 0; i < 400_000; i++ {
		if us.state.Exited {
			break
		}
		us.Step(false)
	}

	require.True(t, state.Exited, "must complete program")
	require.Equal(t, uint8(0), state.ExitCode, "exit with 0")

	require.Equal(t, "hello world!", stdOutBuf.String(), "stdout says hello")
	require.Equal(t, "", stdErrBuf.String(), "stderr silent")
}
