package program

import (
	"bytes"
	"debug/elf"
	"fmt"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
)

// TODO(cp-903) Consider breaking up go patching into performance and threading-related patches so we can
// selectively apply the perf patching to MTCannon
func PatchGo(f *elf.File, st mipsevm.FPVMState) error {
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
			// MIPS64 patch: ret (pseudo instruction)
			// 03e00008 = jr $ra = ret (pseudo instruction)
			// 00000000 = nop (executes with delay-slot, but does nothing)
			if err := st.GetMemory().SetMemoryRange(uint64(s.Value), bytes.NewReader([]byte{
				0x03, 0xe0, 0x00, 0x08,
				0, 0, 0, 0,
			})); err != nil {
				return fmt.Errorf("failed to patch Go runtime.gcenable: %w", err)
			}
		}
	}
	return nil
}

func PatchStack(st mipsevm.FPVMState) error {
	// setup stack pointer
	sp := uint64(0x7F_FF_FF_FF_D0_00_00_00)
	// allocate 1 page for the initial stack data, and 16KB = 4 pages for the stack to grow
	if err := st.GetMemory().SetMemoryRange(sp-4*memory.PageSize, bytes.NewReader(make([]byte, 5*memory.PageSize))); err != nil {
		return fmt.Errorf("failed to allocate page for stack content")
	}
	st.GetRegistersRef()[29] = sp

	storeMem := func(addr uint64, v uint64) {
		st.GetMemory().SetDoubleWord(addr, v)
	}

	// init argc, argv, aux on stack
	storeMem(sp+8*0, 1)       // argc = 1 (argument count)
	storeMem(sp+8*1, sp+8*16) // argv[0]
	storeMem(sp+8*2, 0)       // argv[1] = terminating
	storeMem(sp+8*3, sp+8*12) // envp[0] = x (offset to first env var)
	storeMem(sp+8*4, 0)       // envp[1] = terminating
	storeMem(sp+8*5, 6)       // auxv[0] = _AT_PAGESZ = 6 (key)
	storeMem(sp+8*6, 4096)    // auxv[1] = page size of 4 KiB (value) - (== minPhysPageSize)
	storeMem(sp+8*7, 25)      // auxv[2] = AT_RANDOM
	storeMem(sp+8*8, sp+8*10) // auxv[3] = address of 16 bytes containing random value
	storeMem(sp+8*9, 0)       // auxv[term] = 0

	_ = st.GetMemory().SetMemoryRange(sp+8*10, bytes.NewReader([]byte("4;byfairdiceroll"))) // 16 bytes of "randomness"

	// append 8 extra zero bytes (including null term) to end at 8-byte alignment
	envar := append([]byte("GODEBUG=memprofilerate=0"), 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0)
	_ = st.GetMemory().SetMemoryRange(sp+8*12, bytes.NewReader(envar))

	// 24 bytes for GODEBUG=memprofilerate=0 + 8 null bytes
	// Then append program name + 4 null bytes (including null term) for 8-byte alignment
	programName := append([]byte("op-program"), 0x0, 0x0, 0x0, 0x0, 0x0, 0x0)
	_ = st.GetMemory().SetMemoryRange(sp+8*16, bytes.NewReader(programName))

	return nil
}
