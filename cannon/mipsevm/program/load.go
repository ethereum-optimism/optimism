package program

import (
	"bytes"
	"debug/elf"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
)

const (
	HEAP_START    = arch.HeapStart
	HEAP_END      = arch.HeapEnd
	PROGRAM_BREAK = arch.ProgramBreak
)

type Word = arch.Word

type CreateInitialFPVMState[T mipsevm.FPVMState] func(pc, heapStart Word) T

func LoadELF[T mipsevm.FPVMState](f *elf.File, initState CreateInitialFPVMState[T]) (T, error) {
	var empty T
	s := initState(Word(f.Entry), HEAP_START)

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
					return empty, fmt.Errorf("invalid PT_LOAD program segment %d, file size (%d) > mem size (%d)", i, prog.Filesz, prog.Memsz)
				}
			} else {
				return empty, fmt.Errorf("program segment %d has different file size (%d) than mem size (%d): filling for non PT_LOAD segments is not supported", i, prog.Filesz, prog.Memsz)
			}
		}

		// TODO(#12205)
		if prog.Vaddr+prog.Memsz >= uint64(1<<32) {
			return empty, fmt.Errorf("program %d out of 32-bit mem range: %x - %x (size: %x)", i, prog.Vaddr, prog.Vaddr+prog.Memsz, prog.Memsz)
		}
		if prog.Vaddr+prog.Memsz >= HEAP_START {
			return empty, fmt.Errorf("program %d overlaps with heap: %x - %x (size: %x). The heap start offset must be reconfigured", i, prog.Vaddr, prog.Vaddr+prog.Memsz, prog.Memsz)
		}
		if err := s.GetMemory().SetMemoryRange(Word(prog.Vaddr), r); err != nil {
			return empty, fmt.Errorf("failed to read program segment %d: %w", i, err)
		}
	}

	return s, nil
}
