package main

import (
	"bytes"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

// baseAddr is used in tests to write the results to
const baseAddr = 0xbffffff0

// endAddr is used as return-address for tests
const endAddr = 0xa7ef00d0

func TestState(t *testing.T) {
	testFiles, err := os.ReadDir("test/bin")
	require.NoError(t, err)

	for _, f := range testFiles {
		t.Run(f.Name(), func(t *testing.T) {
			// TODO: currently tests are compiled as flat binary objects
			// We can use more standard tooling to compile them to ELF files and get remove maketests.py
			fn := path.Join("test/bin", f.Name())
			//elfProgram, err := elf.Open()
			//require.NoError(t, err, "must load test ELF binary")
			//state, err := LoadELF(elfProgram)
			//require.NoError(t, err, "must load ELF into state")
			programMem, err := os.ReadFile(fn)
			state := &State{PC: 0, Memory: make(map[uint32]*Page)}
			err = state.SetMemoryRange(0, bytes.NewReader(programMem))
			require.NoError(t, err, "load program into state")

			// set the return address ($ra) to jump into when test completes
			state.Registers[31] = endAddr

			err = state.SetMemoryRange(baseAddr&^pageAddrMask, bytes.NewReader(make([]byte, pageSize)))
			require.NoError(t, err, "must allocate page for the result data")

			err = state.SetMemoryRange(endAddr&^pageAddrMask, bytes.NewReader(make([]byte, pageSize)))
			require.NoError(t, err, "must allocate page to return to")

			mu, err := NewUnicorn()
			require.NoError(t, err, "load unicorn")
			defer mu.Close()
			err = LoadUnicorn(state, mu)
			require.NoError(t, err, "load state into unicorn")
			err = HookUnicorn(state, mu, os.Stdout, os.Stderr)
			require.NoError(t, err, "hook unicorn to state")

			// Add hook to stop unicorn once we reached the end of the test (i.e. "ate food")
			_, err = mu.HookAdd(uc.HOOK_CODE, func(mu uc.Unicorn, addr uint64, size uint32) {
				if state.PC == endAddr {
					require.NoError(t, mu.Stop(), "stop test when returned")
				}
			}, 0, ^uint64(0))
			require.NoError(t, err, "")

			err = RunUnicorn(mu, state.PC, 1000)
			require.NoError(t, err, "must run steps without error")
			// inspect test result
			done, result := state.GetMemory(baseAddr+4), state.GetMemory(baseAddr+8)
			require.Equal(t, done, uint32(1), "must be done")
			require.Equal(t, result, uint32(1), "must have success result")
		})
	}
}
