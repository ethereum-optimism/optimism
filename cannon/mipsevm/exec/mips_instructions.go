package exec

import (
	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
)

const (
	OpLoadLinked       = 0x30
	OpStoreConditional = 0x38
)

func GetInstructionDetails(pc uint32, memory *memory.Memory) (insn, opcode, fun uint32) {
	insn = memory.GetMemory(pc)
	opcode = insn >> 26 // First 6-bits
	fun = insn & 0x3f   // Last 6-bits

	return insn, opcode, fun
}

func ExecMipsCoreStepLogic(cpu *mipsevm.CpuScalars, registers *[32]uint32, memory *memory.Memory, insn, opcode, fun uint32, memTracker MemTracker, stackTracker StackTracker) (memUpdated bool, memAddr uint32, err error) {
	// j-type j/jal
	if opcode == 2 || opcode == 3 {
		linkReg := uint32(0)
		if opcode == 3 {
			linkReg = 31
		}
		// Take top 4 bits of the next PC (its 256 MB region), and concatenate with the 26-bit offset
		target := (cpu.NextPC & 0xF0000000) | ((insn & 0x03FFFFFF) << 2)
		stackTracker.PushStack(cpu.PC, target)
		err = HandleJump(cpu, registers, linkReg, target)
		return
	}

	// register fetch
	rs := uint32(0) // source register 1 value
	rt := uint32(0) // source register 2 / temp value
	rtReg := (insn >> 16) & 0x1F

	// R-type or I-type (stores rt)
	rs = registers[(insn>>21)&0x1F]
	rdReg := rtReg
	if opcode == 0 || opcode == 0x1c {
		// R-type (stores rd)
		rt = registers[rtReg]
		rdReg = (insn >> 11) & 0x1F
	} else if opcode < 0x20 {
		// rt is SignExtImm
		// don't sign extend for andi, ori, xori
		if opcode == 0xC || opcode == 0xD || opcode == 0xe {
			// ZeroExtImm
			rt = insn & 0xFFFF
		} else {
			// SignExtImm
			rt = SignExtend(insn&0xFFFF, 16)
		}
	} else if opcode >= 0x28 || opcode == 0x22 || opcode == 0x26 {
		// store rt value with store
		rt = registers[rtReg]

		// store actual rt with lwl and lwr
		rdReg = rtReg
	}

	if (opcode >= 4 && opcode < 8) || opcode == 1 {
		err = HandleBranch(cpu, registers, opcode, insn, rtReg, rs)
		return
	}

	storeAddr := uint32(0xFF_FF_FF_FF)
	// memory fetch (all I-type)
	// we do the load for stores also
	mem := uint32(0)
	if opcode >= 0x20 {
		// M[R[rs]+SignExtImm]
		rs += SignExtend(insn&0xFFFF, 16)
		addr := rs & 0xFFFFFFFC
		memTracker.TrackMemAccess(addr)
		mem = memory.GetMemory(addr)
		if opcode >= 0x28 {
			// store
			storeAddr = addr
			// store opcodes don't write back to a register
			rdReg = 0
		}
	}

	// ALU
	val := ExecuteMipsInstruction(insn, opcode, fun, rs, rt, mem)

	if opcode == 0 && fun >= 8 && fun < 0x1c {
		if fun == 8 || fun == 9 { // jr/jalr
			linkReg := uint32(0)
			if fun == 9 {
				linkReg = rdReg
				stackTracker.PushStack(cpu.PC, rs)
			} else {
				stackTracker.PopStack()
			}
			err = HandleJump(cpu, registers, linkReg, rs)
			return
		}

		if fun == 0xa { // movz
			err = HandleRd(cpu, registers, rdReg, rs, rt == 0)
			return
		}
		if fun == 0xb { // movn
			err = HandleRd(cpu, registers, rdReg, rs, rt != 0)
			return
		}

		// lo and hi registers
		// can write back
		if fun >= 0x10 && fun < 0x1c {
			err = HandleHiLo(cpu, registers, fun, rs, rt, rdReg)
			return
		}
	}

	// write memory
	if storeAddr != 0xFF_FF_FF_FF {
		memTracker.TrackMemAccess(storeAddr)
		memory.SetMemory(storeAddr, val)
		memUpdated = true
		memAddr = storeAddr
	}

	// write back the value to destination register
	err = HandleRd(cpu, registers, rdReg, val, true)
	return
}

func SignExtendImmediate(insn uint32) uint32 {
	return SignExtend(insn&0xFFFF, 16)
}

func ExecuteMipsInstruction(insn, opcode, fun, rs, rt, mem uint32) uint32 {
	if opcode == 0 || (opcode >= 8 && opcode < 0xF) {
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
			return SignExtend(rt>>shamt, 32-shamt)
		case 0x04: // sllv
			return rt << (rs & 0x1F)
		case 0x06: // srlv
			return rt >> (rs & 0x1F)
		case 0x07: // srav
			shamt := rs & 0x1F
			return SignExtend(rt>>shamt, 32-shamt)
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
			return SignExtend((mem>>(24-(rs&3)*8))&0xFF, 8)
		case 0x21: // lh
			return SignExtend((mem>>(16-(rs&2)*8))&0xFFFF, 16)
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
		default:
			panic("invalid instruction")
		}
	}
	panic("invalid instruction")
}

func SignExtend(dat uint32, idx uint32) uint32 {
	isSigned := (dat >> (idx - 1)) != 0
	signed := ((uint32(1) << (32 - idx)) - 1) << idx
	mask := (uint32(1) << idx) - 1
	if isSigned {
		return dat&mask | signed
	} else {
		return dat & mask
	}
}

func HandleBranch(cpu *mipsevm.CpuScalars, registers *[32]uint32, opcode uint32, insn uint32, rtReg uint32, rs uint32) error {
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
		cpu.NextPC = prevPC + 4 + (SignExtend(insn&0xFFFF, 16) << 2) // then continue with the instruction the branch jumps to.
	} else {
		cpu.NextPC = cpu.NextPC + 4 // branch not taken
	}
	return nil
}

func HandleHiLo(cpu *mipsevm.CpuScalars, registers *[32]uint32, fun uint32, rs uint32, rt uint32, storeReg uint32) error {
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

func HandleJump(cpu *mipsevm.CpuScalars, registers *[32]uint32, linkReg uint32, dest uint32) error {
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

func HandleRd(cpu *mipsevm.CpuScalars, registers *[32]uint32, storeReg uint32, val uint32, conditional bool) error {
	if storeReg >= 32 {
		panic("invalid register")
	}
	if storeReg != 0 && conditional {
		// Register 0 is a special register that always holds a value of 0
		registers[storeReg] = val
	}
	cpu.PC = cpu.NextPC
	cpu.NextPC = cpu.NextPC + 4
	return nil
}
