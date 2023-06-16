package mipsevm

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"fmt"
	"io"
)

func LoadELF(f *elf.File) (*State, error) {
	s := &State{
		PC:        uint32(f.Entry),
		NextPC:    uint32(f.Entry + 4),
		HI:        0,
		LO:        0,
		Heap:      0x20000000,
		Registers: [32]uint32{},
		Memory:    NewMemory(),
		ExitCode:  0,
		Exited:    false,
		Step:      0,
	}

	for i, prog := range f.Progs {
		if prog.Type == 0x70000003 { // MIPS_ABIFLAGS
			continue
		}

		r := io.Reader(io.NewSectionReader(prog, 0, int64(prog.Filesz)))
		if prog.Filesz != prog.Memsz {
			if prog.Type == elf.PT_LOAD {
				if prog.Filesz < prog.Memsz {
					r = io.MultiReader(r, bytes.NewReader(make([]byte, prog.Memsz-prog.Filesz)))
				} else {
					return nil, fmt.Errorf("invalid PT_LOAD program segment %d, file size (%d) > mem size (%d)", i, prog.Filesz, prog.Memsz)
				}
			} else {
				return nil, fmt.Errorf("program segment %d has different file size (%d) than mem size (%d): filling for non PT_LOAD segments is not supported", i, prog.Filesz, prog.Memsz)
			}
		}

		if prog.Vaddr+prog.Memsz >= uint64(1<<32) {
			return nil, fmt.Errorf("program %d out of 32-bit mem range: %x - %x (size: %x)", i, prog.Vaddr, prog.Vaddr+prog.Memsz, prog.Memsz)
		}
		if err := s.Memory.SetMemoryRange(uint32(prog.Vaddr), r); err != nil {
			return nil, fmt.Errorf("failed to read program segment %d: %w", i, err)
		}
	}

	return s, nil
}

func PatchGo(f *elf.File, st *State) error {
	symbols, err := f.Symbols()
	if err != nil {
		return fmt.Errorf("failed to read symbols data, cannot patch program: %w", err)
	}

	for _, s := range symbols {
		// Disable Golang GC by patching the functions that enable the GC to a no-op function.
		switch s.Name {
		case "runtime.gcenable",
			"runtime.init.5",            // patch out: init() { go forcegchelper() }
			"runtime.main.func1",        // patch out: main.func() { newm(sysmon, ....) }
			"runtime.deductSweepCredit", // uses floating point nums and interacts with gc we disabled
			"runtime.(*gcControllerState).commit",
			// these prometheus packages rely on concurrent background things. We cannot run those.
			"github.com/prometheus/client_golang/prometheus.init",
			"github.com/prometheus/client_golang/prometheus.init.0",
			"github.com/prometheus/procfs.init",
			"github.com/prometheus/common/model.init",
			"github.com/prometheus/client_model/go.init",
			"github.com/prometheus/client_model/go.init.0",
			"github.com/prometheus/client_model/go.init.1",
			// skip flag pkg init, we need to debug arg-processing more to see why this fails
			"flag.init",
			// We need to patch this out, we don't pass float64nan because we don't support floats
			"runtime.check":
			// MIPS32 patch: ret (pseudo instruction)
			// 03e00008 = jr $ra = ret (pseudo instruction)
			// 00000000 = nop (executes with delay-slot, but does nothing)
			if err := st.Memory.SetMemoryRange(uint32(s.Value), bytes.NewReader([]byte{
				0x03, 0xe0, 0x00, 0x08,
				0, 0, 0, 0,
			})); err != nil {
				return fmt.Errorf("failed to patch Go runtime.gcenable: %w", err)
			}
		case "runtime.MemProfileRate":
			if err := st.Memory.SetMemoryRange(uint32(s.Value), bytes.NewReader(make([]byte, 4))); err != nil { // disable mem profiling, to avoid a lot of unnecessary floating point ops
				return err
			}
		}
	}
	return nil
}

func PatchStack(st *State) error {
	// setup stack pointer
	sp := uint32(0x7f_ff_d0_00)
	// allocate 1 page for the initial stack data, and 16KB = 4 pages for the stack to grow
	if err := st.Memory.SetMemoryRange(sp-4*PageSize, bytes.NewReader(make([]byte, 5*PageSize))); err != nil {
		return fmt.Errorf("failed to allocate page for stack content")
	}
	st.Registers[29] = sp

	storeMem := func(addr uint32, v uint32) {
		var dat [4]byte
		binary.BigEndian.PutUint32(dat[:], v)
		_ = st.Memory.SetMemoryRange(addr, bytes.NewReader(dat[:]))
	}

	// init argc, argv, aux on stack
	storeMem(sp+4*1, 0x42)   // argc = 0 (argument count)
	storeMem(sp+4*2, 0x35)   // argv[n] = 0 (terminating argv)
	storeMem(sp+4*3, 0)      // envp[term] = 0 (no env vars)
	storeMem(sp+4*4, 6)      // auxv[0] = _AT_PAGESZ = 6 (key)
	storeMem(sp+4*5, 4096)   // auxv[1] = page size of 4 KiB (value) - (== minPhysPageSize)
	storeMem(sp+4*6, 25)     // auxv[2] = AT_RANDOM
	storeMem(sp+4*7, sp+4*9) // auxv[3] = address of 16 bytes containing random value
	storeMem(sp+4*8, 0)      // auxv[term] = 0

	_ = st.Memory.SetMemoryRange(sp+4*9, bytes.NewReader([]byte("4;byfairdiceroll"))) // 16 bytes of "randomness"

	return nil
}
