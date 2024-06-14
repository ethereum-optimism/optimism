package mipsevm

func executeMipsInstruction(insn uint32, rs uint32, rt uint32, mem uint32) uint32 {
	opcode := insn >> 26 // 6-bits

	if opcode == 0 || (opcode >= 8 && opcode < 0xF) {
		fun := insn & 0x3f // 6-bits
		// transform ArithLogI to SPECIAL
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

		switch fun {
		case 0x00: // sll
			return rt << ((insn >> 6) & 0x1F)
		case 0x02: // srl
			return rt >> ((insn >> 6) & 0x1F)
		case 0x03: // sra
			shamt := (insn >> 6) & 0x1F
			return signExtend(rt>>shamt, 32-shamt)
		case 0x04: // sllv
			return rt << (rs & 0x1F)
		case 0x06: // srlv
			return rt >> (rs & 0x1F)
		case 0x07: // srav
			return signExtend(rt>>rs, 32-rs)
		// functs in range [0x8, 0x1b] are handled specially by other functions
		case 0x08: // jr
			return rs
		case 0x09: // jalr
			return rs
		case 0x0a: // movz
			return rs
		case 0x0b: // movn
			return rs
		case 0x0c: // syscall
			return rs
		// 0x0d - break not supported
		case 0x0f: // sync
			return rs
		case 0x10: // mfhi
			return rs
		case 0x11: // mthi
			return rs
		case 0x12: // mflo
			return rs
		case 0x13: // mtlo
			return rs
		case 0x18: // mult
			return rs
		case 0x19: // multu
			return rs
		case 0x1a: // div
			return rs
		case 0x1b: // divu
			return rs
		// The rest includes transformed R-type arith imm instructions
		case 0x20: // add
			return rs + rt
		case 0x21: // addu
			return rs + rt
		case 0x22: // sub
			return rs - rt
		case 0x23: // subu
			return rs - rt
		case 0x24: // and
			return rs & rt
		case 0x25: // or
			return rs | rt
		case 0x26: // xor
			return rs ^ rt
		case 0x27: // nor
			return ^(rs | rt)
		case 0x2a: // slti
			if int32(rs) < int32(rt) {
				return 1
			}
			return 0
		case 0x2b: // sltiu
			if rs < rt {
				return 1
			}
			return 0
		default:
			panic("invalid instruction")
		}
	} else {
		switch opcode {
		// SPECIAL2
		case 0x1C:
			fun := insn & 0x3f // 6-bits
			switch fun {
			case 0x2: // mul
				return uint32(int32(rs) * int32(rt))
			case 0x20, 0x21: // clz, clo
				if fun == 0x20 {
					rs = ^rs
				}
				i := uint32(0)
				for ; rs&0x80000000 != 0; i++ {
					rs <<= 1
				}
				return i
			}
		case 0x0F: // lui
			return rt << 16
		case 0x20: // lb
			return signExtend((mem>>(24-(rs&3)*8))&0xFF, 8)
		case 0x21: // lh
			return signExtend((mem>>(16-(rs&2)*8))&0xFFFF, 16)
		case 0x22: // lwl
			val := mem << ((rs & 3) * 8)
			mask := uint32(0xFFFFFFFF) << ((rs & 3) * 8)
			return (rt & ^mask) | val
		case 0x23: // lw
			return mem
		case 0x24: // lbu
			return (mem >> (24 - (rs&3)*8)) & 0xFF
		case 0x25: //  lhu
			return (mem >> (16 - (rs&2)*8)) & 0xFFFF
		case 0x26: //  lwr
			val := mem >> (24 - (rs&3)*8)
			mask := uint32(0xFFFFFFFF) >> (24 - (rs&3)*8)
			return (rt & ^mask) | val
		case 0x28: //  sb
			val := (rt & 0xFF) << (24 - (rs&3)*8)
			mask := 0xFFFFFFFF ^ uint32(0xFF<<(24-(rs&3)*8))
			return (mem & mask) | val
		case 0x29: //  sh
			val := (rt & 0xFFFF) << (16 - (rs&2)*8)
			mask := 0xFFFFFFFF ^ uint32(0xFFFF<<(16-(rs&2)*8))
			return (mem & mask) | val
		case 0x2a: //  swl
			val := rt >> ((rs & 3) * 8)
			mask := uint32(0xFFFFFFFF) >> ((rs & 3) * 8)
			return (mem & ^mask) | val
		case 0x2b: //  sw
			return rt
		case 0x2e: //  swr
			val := rt << (24 - (rs&3)*8)
			mask := uint32(0xFFFFFFFF) << (24 - (rs&3)*8)
			return (mem & ^mask) | val
		case 0x30: //  ll
			return mem
		case 0x38: //  sc
			return rt
		default:
			panic("invalid instruction")
		}
	}
	panic("invalid instruction")
}

func signExtend(dat uint32, idx uint32) uint32 {
	isSigned := (dat >> (idx - 1)) != 0
	signed := ((uint32(1) << (32 - idx)) - 1) << idx
	mask := (uint32(1) << idx) - 1
	if isSigned {
		return dat&mask | signed
	} else {
		return dat & mask
	}
}

func handleBranch(cpu *CpuScalars, registers *[32]uint32, opcode uint32, insn uint32, rtReg uint32, rs uint32) error {
	if cpu.NextPC != cpu.PC+4 {
		panic("branch in delay slot")
	}

	shouldBranch := false
	if opcode == 4 || opcode == 5 { // beq/bne
		rt := registers[rtReg]
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

	prevPC := cpu.PC
	cpu.PC = cpu.NextPC // execute the delay slot first
	if shouldBranch {
		cpu.NextPC = prevPC + 4 + (signExtend(insn&0xFFFF, 16) << 2) // then continue with the instruction the branch jumps to.
	} else {
		cpu.NextPC = cpu.NextPC + 4 // branch not taken
	}
	return nil
}

func handleHiLo(cpu *CpuScalars, registers *[32]uint32, fun uint32, rs uint32, rt uint32, storeReg uint32) error {
	val := uint32(0)
	switch fun {
	case 0x10: // mfhi
		val = cpu.HI
	case 0x11: // mthi
		cpu.HI = rs
	case 0x12: // mflo
		val = cpu.LO
	case 0x13: // mtlo
		cpu.LO = rs
	case 0x18: // mult
		acc := uint64(int64(int32(rs)) * int64(int32(rt)))
		cpu.HI = uint32(acc >> 32)
		cpu.LO = uint32(acc)
	case 0x19: // multu
		acc := uint64(uint64(rs) * uint64(rt))
		cpu.HI = uint32(acc >> 32)
		cpu.LO = uint32(acc)
	case 0x1a: // div
		cpu.HI = uint32(int32(rs) % int32(rt))
		cpu.LO = uint32(int32(rs) / int32(rt))
	case 0x1b: // divu
		cpu.HI = rs % rt
		cpu.LO = rs / rt
	}

	if storeReg != 0 {
		registers[storeReg] = val
	}

	cpu.PC = cpu.NextPC
	cpu.NextPC = cpu.NextPC + 4
	return nil
}

func handleJump(cpu *CpuScalars, registers *[32]uint32, linkReg uint32, dest uint32) error {
	if cpu.NextPC != cpu.PC+4 {
		panic("jump in delay slot")
	}
	prevPC := cpu.PC
	cpu.PC = cpu.NextPC
	cpu.NextPC = dest
	if linkReg != 0 {
		registers[linkReg] = prevPC + 8 // set the link-register to the instr after the delay slot instruction.
	}
	return nil
}

func handleRd(cpu *CpuScalars, registers *[32]uint32, storeReg uint32, val uint32, conditional bool) error {
	if storeReg >= 32 {
		panic("invalid register")
	}
	if storeReg != 0 && conditional {
		registers[storeReg] = val
	}
	cpu.PC = cpu.NextPC
	cpu.NextPC = cpu.NextPC + 4
	return nil
}
