package mipsevm

import (
	"encoding/binary"
	"fmt"
	"io"
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
	case 4090: // mmap
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
	case 4045: // brk
		v0 = 0x40000000
	case 4120: // clone (not supported)
		v0 = 1
	case 4246: // exit_group
		m.state.Exited = true
		m.state.ExitCode = uint8(a0)
		return nil
	case 4003: // read
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
	case 4004: // write
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
				if hintLen >= uint32(len(m.state.LastHint[4:])) {
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
			copy(key[32-a2:], tmp[:])
			m.state.PreimageKey = key
			m.state.PreimageOffset = 0
			//fmt.Printf("updating pre-image key: %s\n", m.state.PreimageKey)
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
	m.state.Registers[2] = v0
	m.state.Registers[7] = v1

	m.state.PC = m.state.NextPC
	m.state.NextPC = m.state.NextPC + 4
	return nil
}

func (m *InstrumentedState) handleBranch(opcode uint32, insn uint32, rtReg uint32, rs uint32) error {
	shouldBranch := false
	if opcode == 4 || opcode == 5 { // beq/bne
		rt := m.state.Registers[rtReg]
		shouldBranch = (rs == rt && opcode == 4) || (rs != rt && opcode == 5)
	} else if opcode == 6 {
		shouldBranch = int32(rs) <= 0 // blez
	} else if opcode == 7 {
		shouldBranch = int32(rs) > 0 // bgtz
	} else if opcode == 1 {
		// regimm
		rtv := (insn >> 16) & 0x1F
		if rtv == 0 { // bltz
			shouldBranch = int32(rs) < 0
		}
		if rtv == 1 { // bgez
			shouldBranch = int32(rs) >= 0
		}
	}

	prevPC := m.state.PC
	m.state.PC = m.state.NextPC // execute the delay slot first
	if shouldBranch {
		m.state.NextPC = prevPC + 4 + (SE(insn&0xFFFF, 16) << 2) // then continue with the instruction the branch jumps to.
	} else {
		m.state.NextPC = m.state.NextPC + 4 // branch not taken
	}
	return nil
}

func (m *InstrumentedState) handleHiLo(fun uint32, rs uint32, rt uint32, storeReg uint32) error {
	val := uint32(0)
	switch fun {
	case 0x10: // mfhi
		val = m.state.HI
	case 0x11: // mthi
		m.state.HI = rs
	case 0x12: // mflo
		val = m.state.LO
	case 0x13: // mtlo
		m.state.LO = rs
	case 0x18: // mult
		acc := uint64(int64(int32(rs)) * int64(int32(rt)))
		m.state.HI = uint32(acc >> 32)
		m.state.LO = uint32(acc)
	case 0x19: // multu
		acc := uint64(uint64(rs) * uint64(rt))
		m.state.HI = uint32(acc >> 32)
		m.state.LO = uint32(acc)
	case 0x1a: // div
		m.state.HI = uint32(int32(rs) % int32(rt))
		m.state.LO = uint32(int32(rs) / int32(rt))
	case 0x1b: // divu
		m.state.HI = rs % rt
		m.state.LO = rs / rt
	}

	if storeReg != 0 {
		m.state.Registers[storeReg] = val
	}

	m.state.PC = m.state.NextPC
	m.state.NextPC = m.state.NextPC + 4
	return nil
}

func (m *InstrumentedState) handleJump(linkReg uint32, dest uint32) error {
	prevPC := m.state.PC
	m.state.PC = m.state.NextPC
	m.state.NextPC = dest
	if linkReg != 0 {
		m.state.Registers[linkReg] = prevPC + 8 // set the link-register to the instr after the delay slot instruction.
	}
	return nil
}

func (m *InstrumentedState) handleRd(storeReg uint32, val uint32, conditional bool) error {
	if storeReg >= 32 {
		panic("invalid register")
	}
	if storeReg != 0 && conditional {
		m.state.Registers[storeReg] = val
	}
	m.state.PC = m.state.NextPC
	m.state.NextPC = m.state.NextPC + 4
	return nil
}

func (m *InstrumentedState) mipsStep() error {
	if m.state.Exited {
		return nil
	}
	m.state.Step += 1
	// instruction fetch
	insn := m.state.Memory.GetMemory(m.state.PC)
	opcode := insn >> 26 // 6-bits

	// j-type j/jal
	if opcode == 2 || opcode == 3 {
		// TODO likely bug in original code: MIPS spec says this should be in the "current" region;
		// a 256 MB aligned region (i.e. use top 4 bits of branch delay slot (pc+4))
		linkReg := uint32(0)
		if opcode == 3 {
			linkReg = 31
		}
		return m.handleJump(linkReg, SE(insn&0x03FFFFFF, 26)<<2)
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
			rt = SE(insn&0xFFFF, 16)
		}
	} else if opcode >= 0x28 || opcode == 0x22 || opcode == 0x26 {
		// store rt value with store
		rt = m.state.Registers[rtReg]

		// store actual rt with lwl and lwr
		rdReg = rtReg
	}

	if (opcode >= 4 && opcode < 8) || opcode == 1 {
		return m.handleBranch(opcode, insn, rtReg, rs)
	}

	storeAddr := uint32(0xFF_FF_FF_FF)
	// memory fetch (all I-type)
	// we do the load for stores also
	mem := uint32(0)
	if opcode >= 0x20 {
		// M[R[rs]+SignExtImm]
		rs += SE(insn&0xFFFF, 16)
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
	val := execute(insn, rs, rt, mem)

	fun := insn & 0x3f // 6-bits
	if opcode == 0 && fun >= 8 && fun < 0x1c {
		if fun == 8 || fun == 9 { // jr/jalr
			linkReg := uint32(0)
			if fun == 9 {
				linkReg = rdReg
			}
			return m.handleJump(linkReg, rs)
		}

		if fun == 0xa { // movz
			return m.handleRd(rdReg, rs, rt == 0)
		}
		if fun == 0xb { // movn
			return m.handleRd(rdReg, rs, rt != 0)
		}

		// syscall (can read and write)
		if fun == 0xC {
			return m.handleSyscall()
		}

		// lo and hi registers
		// can write back
		if fun >= 0x10 && fun < 0x1c {
			return m.handleHiLo(fun, rs, rt, rdReg)
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
	return m.handleRd(rdReg, val, true)
}

func execute(insn uint32, rs uint32, rt uint32, mem uint32) uint32 {
	opcode := insn >> 26 // 6-bits
	fun := insn & 0x3f   // 6-bits
	// TODO(CLI-4136): deref the immed into a register

	if opcode < 0x20 {
		// transform ArithLogI
		// TODO(CLI-4136): replace with table
		if opcode >= 8 && opcode < 0xF {
			switch opcode {
			case 8:
				fun = 0x20 // addi
			case 9:
				fun = 0x21 // addiu
			case 0xA:
				fun = 0x2A // slti
			case 0xB:
				fun = 0x2B // sltiu
			case 0xC:
				fun = 0x24 // andi
			case 0xD:
				fun = 0x25 // ori
			case 0xE:
				fun = 0x26 // xori
			}
			opcode = 0
		}

		// 0 is opcode SPECIAL
		if opcode == 0 {
			shamt := (insn >> 6) & 0x1F
			if fun < 0x20 {
				switch {
				case fun >= 0x08:
					return rs // jr/jalr/div + others
				case fun == 0x00:
					return rt << shamt // sll
				case fun == 0x02:
					return rt >> shamt // srl
				case fun == 0x03:
					return SE(rt>>shamt, 32-shamt) // sra
				case fun == 0x04:
					return rt << (rs & 0x1F) // sllv
				case fun == 0x06:
					return rt >> (rs & 0x1F) // srlv
				case fun == 0x07:
					return SE(rt>>rs, 32-rs) // srav
				}
			}
			// 0x10-0x13 = mfhi, mthi, mflo, mtlo
			// R-type (ArithLog)
			switch fun {
			case 0x20, 0x21:
				return rs + rt // add or addu
			case 0x22, 0x23:
				return rs - rt // sub or subu
			case 0x24:
				return rs & rt // and
			case 0x25:
				return rs | rt // or
			case 0x26:
				return rs ^ rt // xor
			case 0x27:
				return ^(rs | rt) // nor
			case 0x2A:
				if int32(rs) < int32(rt) {
					return 1 // slt
				} else {
					return 0
				}
			case 0x2B:
				if rs < rt {
					return 1 // sltu
				} else {
					return 0
				}
			}
		} else if opcode == 0xF {
			return rt << 16 // lui
		} else if opcode == 0x1C { // SPECIAL2
			if fun == 2 { // mul
				return uint32(int32(rs) * int32(rt))
			}
			if fun == 0x20 || fun == 0x21 { // clo
				if fun == 0x20 {
					rs = ^rs
				}
				i := uint32(0)
				for ; rs&0x80000000 != 0; i++ {
					rs <<= 1
				}
				return i
			}
		}
	} else if opcode < 0x28 {
		switch opcode {
		case 0x20: // lb
			return SE((mem>>(24-(rs&3)*8))&0xFF, 8)
		case 0x21: // lh
			return SE((mem>>(16-(rs&2)*8))&0xFFFF, 16)
		case 0x22: // lwl
			val := mem << ((rs & 3) * 8)
			mask := uint32(0xFFFFFFFF) << ((rs & 3) * 8)
			return (rt & ^mask) | val
		case 0x23: // lw
			return mem
		case 0x24: // lbu
			return (mem >> (24 - (rs&3)*8)) & 0xFF
		case 0x25: // lhu
			return (mem >> (16 - (rs&2)*8)) & 0xFFFF
		case 0x26: // lwr
			val := mem >> (24 - (rs&3)*8)
			mask := uint32(0xFFFFFFFF) >> (24 - (rs&3)*8)
			return (rt & ^mask) | val
		}
	} else if opcode == 0x28 { // sb
		val := (rt & 0xFF) << (24 - (rs&3)*8)
		mask := 0xFFFFFFFF ^ uint32(0xFF<<(24-(rs&3)*8))
		return (mem & mask) | val
	} else if opcode == 0x29 { // sh
		val := (rt & 0xFFFF) << (16 - (rs&2)*8)
		mask := 0xFFFFFFFF ^ uint32(0xFFFF<<(16-(rs&2)*8))
		return (mem & mask) | val
	} else if opcode == 0x2a { // swl
		val := rt >> ((rs & 3) * 8)
		mask := uint32(0xFFFFFFFF) >> ((rs & 3) * 8)
		return (mem & ^mask) | val
	} else if opcode == 0x2b { // sw
		return rt
	} else if opcode == 0x2e { // swr
		val := rt << (24 - (rs&3)*8)
		mask := uint32(0xFFFFFFFF) << (24 - (rs&3)*8)
		return (mem & ^mask) | val
	} else if opcode == 0x30 {
		return mem // ll
	} else if opcode == 0x38 {
		return rt // sc
	}

	panic("invalid instruction")
}

func SE(dat uint32, idx uint32) uint32 {
	isSigned := (dat >> (idx - 1)) != 0
	signed := ((uint32(1) << (32 - idx)) - 1) << idx
	mask := (uint32(1) << idx) - 1
	if isSigned {
		return dat&mask | signed
	} else {
		return dat & mask
	}
}
