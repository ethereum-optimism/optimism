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
	regValues[33] = uint64(st.LO)
	regValues[34] = uint64(st.HI)
	if err := mu.RegWriteBatch(regBatchKeys(), regValues); err != nil {
		return fmt.Errorf("failed to write registers: %w", err)
	}
	return nil
}

func HookUnicorn(st *State, mu uc.Unicorn, stdOut, stdErr io.Writer, tr Tracer) error {
	_, err := mu.HookAdd(uc.HOOK_INTR, func(mu uc.Unicorn, intno uint32) {
		if intno != 17 {
			log.Fatal("invalid interrupt ", intno, " at step ", st.Step)
		}
		syscallNum, _ := mu.RegRead(uc.MIPS_REG_V0)

		fmt.Printf("syscall: %d\n", syscallNum)
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
			if sz&pageAddrMask != 0 { // adjust size to align with page size
				sz += pageSize - (sz & pageAddrMask)
			}
			if a0 == 0 {
				v0 = uint64(st.Heap)
				fmt.Printf("mmap heap 0x%x size 0x%x\n", v0, sz)
				st.Heap += uint32(sz)
			} else {
				v0 = a0
				fmt.Printf("mmap hint 0x%x size 0x%x\n", v0, sz)
			}
			// Go does this thing where it first gets memory with PROT_NONE,
			// and then mmaps with a hint with prot=3 (PROT_READ|WRITE).
			// We can ignore the NONE case, to avoid duplicate/overlapping mmap calls to unicorn.
			prot, _ := mu.RegRead(uc.MIPS_REG_A2)
			if prot != 0 {
				if err := mu.MemMap(v0, sz); err != nil {
					log.Fatalf("mmap fail: %v", err)
				}
			}
		case 4045: // brk
			v0 = 0x40000000
		case 4246: // exit_group
			st.Exited = true
			st.ExitCode = uint8(v0)
			mu.Stop()
			return
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
		effAddr := uint32(addr64 & 0xFFFFFFFC) // pass effective addr to tracer
		tr.OnRead(effAddr, st.GetMemory(effAddr))
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
		effAddr := uint32(addr64 & 0xFFFFFFFC)
		tr.OnWrite(effAddr, st.GetMemory(effAddr))

		rt := value
		rs := addr64 & 3
		if size == 1 {
			mem := st.GetMemory(effAddr)
			val := uint32((rt & 0xFF) << (24 - (rs&3)*8))
			mask := 0xFFFFFFFF ^ uint32(0xFF<<(24-(rs&3)*8))
			st.SetMemory(effAddr, (mem&mask)|val)
		} else if size == 2 {
			mem := st.GetMemory(effAddr)
			val := uint32((rt & 0xFFFF) << (16 - (rs&2)*8))
			mask := 0xFFFFFFFF ^ uint32(0xFFFF<<(16-(rs&2)*8))
			st.SetMemory(effAddr, (mem&mask)|val)
		} else if size == 4 {
			st.SetMemory(effAddr, uint32(rt))
		} else {
			log.Fatal("bad size write to ram")
		}
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
		prevPC := st.PC
		st.PC = uint32(batch[32])

		// We detect if we are potentially in a delay-slot.
		// If we may be (i.e. last PC is 1 instruction before current),
		// then parse the last instruction to determine what the next PC would be.
		// This reflects the handleBranch / handleJump behavior that schedules next-PC.
		if st.PC == prevPC+4 {
			st.NextPC = prevPC + 8

			prevInsn := st.GetMemory(prevPC)
			opcode := prevInsn >> 26
			switch opcode {
			case 2, 3: // J/JAL
				st.NextPC = signExtend(prevInsn&0x03FFFFFF, 25) << 2
			case 1, 4, 5, 6, 7: // branching
				rs := st.Registers[(prevInsn>>21)&0x1F]
				shouldBranch := false
				switch opcode {
				case 4, 5:
					rt := st.Registers[(prevInsn>>16)&0x1F]
					shouldBranch = (rs == rt && opcode == 4) || (rs != rt && opcode == 5)
				case 6:
					shouldBranch = int32(rs) <= 0 // blez
				case 7:
					shouldBranch = int32(rs) > 0 // bgtz
				case 1:
					rtv := (prevInsn >> 16) & 0x1F
					if rtv == 0 {
						shouldBranch = int32(rs) < 0
					} // bltz
					if rtv == 1 {
						shouldBranch = int32(rs) >= 0
					} // bgez
				}
				if shouldBranch {
					st.NextPC = prevPC + 4 + (signExtend(prevInsn&0xFFFF, 15) << 2)
				}
			case 0:
				if funcv := prevInsn & 0x3f; funcv == 8 || funcv == 9 { // JR/JALR
					rs := st.Registers[(prevInsn>>21)&0x1F]
					st.NextPC = rs
				}
			}
		} else {
			st.NextPC = st.PC + 4
		}

		st.LO = uint32(batch[33])
		st.HI = uint32(batch[34])
		fmt.Printf("pc: 0x%08x\n", st.PC)
	}, 0, ^uint64(0))
	if err != nil {
		return fmt.Errorf("failed to set up instruction hook: %w", err)
	}

	return nil
}

func signExtend(v uint32, i uint32) uint32 {
	mask := ^((uint32(1) << i) - 1)
	if v&(1<<i) != 0 {
		return v | mask
	} else {
		return v &^ mask
	}
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
