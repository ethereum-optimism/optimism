package exec

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
	u128 "lukechampine.com/uint128"
)

func GetInstructionDetails(pc uint64, memory *memory.Memory) (insn, opcode, fun uint64) {
	insn = uint64(memory.GetMemory(pc))
	opcode = insn >> 26 // First 6-bits
	fun = insn & 0x3f   // Last 6-bits

	return insn, opcode, fun
}

func ExecMipsCoreStepLogic(cpu *mipsevm.CpuScalars, registers *[32]uint64, memory *memory.Memory, insn, opcode, fun uint64, memTracker MemTracker, stackTracker StackTracker) error {
	// j-type j/jal
	if opcode == 2 || opcode == 3 {
		linkReg := uint64(0)
		if opcode == 3 {
			linkReg = 31
		}
		// Take top 4 bits of the next PC (its 256 MB region), and concatenate with the 26-bit offset
		target := (cpu.NextPC & 0xFFFFFFFFF0000000) | ((uint64(insn) & 0x03FFFFFF) << 2)
		stackTracker.PushStack(cpu.PC, target)
		return HandleJump(cpu, registers, linkReg, target)
	}

	// register fetch
	rs := uint64(0) // source register 1 value
	rt := uint64(0) // source register 2 / temp value
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
	} else if opcode >= 0x28 || opcode == 0x22 || opcode == 0x26 || opcode == 0x1A || opcode == 0x1B {
		// store rt value with store
		rt = registers[rtReg]

		// store actual rt with lwl, lwr, ldl, and ldr
		rdReg = rtReg
	}

	if (opcode >= 4 && opcode < 8) || opcode == 1 {
		return HandleBranch(cpu, registers, opcode, insn, rtReg, rs)
	}

	storeAddr := uint64(0xFF_FF_FF_FF_FF_FF_FF_FF)
	// memory fetch (all I-type)
	// we do the load for stores also
	mem := uint64(0)
	if opcode >= 0x20 {
		// M[R[rs]+SignExtImm]
		rs += SignExtend(insn&0xFFFF, 16)
		addr := rs & 0xFFFFFFFFFFFFFFF8
		memTracker.TrackMemAccess(addr)
		mem = memory.GetDoubleWord(addr)
		if opcode >= 0x28 && opcode != 0x30 && opcode != 0x34 && opcode != 0x37 {
			// store
			storeAddr = addr
			// store opcodes don't write back to a register
			rdReg = 0
		}
	}

	// ALU
	val := ExecuteMipsInstruction(insn, opcode, fun, rs, rt, mem)

	if opcode == 0 && fun >= 8 && fun < 0x20 {
		if fun == 8 || fun == 9 { // jr/jalr
			linkReg := uint64(0)
			if fun == 9 {
				linkReg = rdReg
			}
			stackTracker.PopStack()
			return HandleJump(cpu, registers, linkReg, rs)
		}

		if fun == 0xa { // movz
			return HandleRd(cpu, registers, rdReg, rs, rt == 0)
		}
		if fun == 0xb { // movn
			return HandleRd(cpu, registers, rdReg, rs, rt != 0)
		}

		// lo and hi registers
		// can write back
		if fun >= 0x10 && fun < 0x20 {
			return HandleHiLo(cpu, registers, fun, rs, rt, rdReg)
		}
	}

	// store conditional, write a 1 to rt
	if (opcode == 0x38 || opcode == 0x3C) && rtReg != 0 {
		registers[rtReg] = 1
	}

	// write memory
	if storeAddr != 0xFF_FF_FF_FF_FF_FF_FF_FF {
		memTracker.TrackMemAccess(storeAddr)
		memory.SetDoubleWord(storeAddr, val)
	}

	// write back the value to destination register
	return HandleRd(cpu, registers, rdReg, val, true)
}

func ExecuteMipsInstruction(insn, opcode, fun, rs, rt, mem uint64) uint64 {
	if opcode == 0 || (opcode >= 8 && opcode < 0xF) || opcode == 0x18 || opcode == 0x19 {
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
		case 0x18:
			fun = 0x2c // daddi
		case 0x19:
			fun = 0x2d // daddiu
		}

		switch fun {
		case 0x00: // sll
			return SignExtend((rt&0xFFFFFFFF)<<((insn>>6)&0x1F), 32)
		case 0x02: // srl
			return SignExtend((rt&0xFFFFFFFF)>>((insn>>6)&0x1F), 32)
		case 0x03: // sra
			shamt := (insn >> 6) & 0x1F
			return SignExtend((rt&0xFFFFFFFF)>>shamt, 32-shamt)
		case 0x04: // sllv
			return SignExtend((rt&0xFFFFFFFF)<<(rs&0x1F), 32)
		case 0x06: // srlv
			return SignExtend((rt&0xFFFFFFFF)>>(rs&0x1F), 32)
		case 0x07: // srav
			return SignExtend((rt&0xFFFFFFFF)>>rs, 32-rs)
		// MIPS32 functs in range [0x8, 0x1f] are handled specially by other functions
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
		case 0x14: // dsllv
			return rt
		case 0x16: // dsrlv
			return rt
		case 0x17: // dsrav
			return rt
		case 0x18: // mult
			return rs
		case 0x19: // multu
			return rs
		case 0x1a: // div
			return rs
		case 0x1b: // divu
			return rs
		case 0x1C: // dmult
			return rs
		case 0x1D: // dmultu
			return rs
		case 0x1E: // ddiv
			return rs
		case 0x1F: // ddivu
			return rs
		// The rest includes transformed R-type arith imm instructions
		case 0x20: // add
			return SignExtend(uint64(int32(rs)+int32(rt)), 32)
		case 0x21: // addu
			return SignExtend(uint64(uint32(rs)+uint32(rt)), 32)
		case 0x22: // sub
			return SignExtend(uint64(int32(rs)-int32(rt)), 32)
		case 0x23: // subu
			return SignExtend(uint64(uint32(rs)-uint32(rt)), 32)
		case 0x24: // and
			return rs & rt
		case 0x25: // or
			return rs | rt
		case 0x26: // xor
			return rs ^ rt
		case 0x27: // nor
			return ^(rs | rt)
		case 0x2a: // slti
			if int64(rs) < int64(rt) {
				return 1
			}
			return 0
		case 0x2b: // sltiu
			if rs < rt {
				return 1
			}
			return 0
		case 0x2c: // dadd
			return rs + rt
		case 0x2d: // daddu
			return rs + rt
		case 0x2e: // dsub
			return rs - rt
		case 0x2f: // dsubu
			return rs - rt
		case 0x38: // dsll
			return rt << ((insn >> 6) & 0x1f)
		case 0x3A: // dsrl
			return rt >> ((insn >> 6) & 0x1f)
		case 0x3B: // dsra
			return uint64(int64(rt) >> ((insn >> 6) & 0x1f))
		case 0x3C: // dsll32
			return rt << (((insn >> 6) & 0x1f) + 32)
		case 0x3E: // dsll32
			return rt >> (((insn >> 6) & 0x1f) + 32)
		case 0x3F: // dsll32
			return uint64(int64(rt) >> (((insn >> 6) & 0x1f) + 32))
		default:
			panic(fmt.Sprintf("invalid instruction: %x", insn))
		}
	} else {
		switch opcode {
		// SPECIAL2
		case 0x1C:
			switch fun {
			case 0x2: // mul
				return SignExtend(uint64(uint32(int32(rs)*int32(rt))), 32)
			case 0x20, 0x21: // clz, clo
				if fun == 0x20 {
					rs = ^rs
				}
				i := uint32(0)
				for ; rs&0x80000000 != 0; i++ {
					rs <<= 1
				}
				return uint64(i)
			}
		case 0x0F: // lui
			return SignExtend(rt<<16, 32)
		case 0x20: // lb
			return SignExtend((mem>>(56-(rs&7)*8))&0xFF, 8)
		case 0x21: // lh
			return SignExtend((mem>>(48-(rs&6)*8))&0xFFFF, 16)
		case 0x22: // lwl
			val := mem << ((rs & 3) * 8)
			mask := uint64(uint32(0xFFFFFFFF) << ((rs & 3) * 8))
			return SignExtend((rt & ^mask)|val, 32)
		case 0x23: // lw
			return SignExtend((mem>>(32-((rs&0x4)<<3)))&0xFFFFFFFF, 32)
		case 0x24: // lbu
			return (mem >> (56 - (rs&7)*8)) & 0xFF
		case 0x25: //  lhu
			return (mem >> (48 - (rs&6)*8)) & 0xFFFF
		case 0x26: //  lwr
			val := mem >> (24 - (rs&3)*8)
			mask := uint64(uint32(0xFFFFFFFF) >> (24 - (rs&3)*8))
			return SignExtend((rt & ^mask)|val, 32)
		case 0x28: //  sb
			val := (rt & 0xFF) << (56 - (rs&7)*8)
			mask := 0xFFFFFFFFFFFFFFFF ^ uint64(0xFF<<(56-(rs&7)*8))
			return (mem & mask) | val
		case 0x29: //  sh
			sl := 48 - ((rs & 0x6) << 3)
			val := (rt & 0xFFFF) << sl
			mask := 0xFFFFFFFFFFFFFFFF ^ uint64(0xFFFF<<sl)
			return (mem & mask) | val
		case 0x2a: //  swl
			sr := (rs & 3) << 3
			val := ((rt & 0xFFFFFFFF) >> sr) << (32 - ((rs & 0x4) << 3))
			mask := (uint64(0xFFFFFFFF) >> sr) << (32 - ((rs & 0x4) << 3))
			return (mem & ^mask) | val
		case 0x2b: //  sw
			sl := 32 - ((rs & 0x4) << 3)
			val := (rt & 0xFFFFFFFF) << sl
			mask := 0xFFFFFFFFFFFFFFFF ^ uint64(0xFFFFFFFF<<sl)
			return (mem & mask) | val
		case 0x2e: //  swr
			sl := 24 - ((rs & 3) << 3)
			val := ((rt & 0xFFFFFFFF) << sl) << (32 - ((rs & 0x4) << 3))
			mask := uint64(uint32(0xFFFFFFFF)<<sl) << (32 - ((rs & 0x4) << 3))
			return (mem & ^mask) | val
		case 0x30: //  ll
			return SignExtend((mem>>(32-((rs&0x4)<<3)))&0xFFFFFFFF, 32)
		case 0x38: //  sc
			return rt
		// MIPS64
		case 0x1A: // ldl
			sl := (rs & 0x7) << 3
			val := mem << sl
			mask := uint64(0xFFFFFFFFFFFFFFFF) << sl
			return val | (rt & ^mask)
		case 0x1B: // ldr
			sr := 56 - ((rs & 0x7) << 3)
			val := mem >> sr
			mask := uint64(0xFFFFFFFFFFFFFFFF) << (64 - sr)
			return val | (rt & mask)
		case 0x27: // lwu
			return (mem >> (32 - ((rs & 0x4) << 3))) & 0xFFFFFFFF
		case 0x2C: // sdl
			sr := (rs & 0x7) << 3
			val := rt >> sr
			mask := uint64(0xFFFFFFFFFFFFFFFF) >> sr
			return val | (mem & ^mask)
		case 0x2D: // sdr
			sl := 56 - ((rs & 0x7) << 3)
			val := rt << sl
			mask := uint64(0xFFFFFFFFFFFFFFFF) << sl
			return val | (mem & ^mask)
		case 0x34: // lld
			return mem
		case 0x37: // ld
			return mem
		case 0x3C: // scd
			return rt
		case 0x3F: // sd
			return rt
		default:
			panic(fmt.Sprintf("invalid instruction: %x", insn))
		}
	}
	panic(fmt.Sprintf("invalid instruction: %x", insn))
}

func SignExtend(dat uint64, idx uint64) uint64 {
	isSigned := (dat >> (idx - 1)) != 0
	signed := ((uint64(1) << (64 - idx)) - 1) << idx
	mask := (uint64(1) << idx) - 1
	if isSigned {
		return dat&mask | signed
	} else {
		return dat & mask
	}
}

func HandleBranch(cpu *mipsevm.CpuScalars, registers *[32]uint64, opcode, insn, rtReg, rs uint64) error {
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

func HandleHiLo(cpu *mipsevm.CpuScalars, registers *[32]uint64, fun, rs, rt, storeReg uint64) error {
	val := uint64(0)
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
		cpu.HI = SignExtend(acc>>32, 32)
		cpu.LO = SignExtend(uint64(uint32(acc)), 32)
	case 0x19: // multu
		acc := uint64(uint32(rs)) * uint64(uint32(rt))
		cpu.HI = SignExtend(acc>>32, 32)
		cpu.LO = SignExtend(uint64(uint32(acc)), 32)
	case 0x1a: // div
		cpu.HI = SignExtend(uint64(int32(rs)%int32(rt)), 32)
		cpu.LO = SignExtend(uint64(int32(rs)/int32(rt)), 32)
	case 0x1b: // divu
		cpu.HI = SignExtend(uint64(uint32(rs)%uint32(rt)), 32)
		cpu.LO = SignExtend(uint64(uint32(rs)/uint32(rt)), 32)
	case 0x14: // dsllv
		val = rt << (rs & 0x3F)
	case 0x16: // dsrlv
		val = rt >> (rs & 0x3F)
	case 0x17: // dsrav
		val = uint64(int64(rt) >> (rs & 0x3F))
	case 0x1c: // dmult
		// TODO: Signed mult for dmult?
		acc := u128.From64(rs).Mul(u128.From64(rt))
		cpu.HI = acc.Hi
		cpu.LO = acc.Lo
	case 0x1d: // dmultu
		acc := u128.From64(rs).Mul(u128.From64(rt))
		cpu.HI = acc.Hi
		cpu.LO = acc.Lo
	case 0x1e: // ddiv
		cpu.HI = uint64(int64(rs) % int64(rt))
		cpu.LO = uint64(int64(rs) / int64(rt))
	case 0x1f: // ddivu
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

func HandleJump(cpu *mipsevm.CpuScalars, registers *[32]uint64, linkReg uint64, dest uint64) error {
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

func HandleRd(cpu *mipsevm.CpuScalars, registers *[32]uint64, storeReg uint64, val uint64, conditional bool) error {
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
