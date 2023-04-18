package main

import (
	"fmt"
	"io"
	"log"
	"math"

	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

func NewUnicorn() (uc.Unicorn, error) {
	return uc.NewUnicorn(uc.ARCH_MIPS, uc.MODE_32|uc.MODE_BIG_ENDIAN)
}

func LoadUnicorn(st *State, mu uc.Unicorn) error {
	// mmap and write each page of memory state into unicorn
	for pageIndex, page := range st.Memory {
		addr := uint64(pageIndex) << pageAddrSize
		if err := mu.MemMap(addr, pageSize); err != nil {
			return fmt.Errorf("failed to mmap page at addr 0x%x: %w", addr, err)
		}
		if err := mu.MemWrite(addr, page[:]); err != nil {
			return fmt.Errorf("failed to write page at addr 0x%x: %w", addr, err)
		}
	}
	// write all registers into unicorn, including PC, LO, HI
	regValues := make([]uint64, 32+3)
	// TODO: do we have to sign-extend registers before writing them to unicorn, or are the trailing bits unused?
	for i, v := range st.Registers {
		regValues[i] = uint64(v)
	}
	regValues[32] = uint64(st.PC)
	regValues[33] = uint64(st.Lo)
	regValues[34] = uint64(st.Hi)
	if err := mu.RegWriteBatch(regBatchKeys(), regValues); err != nil {
		return fmt.Errorf("failed to write registers: %w", err)
	}
	return nil
}

func HookUnicorn(st *State, mu uc.Unicorn, stdOut, stdErr io.Writer) error {
	_, err := mu.HookAdd(uc.HOOK_INTR, func(mu uc.Unicorn, intno uint32) {
		if intno != 17 {
			log.Fatal("invalid interrupt ", intno, " at step ", st.Step)
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
				_, _ = io.Copy(stdOut, st.ReadMemoryRange(uint32(addr), uint32(count)))
			case 2:
				_, _ = io.Copy(stdErr, st.ReadMemoryRange(uint32(addr), uint32(count)))
			default:
				// ignore other output data
			}
		case 4090: // mmap
			a0, _ := mu.RegRead(uc.MIPS_REG_A0)
			sz, _ := mu.RegRead(uc.MIPS_REG_A1)
			if a0 == 0 {
				v0 = uint64(st.Heap)
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
	}, 0, ^uint64(0))
	if err != nil {
		return fmt.Errorf("failed to set up interrupt/syscall hook: %w", err)
	}

	// Shout if Go mmap calls didn't allocate the memory properly
	_, err = mu.HookAdd(uc.HOOK_MEM_UNMAPPED, func(mu uc.Unicorn, typ int, addr uint64, size int, value int64) bool {
		fmt.Printf("MEM UNMAPPED typ %d  addr %016x  size %x  value  %x\n", typ, addr, size, value)
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
	}, 0, ^uint64(0))
	if err != nil {
		return fmt.Errorf("failed to set up mem-write hook: %w", err)
	}

	_, err = mu.HookAdd(uc.HOOK_MEM_WRITE, func(mu uc.Unicorn, access int, addr64 uint64, size int, value int64) {
		if addr64 > math.MaxUint32 {
			panic("invalid addr")
		}
		if size < 0 || size > 4 {
			panic("invalid mem size")
		}
		st.SetMemory(uint32(addr64), uint32(size), uint32(value))
	}, 0, ^uint64(0))
	if err != nil {
		return fmt.Errorf("failed to set up mem-write hook: %w", err)
	}

	regBatch := regBatchKeys()
	_, err = mu.HookAdd(uc.HOOK_CODE, func(mu uc.Unicorn, addr uint64, size uint32) {
		st.Step += 1
		batch, err := mu.RegReadBatch(regBatch)
		if err != nil {
			panic(fmt.Errorf("failed to read register batch: %w", err))
		}
		for i := 0; i < 32; i++ {
			st.Registers[i] = uint32(batch[i])
		}
		st.PC = uint32(batch[32])
		st.Lo = uint32(batch[33])
		st.Hi = uint32(batch[34])
	}, 0, ^uint64(0))
	if err != nil {
		return fmt.Errorf("failed to set up instruction hook: %w", err)
	}

	return nil
}

func RunUnicorn(mu uc.Unicorn, entrypoint uint32, steps uint64) error {
	return mu.StartWithOptions(uint64(entrypoint), ^uint64(0), &uc.UcOptions{
		Timeout: 0, // 0 to disable, value is in ms.
		Count:   steps,
	})
}

func regBatchKeys() []int {
	var batch []int
	for i := 0; i < 32; i++ {
		batch = append(batch, uc.MIPS_REG_ZERO+i)
	}
	batch = append(batch, uc.MIPS_REG_PC, uc.MIPS_REG_LO, uc.MIPS_REG_HI)
	return batch
}
