package exec

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
	u128 "lukechampine.com/uint128"
)

const (
	OpLoadLinked       = 0x30
	OpStoreConditional = 0x38
)

func GetInstructionDetails(pc Word, memory *memory.Memory) (insn, opcode, fun uint32) {
	insn = memory.GetMemory(pc)
	opcode = insn >> 26 // First 6-bits
	fun = insn & 0x3f   // Last 6-bits

	return insn, opcode, fun
}

func ExecMipsCoreStepLogic(cpu *mipsevm.CpuScalars, registers *[32]Word, memory *memory.Memory, insn, opcode, fun uint32, memTracker MemTracker, stackTracker StackTracker) (memUpdated bool, memAddr Word, err error) {
	// j-type j/jal
	if opcode == 2 || opcode == 3 {
		linkReg := Word(0)
		if opcode == 3 {
			linkReg = 31
		}
		// Take top 4 bits of the next PC (its 256 MB region), and concatenate with the 26-bit offset
		target := (cpu.NextPC & SignExtend(0xF0000000, 32)) | Word((insn & 0x03FFFFFF) << 2)
		stackTracker.PushStack(cpu.PC, target)
		err = HandleJump(cpu, registers, linkReg, target)
		return
	}

	// register fetch
	rs := Word(0) // source register 1 value
	rt := Word(0) // source register 2 / temp value
	rtReg := Word((insn >> 16) & 0x1F)

	// R-type or I-type (stores rt)
	rs = registers[(insn>>21)&0x1F]
	rdReg := rtReg
	if opcode == 0 || opcode == 0x1c {
		// R-type (stores rd)
		rt = registers[rtReg]
		rdReg = Word((insn >> 11) & 0x1F)
	} else if opcode < 0x20 {
		// rt is SignExtImm
		// don't sign extend for andi, ori, xori
		if opcode == 0xC || opcode == 0xD || opcode == 0xe {
			// ZeroExtImm
			rt = Word(insn & 0xFFFF)
		} else {
			// SignExtImm
			rt = SignExtendImmediate(insn)
		}
	} else if opcode >= 0x28 || opcode == 0x22 || opcode == 0x26 {
		// store rt value with store
		rt = registers[rtReg]

		// store actual rt with lwl and lwr
		rdReg = rtReg
	} else if opcode == 0x27 || opcode == 0x1A || opcode == 0x1B { // 64-bit opcodes
		// store rt value with store
		rt = registers[rtReg]
		// store actual rt with ldl and ldr
		rdReg = rtReg
	}

	if (opcode >= 4 && opcode < 8) || opcode == 1 {
		err = HandleBranch(cpu, registers, opcode, insn, rtReg, rs)
		return
	}

	storeAddr := ^Word(0)
	// memory fetch (all I-type)
	// we do the load for stores also
	mem := Word(0)
	if opcode >= 0x20 {
		// M[R[rs]+SignExtImm]
		rs += SignExtendImmediate(insn)
		addr := rs & arch.AddressMask
		memTracker.TrackMemAccess(addr)
		mem = memory.GetWord(addr)
		if opcode >= 0x28 {
			// store for 32-bit and certain 64-bit opcodes
			if arch.IsMips32 || opcode != 0x30 && opcode != 0x34 && opcode != 0x37 {
				// store
				storeAddr = addr
				// store opcodes don't write back to a register
				rdReg = 0
			}
		}
	}

	// ALU
	val := ExecuteMipsInstruction(insn, opcode, fun, rs, rt, mem)

	funSel := uint32(0x1c)
	if !arch.IsMips32 {
		funSel = 0x20
	}
	if opcode == 0 && fun >= 8 && fun < funSel {
		if fun == 8 || fun == 9 { // jr/jalr
			linkReg := Word(0)
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
		if fun >= 0x10 && fun < funSel {
			err = HandleHiLo(cpu, registers, fun, rs, rt, rdReg)
			return
		}
	}
	// TODO: 64-bit ll/sc

	// write memory
	if storeAddr != ^Word(0) {
		memTracker.TrackMemAccess(storeAddr)
		memory.SetWord(storeAddr, val)
		memUpdated = true
		memAddr = storeAddr
	}

	// write back the value to destination register
	err = HandleRd(cpu, registers, rdReg, val, true)
	return
}

func SignExtendImmediate(insn uint32) Word {
	return SignExtend(Word(insn&0xFFFF), 16)
}

func ExecuteMipsInstruction(insn uint32, opcode uint32, fun uint32, rs, rt, mem Word) Word {
	assertMips64 := func() {
		if arch.IsMips32 {
			panic(fmt.Sprintf("invalid instruction: %x", insn))
		}
	}

	if opcode == 0 || (opcode >= 8 && opcode < 0xF) || (!arch.IsMips32 && (opcode == 0x18 || opcode == 0x19)) {
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
			shamt := Word((insn >> 6) & 0x1F)
			return SignExtend((rt&0xFFFFFFFF)>>shamt, 32-shamt)
		case 0x04: // sllv
			return SignExtend((rt&0xFFFFFFFF)<<(rs&0x1F), 32)
		case 0x06: // srlv
			return SignExtend((rt&0xFFFFFFFF)>>(rs&0x1F), 32)
		case 0x07: // srav
			shamt := Word(rs & 0x1F)
			return SignExtend((rt&0xFFFFFFFF)>>shamt, 32-shamt)
		// functs in range [0x8, 0x1b] for 32-bit and [0x8, 0x1f] for 64-bit are handled specially by other functions
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
			assertMips64()
			return rt
		case 0x16: // dsrlv
			assertMips64()
			return rt
		case 0x17: // dsrav
			assertMips64()
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
			assertMips64()
			return rs
		case 0x1D: // dmultu
			assertMips64()
			return rs
		case 0x1E: // ddiv
			assertMips64()
			return rs
		case 0x1F: // ddivu
			assertMips64()
			return rs
		// The rest includes transformed R-type arith imm instructions
		case 0x20: // add
			return SignExtend(Word(int32(rs)+int32(rt)), 32)
		case 0x21: // addu
			return SignExtend(Word(uint32(rs)+uint32(rt)), 32)
		case 0x22: // sub
			return SignExtend(Word(int32(rs)-int32(rt)), 32)
		case 0x23: // subu
			return SignExtend(Word(uint32(rs)-uint32(rt)), 32)
		case 0x24: // and
			return rs & rt
		case 0x25: // or
			return rs | rt
		case 0x26: // xor
			return rs ^ rt
		case 0x27: // nor
			return ^(rs | rt)
		case 0x2a: // slti
			if arch.SignedInteger(rs) < arch.SignedInteger(rt) {
				return 1
			}
			return 0
		case 0x2b: // sltiu
			if rs < rt {
				return 1
			}
			return 0
		case 0x2c: // dadd
			assertMips64()
			return rs + rt
		case 0x2d: // daddu
			assertMips64()
			return rs + rt
		case 0x2e: // dsub
			assertMips64()
			return rs - rt
		case 0x2f: // dsubu
			assertMips64()
			return rs - rt
		case 0x38: // dsll
			assertMips64()
			return rt << ((insn >> 6) & 0x1f)
		case 0x3A: // dsrl
			assertMips64()
			return rt >> ((insn >> 6) & 0x1f)
		case 0x3B: // dsra
			assertMips64()
			return Word(int64(rt) >> ((insn >> 6) & 0x1f))
		case 0x3C: // dsll32
			assertMips64()
			return rt << (((insn >> 6) & 0x1f) + 32)
		case 0x3E: // dsll32
			assertMips64()
			return rt >> (((insn >> 6) & 0x1f) + 32)
		case 0x3F: // dsll32
			assertMips64()
			return Word(int64(rt) >> (((insn >> 6) & 0x1f) + 32))
		default:
			panic(fmt.Sprintf("invalid instruction: %x", insn))
		}
	} else {
		switch opcode {
		// SPECIAL2
		case 0x1C:
			switch fun {
			case 0x2: // mul
				return SignExtend(Word(int32(rs)*int32(rt)), 32)
			case 0x20, 0x21: // clz, clo
				if fun == 0x20 {
					rs = ^rs
				}
				i := uint32(0)
				for ; rs&0x80000000 != 0; i++ {
					rs <<= 1
				}
				return Word(i)
			}
		case 0x0F: // lui
			return SignExtend(rt<<16, 32)
		case 0x20: // lb
			msb := uint32(arch.WordSize - 8) // 24 for 32-bit and 56 for 64-bit
			return SignExtend((mem>>(msb-uint32(rs&arch.ExtMask)*8))&0xFF, 8)
		case 0x21: // lh
			msb := uint32(arch.WordSize - 16) // 16 for 32-bit and 48 for 64-bit
			mask := Word(arch.ExtMask - 1)
			return SignExtend((mem>>(msb-uint32(rs&mask)*8))&0xFFFF, 16)
		case 0x22: // lwl
			val := mem << ((rs & 3) * 8)
			mask := Word(uint32(0xFFFFFFFF) << ((rs & 3) * 8))
			return SignExtend(((rt & ^mask)|val)&0xFFFFFFFF, 32)
		case 0x23: // lw
			// TODO
			return mem
			//return SignExtend((mem>>(32-((rs&0x4)<<3)))&0xFFFFFFFF, 32)
		case 0x24: // lbu
			msb := uint32(arch.WordSize - 8) // 24 for 32-bit and 56 for 64-bit
			return (mem >> (msb - uint32(rs&arch.ExtMask)*8)) & 0xFF
		case 0x25: //  lhu
			msb := uint32(arch.WordSize - 16) // 16 for 32-bit and 48 for 64-bit
			mask := Word(arch.ExtMask - 1)
			return (mem >> (msb - uint32(rs&mask)*8)) & 0xFFFF
		case 0x26: //  lwr
			val := mem >> (24 - (rs&3)*8)
			mask := Word(uint32(0xFFFFFFFF) >> (24 - (rs&3)*8))
			return SignExtend(((rt & ^mask)|val)&0xFFFFFFFF, 32)
		case 0x28: //  sb
			msb := uint32(arch.WordSize - 8) // 24 for 32-bit and 56 for 64-bit
			val := (rt & 0xFF) << (msb - uint32(rs&arch.ExtMask)*8)
			mask := ^Word(0) ^ Word(0xFF<<(msb-uint32(rs&arch.ExtMask)*8))
			return (mem & mask) | val
		case 0x29: //  sh
			msb := uint32(arch.WordSize - 16) // 16 for 32-bit and 48 for 64-bit
			rsMask := Word(arch.ExtMask - 1)  // 2 for 32-bit and 6 for 64-bit
			sl := msb - uint32(rs&rsMask)*8
			val := (rt & 0xFFFF) << sl
			mask := ^Word(0) ^ Word(0xFFFF<<sl)
			return (mem & mask) | val
		case 0x2a: //  swl
			// TODO
			val := rt >> ((rs & 3) * 8)
			mask := uint32(0xFFFFFFFF) >> ((rs & 3) * 8)
			return (mem & Word(^mask)) | val
		case 0x2b: //  sw
			// TODO
			return rt
		case 0x2e: //  swr
			// TODO
			val := rt << (24 - (rs&3)*8)
			mask := uint32(0xFFFFFFFF) << (24 - (rs&3)*8)
			return (mem & Word(^mask)) | val

		// MIPS64
		case 0x1A: // ldl
			assertMips64()
			sl := (rs & 0x7) << 3
			val := mem << sl
			mask := ^Word(0) << sl
			return val | (rt & ^mask)
		case 0x1B: // ldr
			assertMips64()
			sr := 56 - ((rs & 0x7) << 3)
			val := mem >> sr
			mask := ^Word(0) << (64 - sr)
			return val | (rt & mask)
		case 0x27: // lwu
			assertMips64()
			return (mem >> (32 - ((rs & 0x4) << 3))) & 0xFFFFFFFF
		case 0x2C: // sdl
			assertMips64()
			sr := (rs & 0x7) << 3
			val := rt >> sr
			mask := ^Word(0) >> sr
			return val | (mem & ^mask)
		case 0x2D: // sdr
			assertMips64()
			sl := 56 - ((rs & 0x7) << 3)
			val := rt << sl
			mask := ^Word(0) << sl
			return val | (mem & ^mask)
		case 0x34: // lld
			assertMips64()
			return mem
		case 0x37: // ld
			assertMips64()
			return mem
		case 0x3C: // scd
			assertMips64()
			sl := (rs & 0x7) << 3
			val := rt << sl
			mask := ^Word(0) << sl
			return (mem & ^mask) | val
		case 0x3F: // sd
			assertMips64()
			sl := (rs & 0x7) << 3
			val := rt << sl
			mask := ^Word(0) << sl
			return (mem & ^mask) | val
		default:
			panic("invalid instruction")
		}
	}
	panic("invalid instruction")
}

// TODO: consider moving this to the arch package as a generic function to avoid frequent Word casts
func SignExtend(dat Word, idx Word) Word {
	isSigned := (dat >> (idx - 1)) != 0
	signed := ((Word(1) << (arch.WordSize - idx)) - 1) << idx
	mask := (Word(1) << idx) - 1
	if isSigned {
		return dat&mask | signed
	} else {
		return dat & mask
	}
}

func HandleBranch(cpu *mipsevm.CpuScalars, registers *[32]Word, opcode uint32, insn uint32, rtReg Word, rs Word) error {
	if cpu.NextPC != cpu.PC+4 {
		panic("branch in delay slot")
	}

	shouldBranch := false
	if opcode == 4 || opcode == 5 { // beq/bne
		rt := registers[rtReg]
		shouldBranch = (rs == rt && opcode == 4) || (rs != rt && opcode == 5)
	} else if opcode == 6 {
		shouldBranch = arch.SignedInteger(rs) <= 0 // blez
	} else if opcode == 7 {
		shouldBranch = arch.SignedInteger(rs) > 0 // bgtz
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
		cpu.NextPC = prevPC + 4 + (SignExtend(Word(insn&0xFFFF), 16) << 2) // then continue with the instruction the branch jumps to.
	} else {
		cpu.NextPC = cpu.NextPC + 4 // branch not taken
	}
	return nil
}

func HandleHiLo(cpu *mipsevm.CpuScalars, registers *[32]Word, fun uint32, rs Word, rt Word, storeReg Word) error {
	val := Word(0)
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
		cpu.HI = SignExtend(Word(acc>>32), 32)
		cpu.LO = SignExtend(Word(uint32(acc)), 32)
	case 0x19: // multu
		acc := uint64(uint32(rs)) * uint64(uint32(rt))
		cpu.HI = SignExtend(Word(acc>>32), 32)
		cpu.LO = SignExtend(Word(uint32(acc)), 32)
	case 0x1a: // div
		cpu.HI = SignExtend(Word(int32(rs)%int32(rt)), 32)
		cpu.LO = SignExtend(Word(int32(rs)/int32(rt)), 32)
	case 0x1b: // divu
		cpu.HI = SignExtend(Word(uint32(rs)%uint32(rt)), 32)
		cpu.LO = SignExtend(Word(uint32(rs)/uint32(rt)), 32)
	case 0x14: // dsllv
		val = rt << (rs & 0x3F)
	case 0x16: // dsrlv
		val = rt >> (rs & 0x3F)
	case 0x17: // dsrav
		val = Word(int64(rt) >> (rs & 0x3F))
	case 0x1c: // dmult
		// TODO: Signed mult for dmult?
		acc := u128.From64(uint64(rs)).Mul(u128.From64(uint64(rt)))
		cpu.HI = Word(acc.Hi)
		cpu.LO = Word(acc.Lo)
	case 0x1d: // dmultu
		acc := u128.From64(uint64(rs)).Mul(u128.From64(uint64(rt)))
		cpu.HI = Word(acc.Hi)
		cpu.LO = Word(acc.Lo)
	case 0x1e: // ddiv
		cpu.HI = Word(int64(rs) % int64(rt))
		cpu.LO = Word(int64(rs) / int64(rt))
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

func HandleJump(cpu *mipsevm.CpuScalars, registers *[32]Word, linkReg Word, dest Word) error {
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

func HandleRd(cpu *mipsevm.CpuScalars, registers *[32]Word, storeReg Word, val Word, conditional bool) error {
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
