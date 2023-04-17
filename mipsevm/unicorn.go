package main

import (
	"fmt"
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

func HookUnicorn(st *State, mu uc.Unicorn) error {
	_, err := mu.HookAdd(uc.HOOK_INTR, func(mu uc.Unicorn, intno uint32) {
		if intno != 17 {
			log.Fatal("invalid interrupt ", intno, " at step ", steps)
		}
		syscallNum, _ := mu.RegRead(uc.MIPS_REG_V0)
		// TODO process syscalls
		switch syscallNum {
		// TODO mmap
		}
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
