package mipsevm

import (
	"encoding/binary"
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

	// cached pre-image data, including 8 byte length prefix
	lastPreimage []byte
	// key for above preimage
	lastPreimageKey [32]byte
	// offset we last read from, or max uint32 if nothing is read this step
	lastPreimageOffset uint32
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

func NewNonUnicornState(state *State, po PreimageOracle, stdOut, stdErr io.Writer) *UnicornState {
	return &UnicornState{
		state:          state,
		stdOut:         stdOut,
		stdErr:         stdErr,
		preimageOracle: po,
	}
}

func NewUnicornState(mu uc.Unicorn, state *State, po PreimageOracle, stdOut, stdErr io.Writer) (*UnicornState, error) {
	m := &UnicornState{
		mu:             mu,
		state:          state,
		stdOut:         stdOut,
		stdErr:         stdErr,
		preimageOracle: po,
	}
	st := m.state

	readPreimage := func(key [32]byte, offset uint32) (dat [32]byte, datLen uint32) {
		preimage := m.lastPreimage
		if key != m.lastPreimageKey {
			m.lastPreimageKey = key
			data := po.GetPreimage(key)
			// add the length prefix
			preimage = make([]byte, 0, 8+len(data))
			preimage = binary.BigEndian.AppendUint64(preimage, uint64(len(data)))
			preimage = append(preimage, data...)
			m.lastPreimage = preimage
		}
		m.lastPreimageOffset = offset
		datLen = uint32(copy(dat[:], preimage[offset:]))
		return
	}

	trackMemAccess := func(effAddr uint32) {
		if m.memProofEnabled && m.lastMemAccess != effAddr {
			if m.lastMemAccess != ^uint32(0) {
				panic(fmt.Errorf("unexpected different mem access at %08x, already have access at %08x buffered", effAddr, m.lastMemAccess))
			}
			m.lastMemAccess = effAddr
			m.memProof = m.state.Memory.MerkleProof(effAddr)
		}
	}

	var err error
	_, err = mu.HookAdd(uc.HOOK_INTR, func(mu uc.Unicorn, intno uint32) {
		if intno != 17 {
			log.Fatal("invalid interrupt ", intno, " at step ", st.Step)
		}

		syscallNum := st.Registers[2] // v0
		v0 := uint32(0)
		v1 := uint32(0)

		a0 := st.Registers[4]
		a1 := st.Registers[5]
		a2 := st.Registers[6]

		fmt.Printf("syscall: %d\n", syscallNum)
		switch syscallNum {
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
		case 4120: // clone (not supported)
			v0 = 1
		case 4246: // exit_group
			st.Exited = true
			st.ExitCode = uint8(a0)
			return
		case 4003: // read
			// args: a0 = fd, a1 = addr, a2 = count
			// returns: v0 = read, v1 = err code
			switch a0 {
			case fdStdin:
				// leave v0 and v1 zero: read nothing, no error
			case fdPreimageRead: // pre-image oracle
				effAddr := a1 & 0xFFffFFfc
				trackMemAccess(effAddr)
				mem := st.Memory.GetMemory(effAddr)
				dat, datLen := readPreimage(st.PreimageKey, st.PreimageOffset)
				fmt.Printf("reading pre-image data: addr: %08x, offset: %d, datLen: %d, data: %x, key: %s  count: %d\n", a1, st.PreimageOffset, datLen, dat[:datLen], st.PreimageKey, a2)
				alignment := a1 & 3
				space := 4 - alignment
				if space < datLen {
					datLen = space
				}
				if a2 < datLen {
					datLen = a2
				}
				var outMem [4]byte
				binary.BigEndian.PutUint32(outMem[:], mem)
				copy(outMem[alignment:], dat[:datLen])
				st.Memory.SetMemory(effAddr, binary.BigEndian.Uint32(outMem[:]))
				if err := mu.MemWrite(uint64(effAddr), outMem[:]); err != nil {
					log.Fatalf("failed to write pre-image data to memory: %v", err)
				}
				st.PreimageOffset += datLen
				v0 = datLen
				fmt.Printf("read %d pre-image bytes, new offset: %d, eff addr: %08x mem: %08x\n", datLen, st.PreimageOffset, effAddr, outMem)
			case fdHintRead: // hint response
				// don't actually read into memory, just say we read it all, we ignore the result anyway
				v0 = a2
			default:
				v0 = 0xFFffFFff
				v1 = MipsEBADF
			}
		case 4004: // write
			// args: a0 = fd, a1 = addr, a2 = count
			// returns: v0 = written, v1 = err code
			switch a0 {
			case fdStdout:
				_, _ = io.Copy(stdOut, st.Memory.ReadMemoryRange(a1, a2))
				v0 = a2
			case fdStderr:
				_, _ = io.Copy(stdErr, st.Memory.ReadMemoryRange(a1, a2))
				v0 = a2
			case fdHintWrite:
				hintData, _ := io.ReadAll(st.Memory.ReadMemoryRange(a1, a2))
				st.LastHint = append(st.LastHint, hintData...)
				for len(st.LastHint) >= 4 { // process while there is enough data to check if there are any hints
					hintLen := binary.BigEndian.Uint32(st.LastHint[:4])
					if hintLen >= uint32(len(st.LastHint[4:])) {
						hint := st.LastHint[4 : 4+hintLen] // without the length prefix
						st.LastHint = st.LastHint[4+hintLen:]
						po.Hint(hint)
					} else {
						break // stop processing hints if there is incomplete data buffered
					}
				}
				v0 = a2
			case fdPreimageWrite:
				effAddr := a1 & 0xFFffFFfc
				trackMemAccess(effAddr)
				mem := st.Memory.GetMemory(effAddr)
				key := st.PreimageKey
				alignment := a1 & 3
				space := 4 - alignment
				if space < a2 {
					a2 = space
				}
				copy(key[:], key[a2:])
				var tmp [4]byte
				binary.BigEndian.PutUint32(tmp[:], mem)
				copy(key[32-a2:], tmp[:])
				st.PreimageKey = key
				st.PreimageOffset = 0
				fmt.Printf("updating pre-image key: %s\n", st.PreimageKey)
				v0 = a2
			default:
				v0 = 0xFFffFFff
				v1 = MipsEBADF
			}
		case 4055: // fcntl
			// args: a0 = fd, a1 = cmd
			if a1 == 3 { // F_GETFL: get file descriptor flags
				switch a0 {
				case fdStdin, fdPreimageRead, fdHintRead:
					v0 = 0 // O_RDONLY
				case fdStdout, fdStderr, fdPreimageWrite, fdHintWrite:
					v0 = 1 // O_WRONLY
				default:
					v0 = 0xFFffFFff
					v1 = MipsEBADF
				}
			} else {
				v0 = 0xFFffFFff
				v1 = MipsEINVAL // cmd not recognized by this kernel
			}
		}
		_ = mu.RegWrite(uc.MIPS_REG_V0, uint64(v0))
		_ = mu.RegWrite(uc.MIPS_REG_A3, uint64(v1))
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
		trackMemAccess(effAddr)
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
		trackMemAccess(effAddr)
		// only set memory after making the proof: we need the pre-state
		st.Memory.SetMemory(effAddr, post)
	}, 0, ^uint64(0))
	if err != nil {
		return nil, fmt.Errorf("failed to set up mem-write hook: %w", err)
	}

	return m, nil
}

func (m *UnicornState) NonUnicornStep(proof bool) (wit *StepWitness, err error) {
	m.memProofEnabled = proof
	m.lastMemAccess = ^uint32(0)
	m.lastPreimageOffset = ^uint32(0)

	if proof {
		insnProof := m.state.Memory.MerkleProof(m.state.PC)
		wit = &StepWitness{
			state:    m.state.EncodeWitness(),
			memProof: insnProof[:],
		}
	}
	err = m.mipsStep()
	if err != nil {
		return nil, err
	}

	if proof {
		wit.memProof = append(wit.memProof, m.memProof[:]...)
		if m.lastPreimageOffset != ^uint32(0) {
			wit.preimageOffset = m.lastPreimageOffset
			wit.preimageKey = m.lastPreimageKey
			wit.preimageValue = m.lastPreimage
		}
	}
	return
}

func (m *UnicornState) Step(proof bool) (wit *StepWitness, err error) {
	defer func() { // pre-image oracle or emulator hooks might panic
		if a := recover(); a != nil {
			if ae, ok := a.(error); ok {
				err = ae
			} else {
				err = fmt.Errorf("panic: %v", a)
			}
		}
	}()

	m.memProofEnabled = proof
	m.lastMemAccess = ^uint32(0)
	m.lastPreimageOffset = ^uint32(0)

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
	err = m.mu.StartWithOptions(uint64(m.state.PC), uint64(m.state.NextPC), &uc.UcOptions{
		Timeout: 0, // 0 to disable, value is in ms.
		Count:   1,
	})
	if err != nil {
		return nil, err
	}

	if proof {
		wit.memProof = append(wit.memProof, m.memProof[:]...)
		if m.lastPreimageOffset != ^uint32(0) {
			wit.preimageOffset = m.lastPreimageOffset
			wit.preimageKey = m.lastPreimageKey
			wit.preimageValue = m.lastPreimage
		}
	}

	// count it
	m.state.Step += 1

	// Now do post-processing to keep our state in sync:

	// 1) match the registers post-state
	batch, err := m.mu.RegReadBatch(regBatchKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to read register batch: %w", err)
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
			return nil, fmt.Errorf("failed to write PC register: %w", err)
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
