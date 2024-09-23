package program

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
)

// PatchGoGC patches out garbage-collection-related symbols to disable garbage collection
// and improves performance by patching out floating-point-related symbols
func PatchGoGC(f *elf.File, st mipsevm.FPVMState) error {
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
			if err := st.GetMemory().SetMemoryRange(uint32(s.Value), bytes.NewReader([]byte{
				0x03, 0xe0, 0x00, 0x08,
				0, 0, 0, 0,
			})); err != nil {
				return fmt.Errorf("failed to patch Go runtime.gcenable: %w", err)
			}
		}
	}
	return nil
}

// PatchStack sets up the program's initial stack frame and stack pointer
func PatchStack(st mipsevm.FPVMState) error {
	// setup stack pointer
	sp := uint32(0x7f_ff_d0_00)
	// allocate 1 page for the initial stack data, and 16KB = 4 pages for the stack to grow
	if err := st.GetMemory().SetMemoryRange(sp-4*memory.PageSize, bytes.NewReader(make([]byte, 5*memory.PageSize))); err != nil {
		return errors.New("failed to allocate page for stack content")
	}
	st.GetRegistersRef()[29] = sp

	storeMem := func(addr uint32, v uint32) {
		var dat [4]byte
		binary.BigEndian.PutUint32(dat[:], v)
		_ = st.GetMemory().SetMemoryRange(addr, bytes.NewReader(dat[:]))
	}

	// init argc, argv, aux on stack
	storeMem(sp+4*0, 1)       // argc = 1 (argument count)
	storeMem(sp+4*1, sp+4*21) // argv[0]
	storeMem(sp+4*2, 0)       // argv[1] = terminating
	storeMem(sp+4*3, sp+4*14) // envp[0] = x (offset to first env var)
	storeMem(sp+4*4, 0)       // envp[1] = terminating
	storeMem(sp+4*5, 6)       // auxv[0] = _AT_PAGESZ = 6 (key)
	storeMem(sp+4*6, 4096)    // auxv[1] = page size of 4 KiB (value) - (== minPhysPageSize)
	storeMem(sp+4*7, 25)      // auxv[2] = AT_RANDOM
	storeMem(sp+4*8, sp+4*10) // auxv[3] = address of 16 bytes containing random value
	storeMem(sp+4*9, 0)       // auxv[term] = 0

	_ = st.GetMemory().SetMemoryRange(sp+4*10, bytes.NewReader([]byte("4;byfairdiceroll"))) // 16 bytes of "randomness"

	// append 4 extra zero bytes to end at 4-byte alignment
	envar := append([]byte("GODEBUG=memprofilerate=0"), 0x0, 0x0, 0x0, 0x0)
	_ = st.GetMemory().SetMemoryRange(sp+4*14, bytes.NewReader(envar))

	// 24 bytes for GODEBUG=memprofilerate=0 + 4 null bytes
	// Then append program name + 2 null bytes for 4-byte alignment
	programName := append([]byte("op-program"), 0x0, 0x0)
	_ = st.GetMemory().SetMemoryRange(sp+4*21, bytes.NewReader(programName))

	return nil
}
