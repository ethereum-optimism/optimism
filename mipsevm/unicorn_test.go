package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

// TestUnicorn test that unicorn works
func TestUnicorn(t *testing.T) {
	mu, err := NewUnicorn()
	require.NoError(t, err)
	defer mu.Close()

	require.NoError(t, mu.MemMap(0, 4096))
	require.NoError(t, mu.RegWrite(uc.MIPS_REG_RA, 420), "set RA to addr that is multiple of 4")
	require.NoError(t, mu.MemWrite(0, []byte{0x03, 0xe0, 0x00, 0x08}), "jmp $ra")

	require.NoError(t, RunUnicorn(mu, 0, 1))
	pc, err := mu.RegRead(uc.MIPS_REG_PC)
	require.NoError(t, err)
	require.Equal(t, uint64(420), pc, "jumped")
}
