package program

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
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
			// MIPS32 patch: ret (pseudo instruction)
			// 03e00008 = jr $ra = ret (pseudo instruction)
			// 00000000 = nop (executes with delay-slot, but does nothing)
			if err := st.GetMemory().SetMemoryRange(uint32(s.Value), bytes.NewReader([]byte{
				0x03, 0xe0, 0x00, 0x08,
				0, 0, 0, 0,
			})); err != nil {
				return fmt.Errorf("failed to patch Go runtime.gcenable: %w", err)
			}
		case "runtime.MemProfileRate":
			if err := st.GetMemory().SetMemoryRange(uint32(s.Value), bytes.NewReader(make([]byte, 4))); err != nil { // disable mem profiling, to avoid a lot of unnecessary floating point ops
				return err
			}
		}
	}
	return nil
}

// TODO(cp-903) Consider setting envar "GODEBUG=memprofilerate=0" for go programs to disable memprofiling, instead of patching it out in PatchGo()
func PatchStack(st mipsevm.FPVMState) error {
	// setup stack pointer
	sp := uint32(0x7f_ff_d0_00)
	// allocate 1 page for the initial stack data, and 16KB = 4 pages for the stack to grow
	if err := st.GetMemory().SetMemoryRange(sp-4*memory.PageSize, bytes.NewReader(make([]byte, 5*memory.PageSize))); err != nil {
		return fmt.Errorf("failed to allocate page for stack content")
	}
	st.GetRegisters()[29] = sp

	storeMem := func(addr uint32, v uint32) {
		var dat [4]byte
		binary.BigEndian.PutUint32(dat[:], v)
		_ = st.GetMemory().SetMemoryRange(addr, bytes.NewReader(dat[:]))
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

	_ = st.GetMemory().SetMemoryRange(sp+4*9, bytes.NewReader([]byte("4;byfairdiceroll"))) // 16 bytes of "randomness"

	return nil
}
