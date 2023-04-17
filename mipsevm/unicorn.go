package main

import (
	"fmt"
	"io"
	"log"

	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

func NewUnicorn() (uc.Unicorn, error) {
	return uc.NewUnicorn(uc.ARCH_MIPS, uc.MODE_32|uc.MODE_BIG_ENDIAN)
}

func LoadUnicorn(st *State, mu uc.Unicorn) error {
	// TODO mmap each page of state into unicorn
	// TODO set all registers
	return nil
}

func HookUnicorn(st *State, mu uc.Unicorn, stdOut, stdErr io.Writer) error {
	_, err := mu.HookAdd(uc.HOOK_INTR, func(mu uc.Unicorn, intno uint32) {
		if intno != 17 {
			log.Fatal("invalid interrupt ", intno, " at step ", steps)
		}
		syscallNum, _ := mu.RegRead(uc.MIPS_REG_V0)

		v0 := uint64(0)
		switch syscallNum {
		case 4004: // write
			fd, _ := mu.RegRead(uc.MIPS_REG_A0)
			addr, _ := mu.RegRead(uc.MIPS_REG_A1)
			count, _ := mu.RegRead(uc.MIPS_REG_A2)
			switch fd {
			case 1:
				_, _ = io.Copy(stdOut, st.ReadMemory(uint32(addr), uint32(count)))
			case 2:
				_, _ = io.Copy(stdErr, st.ReadMemory(uint32(addr), uint32(count)))
			default:
				// ignore other output data
			}
		case 4090: // mmap
			a0, _ := mu.RegRead(uc.MIPS_REG_A0)
			sz, _ := mu.RegRead(uc.MIPS_REG_A1)
			if a0 == 0 {
				v0 = 0x20000000 + heap_start
				st.Heap += uint32(sz)
			} else {
				v0 = a0
			}
			// TODO mmap
		case 4045: // brk
			v0 = 0x40000000
		case 4246: // exit_group
			mu.RegWrite(uc.MIPS_REG_PC, 0x5ead0000)
		}
		mu.RegWrite(uc.MIPS_REG_V0, v0)
		mu.RegWrite(uc.MIPS_REG_A3, 0)
	}, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to set up interrupt/syscall hook: %w", err)
	}

	// Shout if Go mmap calls didn't allocate the memory properly
	_, err = mu.HookAdd(uc.HOOK_MEM_WRITE_UNMAPPED, func(mu uc.Unicorn, typ int, addr uint64, size int, value int64) bool {
		fmt.Printf("WRITE UNMAPPED typ %d  addr %016x  size %x  value  %x\n", typ, addr, size, value)
		return false
	}, 0, ^uint64(0))
	if err != nil {
		return fmt.Errorf("failed to set up unmapped-mem-write hook: %w", err)
	}

	_, err = mu.HookAdd(uc.HOOK_MEM_READ, func(mu uc.Unicorn, access int, addr64 uint64, size int, value int64) {
		//rt := value
		//rs := addr64 & 3
		//addr := uint32(addr64 & 0xFFFFFFFC)
		// TODO sanity check matches the state value
		// TODO access-list entry
	}, 0, 0x80000000)
	if err != nil {
		return fmt.Errorf("failed to set up mem-write hook: %w", err)
	}

	_, err = mu.HookAdd(uc.HOOK_MEM_WRITE, func(mu uc.Unicorn, access int, addr64 uint64, size int, value int64) {
		//rt := value
		//rs := addr64 & 3
		//addr := uint32(addr64 & 0xFFFFFFFC)
		// TODO write to state memory
	}, 0, 0x80000000)
	if err != nil {
		return fmt.Errorf("failed to set up mem-write hook: %w", err)
	}

	_, err = mu.HookAdd(uc.HOOK_CODE, func(mu uc.Unicorn, addr uint64, size uint32) {
		steps += 1

		// TODO: diff all registers
	}, 0, 0x80000000)
	if err != nil {
		return fmt.Errorf("failed to set up instruction hook: %w", err)
	}

	return nil
}
