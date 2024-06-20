package mipsevm

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	sysMmap      = 4090
	sysBrk       = 4045
	sysClone     = 4120
	sysExitGroup = 4246
	sysRead      = 4003
	sysWrite     = 4004
	sysFcntl     = 4055
)

func (m *InstrumentedState) readPreimage(key [32]byte, offset uint32) (dat [32]byte, datLen uint32) {
	preimage := m.lastPreimage
	if key != m.lastPreimageKey {
		m.lastPreimageKey = key
		data := m.preimageOracle.GetPreimage(key)
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

func (m *InstrumentedState) trackMemAccess(effAddr uint32) {
	if m.memProofEnabled && m.lastMemAccess != effAddr {
		if m.lastMemAccess != ^uint32(0) {
			panic(fmt.Errorf("unexpected different mem access at %08x, already have access at %08x buffered", effAddr, m.lastMemAccess))
		}
		m.lastMemAccess = effAddr
		m.memProof = m.state.Memory.MerkleProof(effAddr)
	}
}

func (m *InstrumentedState) handleSyscall() error {
	syscallNum := m.state.Registers[2] // v0
	v0 := uint32(0)
	v1 := uint32(0)

	a0 := m.state.Registers[4]
	a1 := m.state.Registers[5]
	a2 := m.state.Registers[6]

	//fmt.Printf("syscall: %d\n", syscallNum)
	switch syscallNum {
	case sysMmap:
		sz := a1
		if sz&PageAddrMask != 0 { // adjust size to align with page size
			sz += PageSize - (sz & PageAddrMask)
		}
		if a0 == 0 {
			v0 = m.state.Heap
			//fmt.Printf("mmap heap 0x%x size 0x%x\n", v0, sz)
			m.state.Heap += sz
		} else {
			v0 = a0
			//fmt.Printf("mmap hint 0x%x size 0x%x\n", v0, sz)
		}
	case sysBrk:
		v0 = 0x40000000
	case sysClone: // clone (not supported)
		v0 = 1
	case sysExitGroup:
		m.state.Exited = true
		m.state.ExitCode = uint8(a0)
		return nil
	case sysRead:
		// args: a0 = fd, a1 = addr, a2 = count
		// returns: v0 = read, v1 = err code
		switch a0 {
		case fdStdin:
			// leave v0 and v1 zero: read nothing, no error
		case fdPreimageRead: // pre-image oracle
			effAddr := a1 & 0xFFffFFfc
			m.trackMemAccess(effAddr)
			mem := m.state.Memory.GetMemory(effAddr)
			dat, datLen := m.readPreimage(m.state.PreimageKey, m.state.PreimageOffset)
			//fmt.Printf("reading pre-image data: addr: %08x, offset: %d, datLen: %d, data: %x, key: %s  count: %d\n", a1, m.state.PreimageOffset, datLen, dat[:datLen], m.state.PreimageKey, a2)
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
			m.state.Memory.SetMemory(effAddr, binary.BigEndian.Uint32(outMem[:]))
			m.state.PreimageOffset += datLen
			v0 = datLen
			//fmt.Printf("read %d pre-image bytes, new offset: %d, eff addr: %08x mem: %08x\n", datLen, m.state.PreimageOffset, effAddr, outMem)
		case fdHintRead: // hint response
			// don't actually read into memory, just say we read it all, we ignore the result anyway
			v0 = a2
		default:
			v0 = 0xFFffFFff
			v1 = MipsEBADF
		}
	case sysWrite:
		// args: a0 = fd, a1 = addr, a2 = count
		// returns: v0 = written, v1 = err code
		switch a0 {
		case fdStdout:
			_, _ = io.Copy(m.stdOut, m.state.Memory.ReadMemoryRange(a1, a2))
			v0 = a2
		case fdStderr:
			_, _ = io.Copy(m.stdErr, m.state.Memory.ReadMemoryRange(a1, a2))
			v0 = a2
		case fdHintWrite:
			hintData, _ := io.ReadAll(m.state.Memory.ReadMemoryRange(a1, a2))
			m.state.LastHint = append(m.state.LastHint, hintData...)
			for len(m.state.LastHint) >= 4 { // process while there is enough data to check if there are any hints
				hintLen := binary.BigEndian.Uint32(m.state.LastHint[:4])
				if hintLen <= uint32(len(m.state.LastHint[4:])) {
					hint := m.state.LastHint[4 : 4+hintLen] // without the length prefix
					m.state.LastHint = m.state.LastHint[4+hintLen:]
					m.preimageOracle.Hint(hint)
				} else {
					break // stop processing hints if there is incomplete data buffered
				}
			}
			v0 = a2
		case fdPreimageWrite:
			effAddr := a1 & 0xFFffFFfc
			m.trackMemAccess(effAddr)
			mem := m.state.Memory.GetMemory(effAddr)
			key := m.state.PreimageKey
			alignment := a1 & 3
			space := 4 - alignment
			if space < a2 {
				a2 = space
			}
			copy(key[:], key[a2:])
			var tmp [4]byte
			binary.BigEndian.PutUint32(tmp[:], mem)
			copy(key[32-a2:], tmp[alignment:])
			m.state.PreimageKey = key
			m.state.PreimageOffset = 0
			//fmt.Printf("updating pre-image key: %s\n", m.state.PreimageKey)
			v0 = a2
		default:
			v0 = 0xFFffFFff
			v1 = MipsEBADF
		}
	case sysFcntl:
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
	m.state.Registers[2] = v0
	m.state.Registers[7] = v1

	m.state.Cpu.PC = m.state.Cpu.NextPC
	m.state.Cpu.NextPC = m.state.Cpu.NextPC + 4
	return nil
}

func (m *InstrumentedState) pushStack(target uint32) {
	if !m.debugEnabled {
		return
	}
	m.debug.stack = append(m.debug.stack, target)
	m.debug.caller = append(m.debug.caller, m.state.Cpu.PC)
}

func (m *InstrumentedState) popStack() {
	if !m.debugEnabled {
		return
	}
	if len(m.debug.stack) != 0 {
		fn := m.debug.meta.LookupSymbol(m.state.Cpu.PC)
		topFn := m.debug.meta.LookupSymbol(m.debug.stack[len(m.debug.stack)-1])
		if fn != topFn {
			// most likely the function was inlined. Snap back to the last return.
			i := len(m.debug.stack) - 1
			for ; i >= 0; i-- {
				if m.debug.meta.LookupSymbol(m.debug.stack[i]) == fn {
					m.debug.stack = m.debug.stack[:i]
					m.debug.caller = m.debug.caller[:i]
					break
				}
			}
		} else {
			m.debug.stack = m.debug.stack[:len(m.debug.stack)-1]
			m.debug.caller = m.debug.caller[:len(m.debug.caller)-1]
		}
	} else {
		fmt.Printf("ERROR: stack underflow at pc=%x. step=%d\n", m.state.Cpu.PC, m.state.Step)
	}
}

func (m *InstrumentedState) Traceback() {
	fmt.Printf("traceback at pc=%x. step=%d\n", m.state.Cpu.PC, m.state.Step)
	for i := len(m.debug.stack) - 1; i >= 0; i-- {
		s := m.debug.stack[i]
		idx := len(m.debug.stack) - i - 1
		fmt.Printf("\t%d %x in %s caller=%08x\n", idx, s, m.debug.meta.LookupSymbol(s), m.debug.caller[i])
	}
}

func (m *InstrumentedState) mipsStep() error {
	if m.state.Exited {
		return nil
	}
	m.state.Step += 1
	// instruction fetch
	insn := m.state.Memory.GetMemory(m.state.Cpu.PC)
	opcode := insn >> 26 // 6-bits

	// j-type j/jal
	if opcode == 2 || opcode == 3 {
		linkReg := uint32(0)
		if opcode == 3 {
			linkReg = 31
		}
		// Take top 4 bits of the next PC (its 256 MB region), and concatenate with the 26-bit offset
		target := (m.state.Cpu.NextPC & 0xF0000000) | ((insn & 0x03FFFFFF) << 2)
		m.pushStack(target)
		return handleJump(&m.state.Cpu, &m.state.Registers, linkReg, target)
	}

	// register fetch
	rs := uint32(0) // source register 1 value
	rt := uint32(0) // source register 2 / temp value
	rtReg := (insn >> 16) & 0x1F

	// R-type or I-type (stores rt)
	rs = m.state.Registers[(insn>>21)&0x1F]
	rdReg := rtReg
	if opcode == 0 || opcode == 0x1c {
		// R-type (stores rd)
		rt = m.state.Registers[rtReg]
		rdReg = (insn >> 11) & 0x1F
	} else if opcode < 0x20 {
		// rt is SignExtImm
		// don't sign extend for andi, ori, xori
		if opcode == 0xC || opcode == 0xD || opcode == 0xe {
			// ZeroExtImm
			rt = insn & 0xFFFF
		} else {
			// SignExtImm
			rt = signExtend(insn&0xFFFF, 16)
		}
	} else if opcode >= 0x28 || opcode == 0x22 || opcode == 0x26 {
		// store rt value with store
		rt = m.state.Registers[rtReg]

		// store actual rt with lwl and lwr
		rdReg = rtReg
	}

	if (opcode >= 4 && opcode < 8) || opcode == 1 {
		return handleBranch(&m.state.Cpu, &m.state.Registers, opcode, insn, rtReg, rs)
	}

	storeAddr := uint32(0xFF_FF_FF_FF)
	// memory fetch (all I-type)
	// we do the load for stores also
	mem := uint32(0)
	if opcode >= 0x20 {
		// M[R[rs]+SignExtImm]
		rs += signExtend(insn&0xFFFF, 16)
		addr := rs & 0xFFFFFFFC
		m.trackMemAccess(addr)
		mem = m.state.Memory.GetMemory(addr)
		if opcode >= 0x28 && opcode != 0x30 {
			// store
			storeAddr = addr
			// store opcodes don't write back to a register
			rdReg = 0
		}
	}

	// ALU
	val := executeMipsInstruction(insn, rs, rt, mem)

	fun := insn & 0x3f // 6-bits
	if opcode == 0 && fun >= 8 && fun < 0x1c {
		if fun == 8 || fun == 9 { // jr/jalr
			linkReg := uint32(0)
			if fun == 9 {
				linkReg = rdReg
			}
			m.popStack()
			return handleJump(&m.state.Cpu, &m.state.Registers, linkReg, rs)
		}

		if fun == 0xa { // movz
			return handleRd(&m.state.Cpu, &m.state.Registers, rdReg, rs, rt == 0)
		}
		if fun == 0xb { // movn
			return handleRd(&m.state.Cpu, &m.state.Registers, rdReg, rs, rt != 0)
		}

		// syscall (can read and write)
		if fun == 0xC {
			return m.handleSyscall()
		}

		// lo and hi registers
		// can write back
		if fun >= 0x10 && fun < 0x1c {
			return handleHiLo(&m.state.Cpu, &m.state.Registers, fun, rs, rt, rdReg)
		}
	}

	// stupid sc, write a 1 to rt
	if opcode == 0x38 && rtReg != 0 {
		m.state.Registers[rtReg] = 1
	}

	// write memory
	if storeAddr != 0xFF_FF_FF_FF {
		m.trackMemAccess(storeAddr)
		m.state.Memory.SetMemory(storeAddr, val)
	}

	// write back the value to destination register
	return handleRd(&m.state.Cpu, &m.state.Registers, rdReg, val, true)
}
