package mipsevm

import (
	"fmt"
	"io"
	"log"
	"math"
	"sync"

	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

type PreimageOracle interface {
	Hint(v []byte)
	GetPreimage(k [32]byte) []byte
}

type UnicornState struct {
	sync.Mutex

	mu uc.Unicorn

	state *State

	stdOut io.Writer
	stdErr io.Writer

	lastMemAccess   uint32
	memProofEnabled bool
	memProof        [28 * 32]byte

	preimageOracle PreimageOracle

	// number of bytes last read from the oracle.
	// The read data is preimage[state.PreimageOffset-lastPreimageRead : state.PreimageOffset]
	// when inspecting the post-step state.
	lastPreimageRead uint32

	// cached pre-image data for state.PreimageKey
	lastPreimage []byte

	onStep func()
}

const (
	fdStdin         = 0
	fdStdout        = 1
	fdStderr        = 2
	fdHintRead      = 3
	fdHintWrite     = 4
	fdPreimageRead  = 5
	fdPreimageWrite = 6
)

const (
	MipsEBADF  = 0x9
	MipsEINVAL = 0x16
)

func NewUnicornState(mu uc.Unicorn, state *State, po PreimageOracle, stdOut, stdErr io.Writer) (*UnicornState, error) {
	m := &UnicornState{
		mu:             mu,
		state:          state,
		stdOut:         stdOut,
		stdErr:         stdErr,
		preimageOracle: po,
	}
	st := m.state

	var err error
	_, err = mu.HookAdd(uc.HOOK_INTR, func(mu uc.Unicorn, intno uint32) {
		if intno != 17 {
			log.Fatal("invalid interrupt ", intno, " at step ", st.Step)
		}

		syscallNum := st.Registers[2] // v0
		v0 := uint32(0)
		//v1 := uint32(0)

		a0 := st.Registers[4]
		a1 := st.Registers[5]
		a2 := st.Registers[6]

		fmt.Printf("syscall: %d\n", syscallNum)
		switch syscallNum {
		case 4004: // write
			fd := a0
			addr := a1
			count := a2
			switch fd {
			case fdStdout:
				_, _ = io.Copy(stdOut, st.Memory.ReadMemoryRange(addr, count))
				v0 = count
			case fdStderr:
				_, _ = io.Copy(stdErr, st.Memory.ReadMemoryRange(addr, count))
				v0 = count
			case fdHintWrite:
				hint, _ := io.ReadAll(st.Memory.ReadMemoryRange(addr, count))
				v0 = count
				po.Hint(hint)
			case fdPreimageWrite:
				// TODO
				v0 = count
			default:
				v0 = 0xFFffFFff
				//v1 = MipsEBADF
				// ignore other output data
			}
		case 4090: // mmap
			sz := a1
			if sz&pageAddrMask != 0 { // adjust size to align with page size
				sz += pageSize - (sz & pageAddrMask)
			}
			if a0 == 0 {
				v0 = st.Heap
				fmt.Printf("mmap heap 0x%x size 0x%x\n", v0, sz)
				st.Heap += sz
			} else {
				v0 = a0
				fmt.Printf("mmap hint 0x%x size 0x%x\n", v0, sz)
			}
			// Go does this thing where it first gets memory with PROT_NONE,
			// and then mmaps with a hint with prot=3 (PROT_READ|WRITE).
			// We can ignore the NONE case, to avoid duplicate/overlapping mmap calls to unicorn.
			prot := a2
			if prot != 0 {
				if err := mu.MemMap(uint64(v0), uint64(sz)); err != nil {
					log.Fatalf("mmap fail: %v", err)
				}
			}
		case 4045: // brk
			v0 = 0x40000000
		case 4246: // exit_group
			st.Exited = true
			v0, _ := mu.RegRead(uc.MIPS_REG_4)
			st.ExitCode = uint8(v0)
			return
		}
		mu.RegWrite(uc.MIPS_REG_V0, uint64(v0))
		mu.RegWrite(uc.MIPS_REG_A3, 0)
	}, 0, ^uint64(0))
	if err != nil {
		return nil, fmt.Errorf("failed to set up interrupt/syscall hook: %w", err)
	}

	// Shout if Go mmap calls didn't allocate the memory properly
	_, err = mu.HookAdd(uc.HOOK_MEM_UNMAPPED, func(mu uc.Unicorn, typ int, addr uint64, size int, value int64) bool {
		fmt.Printf("MEM UNMAPPED typ %d  addr %016x  size %x  value  %x\n", typ, addr, size, value)
		return false
	}, 0, ^uint64(0))
	if err != nil {
		return nil, fmt.Errorf("failed to set up unmapped-mem-write hook: %w", err)
	}

	_, err = mu.HookAdd(uc.HOOK_MEM_READ, func(mu uc.Unicorn, access int, addr64 uint64, size int, value int64) {
		effAddr := uint32(addr64 & 0xFFFFFFFC) // pass effective addr to tracer
		if m.memProofEnabled && m.lastMemAccess != effAddr {
			if m.lastMemAccess != ^uint32(0) {
				panic(fmt.Errorf("unexpected different mem access at %08x, already have access at %08x buffered", effAddr, m.lastMemAccess))
			}
			m.lastMemAccess = effAddr
			m.memProof = m.state.Memory.MerkleProof(effAddr)
		}
	}, 0, ^uint64(0))
	if err != nil {
		return nil, fmt.Errorf("failed to set up mem-write hook: %w", err)
	}

	_, err = mu.HookAdd(uc.HOOK_MEM_WRITE, func(mu uc.Unicorn, access int, addr64 uint64, size int, value int64) {
		if addr64 > math.MaxUint32 {
			panic("invalid addr")
		}
		if size < 0 || size > 4 {
			panic("invalid mem size")
		}
		effAddr := uint32(addr64 & 0xFFFFFFFC)

		pre := st.Memory.GetMemory(effAddr)

		var post uint32
		rt := value
		rs := addr64 & 3
		if size == 1 {
			val := uint32((rt & 0xFF) << (24 - (rs&3)*8))
			mask := 0xFFFFFFFF ^ uint32(0xFF<<(24-(rs&3)*8))
			post = (pre & mask) | val
		} else if size == 2 {
			val := uint32((rt & 0xFFFF) << (16 - (rs&2)*8))
			mask := 0xFFFFFFFF ^ uint32(0xFFFF<<(16-(rs&2)*8))
			post = (pre & mask) | val
		} else if size == 4 {
			post = uint32(rt)
		} else {
			log.Fatal("bad size write to ram")
		}
		if m.memProofEnabled && m.lastMemAccess != effAddr {
			if m.lastMemAccess != ^uint32(0) {
				panic(fmt.Errorf("unexpected different mem access at %08x, already have access at %08x buffered", effAddr, m.lastMemAccess))
			}
			m.lastMemAccess = effAddr
			m.memProof = m.state.Memory.MerkleProof(effAddr)
		}
		// only set memory after making the proof: we need the pre-state
		st.Memory.SetMemory(effAddr, post)
	}, 0, ^uint64(0))
	if err != nil {
		return nil, fmt.Errorf("failed to set up mem-write hook: %w", err)
	}

	return m, nil
}

func (m *UnicornState) Step(proof bool) (wit *StepWitness) {
	m.memProofEnabled = proof
	m.lastMemAccess = ^uint32(0)

	if proof {
		insnProof := m.state.Memory.MerkleProof(m.state.PC)
		wit = &StepWitness{
			state:    m.state.EncodeWitness(),
			memProof: insnProof[:],
		}
	}

	insn := m.state.Memory.GetMemory(m.state.PC)
	oldNextPC := m.state.NextPC
	newNextPC := oldNextPC + 4

	opcode := insn >> 26
	switch opcode {
	case 2, 3: // J/JAL
		newNextPC = signExtend(insn&0x03FFFFFF, 25) << 2
	case 1, 4, 5, 6, 7: // branching
		rs := m.state.Registers[(insn>>21)&0x1F]
		shouldBranch := false
		switch opcode {
		case 4, 5:
			rt := m.state.Registers[(insn>>16)&0x1F]
			shouldBranch = (rs == rt && opcode == 4) || (rs != rt && opcode == 5)
		case 6:
			shouldBranch = int32(rs) <= 0 // blez
		case 7:
			shouldBranch = int32(rs) > 0 // bgtz
		case 1:
			rtv := (insn >> 16) & 0x1F
			if rtv == 0 {
				shouldBranch = int32(rs) < 0
			} // bltz
			if rtv == 1 {
				shouldBranch = int32(rs) >= 0
			} // bgez
		}
		if shouldBranch {
			newNextPC = m.state.PC + 4 + (signExtend(insn&0xFFFF, 15) << 2)
		}
	case 0:
		if funcv := insn & 0x3f; funcv == 8 || funcv == 9 { // JR/JALR
			rs := m.state.Registers[(insn>>21)&0x1F]
			newNextPC = rs
		}
	}

	// Execute only a single instruction.
	// The memory and syscall hooks will update the state with any of the dynamic changes.
	err := m.mu.StartWithOptions(uint64(m.state.PC), uint64(m.state.NextPC), &uc.UcOptions{
		Timeout: 0, // 0 to disable, value is in ms.
		Count:   1,
	})

	if proof {
		wit.memProof = append(wit.memProof, m.memProof[:]...)
		if m.lastPreimageRead > 0 {
			wit.preimageOffset = m.state.PreimageOffset
			wit.preimageKey = m.state.PreimageKey
			wit.preimageValue = m.lastPreimage
		}
	}

	// count it
	m.state.Step += 1

	// Now do post-processing to keep our state in sync:

	// 1) match the registers post-state
	batch, err := m.mu.RegReadBatch(regBatchKeys)
	if err != nil {
		panic(fmt.Errorf("failed to read register batch: %w", err))
	}
	for i := 0; i < 32; i++ {
		m.state.Registers[i] = uint32(batch[i])
	}
	_ = uint32(batch[32]) // ignore the PC, we follow oldNextPC instead, to emulate delay-slot behavior
	m.state.LO = uint32(batch[33])
	m.state.HI = uint32(batch[34])

	// 2) adopt the old nextPC as new PC. Unless we just exited.
	// This effectively implements delay-slots, even though unicorn immediately loses
	// delay-slot information when only executing a single instruction.
	if !m.state.Exited {
		m.state.PC = oldNextPC
		err = m.mu.RegWrite(uc.MIPS_REG_PC, uint64(oldNextPC))
		if err != nil {
			panic("failed to write PC register")
		}

		m.state.NextPC = newNextPC
	}
	return
}

func NewUnicorn() (uc.Unicorn, error) {
	return uc.NewUnicorn(uc.ARCH_MIPS, uc.MODE_32|uc.MODE_BIG_ENDIAN)
}

func LoadUnicorn(st *State, mu uc.Unicorn) error {
	// mmap and write each page of memory state into unicorn
	for pageIndex, page := range st.Memory.Pages {
		addr := uint64(pageIndex) << pageAddrSize
		if err := mu.MemMap(addr, pageSize); err != nil {
			return fmt.Errorf("failed to mmap page at addr 0x%x: %w", addr, err)
		}
		if err := mu.MemWrite(addr, page.Data[:]); err != nil {
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
	if err := mu.RegWriteBatch(regBatchKeys, regValues); err != nil {
		return fmt.Errorf("failed to write registers: %w", err)
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

var regBatchKeys = func() []int {
	var batch []int
	for i := 0; i < 32; i++ {
		batch = append(batch, uc.MIPS_REG_ZERO+i)
	}
	batch = append(batch, uc.MIPS_REG_PC, uc.MIPS_REG_LO, uc.MIPS_REG_HI)
	return batch
}()
