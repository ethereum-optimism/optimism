package program

import (
	"bytes"
	"errors"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/register"
)

const WordSizeBytes = arch.WordSizeBytes

// PatchStack sets up the program's initial stack frame and stack pointer
func PatchStack(st mipsevm.FPVMState) error {
	// setup stack pointer
	sp := Word(arch.HighMemoryStart)
	// allocate 1 page for the initial stack data, and 16KB = 4 pages for the stack to grow
	if err := st.GetMemory().SetMemoryRange(sp-4*memory.PageSize, bytes.NewReader(make([]byte, 5*memory.PageSize))); err != nil {
		return errors.New("failed to allocate page for stack content")
	}
	st.GetRegistersRef()[register.RegSP] = sp

	storeMem := func(addr Word, v Word) {
		var dat [WordSizeBytes]byte
		arch.ByteOrderWord.PutWord(dat[:], v)
		_ = st.GetMemory().SetMemoryRange(addr, bytes.NewReader(dat[:]))
	}

	auxv3Offset := sp + WordSizeBytes*10
	randomness := []byte("4;byfairdiceroll")
	randomness = pad(randomness)
	_ = st.GetMemory().SetMemoryRange(auxv3Offset, bytes.NewReader(randomness))

	envp0Offset := auxv3Offset + Word(len(randomness))
	envar := append([]byte("GODEBUG=memprofilerate=0"), 0x0)
	envar = pad(envar)
	_ = st.GetMemory().SetMemoryRange(envp0Offset, bytes.NewReader(envar))

	argv0Offset := envp0Offset + Word(len(envar))
	programName := append([]byte("op-program"), 0x0)
	programName = pad(programName)
	_ = st.GetMemory().SetMemoryRange(argv0Offset, bytes.NewReader(programName))

	// init argc, argv, aux on stack
	storeMem(sp+WordSizeBytes*0, 1)           // argc = 1 (argument count)
	storeMem(sp+WordSizeBytes*1, argv0Offset) // argv[0]
	storeMem(sp+WordSizeBytes*2, 0)           // argv[1] = terminating
	storeMem(sp+WordSizeBytes*3, envp0Offset) // envp[0] = x (offset to first env var)
	storeMem(sp+WordSizeBytes*4, 0)           // envp[1] = terminating
	storeMem(sp+WordSizeBytes*5, 6)           // auxv[0] = _AT_PAGESZ = 6 (key)
	storeMem(sp+WordSizeBytes*6, 4096)        // auxv[1] = page size of 4 KiB (value) - (== minPhysPageSize)
	storeMem(sp+WordSizeBytes*7, 25)          // auxv[2] = AT_RANDOM
	storeMem(sp+WordSizeBytes*8, auxv3Offset) // auxv[3] = address of 16 bytes containing random value
	storeMem(sp+WordSizeBytes*9, 0)           // auxv[term] = 0

	return nil
}

// pad adds appropriate padding to buf to end at Word alignment
func pad(buf []byte) []byte {
	if len(buf)%WordSizeBytes == 0 {
		return buf
	}
	bytesToAlignment := WordSizeBytes - len(buf)%WordSizeBytes
	return append(buf, make([]byte, bytesToAlignment)...)
}
