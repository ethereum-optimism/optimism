package unicorntest

import (
	"testing"

	"github.com/stretchr/testify/require"

	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

// TestUnicornDelaySlot test that unicorn works, and determine exactly how delay slots behave
func TestUnicornDelaySlot(t *testing.T) {
	mu, err := NewUnicorn()
	require.NoError(t, err)
	defer mu.Close()

	require.NoError(t, mu.MemMap(0, 4096))
	require.NoError(t, mu.RegWrite(uc.MIPS_REG_RA, 420), "set RA to addr that is multiple of 4")
	require.NoError(t, mu.MemWrite(0, []byte{0x03, 0xe0, 0x00, 0x08}), "jr $ra")
	require.NoError(t, mu.MemWrite(4, []byte{0x20, 0x09, 0x0a, 0xFF}), "addi $t1 $r0 0x0aff")
	require.NoError(t, mu.MemWrite(32, []byte{0x20, 0x09, 0x0b, 0xFF}), "addi $t1 $r0 0x0bff")

	_, err = mu.HookAdd(uc.HOOK_CODE, func(mu uc.Unicorn, addr uint64, size uint32) {
		t.Logf("addr: %08x", addr)
	}, uint64(0), ^uint64(0))
	require.NoError(t, err)
	// stop at instruction in addr=4, the delay slot
	require.NoError(t, mu.StartWithOptions(uint64(0), uint64(4), &uc.UcOptions{
		Timeout: 0, // 0 to disable, value is in ms.
		Count:   2,
	}))

	t1, err := mu.RegRead(uc.MIPS_REG_T1)
	require.NoError(t, err)
	require.NotEqual(t, uint64(0x0aff), t1, "delay slot should not execute")

	pc, err := mu.RegRead(uc.MIPS_REG_PC)
	require.NoError(t, err)
	// unicorn is weird here: when entering a delay slot, it does not update the PC register by itself.
	require.Equal(t, uint64(0), pc, "delay slot, no jump yet")

	// now restart, but run two instructions, to include the delay slot
	require.NoError(t, mu.StartWithOptions(uint64(0), ^uint64(0), &uc.UcOptions{
		Timeout: 0, // 0 to disable, value is in ms.
		Count:   2,
	}))

	pc, err = mu.RegRead(uc.MIPS_REG_PC)
	require.NoError(t, err)
	require.Equal(t, uint64(420), pc, "jumped after NOP delay slot")

	t1, err = mu.RegRead(uc.MIPS_REG_T1)
	require.NoError(t, err)
	require.Equal(t, uint64(0x0aff), t1, "delay slot should execute")

	require.NoError(t, mu.StartWithOptions(uint64(32), uint64(32+4), &uc.UcOptions{
		Timeout: 0, // 0 to disable, value is in ms.
		Count:   1,
	}))
	t1, err = mu.RegRead(uc.MIPS_REG_T1)
	require.NoError(t, err)
	require.Equal(t, uint64(0x0bff), t1, "regular instruction should work fine")
}
