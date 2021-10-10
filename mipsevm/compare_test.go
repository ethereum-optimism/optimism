package main

import (
	"fmt"
	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
	"testing"
)

var REG_OFFSET uint32 = 0xc0000000
var REG_PC uint32 = REG_OFFSET + 0x20*4
var REG_PENDPC uint32 = REG_OFFSET + 0x24*4
var PC_PEND uint32 = 0x80000000
var PC_MASK uint32 = 0x7FFFFFFF

func WriteRam(ram map[uint32](uint32), addr uint32, value uint32) {
	if value != 0 {
		ram[addr] = value
	} else {
		delete(ram, addr)
	}
}

func SE(dat uint32, idx uint32) uint32 {
	isSigned := (dat >> (idx - 1)) != 0
	signed := ((1 << (32 - idx)) - 1) << idx
	mask := (1 << idx) - 1
	ret := dat & uint32(mask)
	if isSigned {
		ret |= uint32(signed)
	}
	return ret
}

// UGH: is there a better way?
// I don't see a better way to get this out
func FixBranchDelay(ram map[uint32](uint32)) {
	pc := ram[REG_PC] & 0x7FFFFFFF
	insn := ram[pc-4]
	opcode := insn >> 26
	mfunc := insn & 0x3f
	//fmt.Println(opcode)

	if opcode == 2 || opcode == 3 {
		ram[REG_PENDPC] = SE(insn&0x03FFFFFF, 26) << 2
		ram[REG_PC] |= 0x80000000
		return
	}
	rs := ram[REG_OFFSET+((insn>>19)&0x7C)]
	if (opcode >= 4 && opcode < 8) || opcode == 1 {
		shouldBranch := false
		if opcode == 4 || opcode == 5 {
			rt := ram[REG_OFFSET+((insn>>14)&0x7C)]
			shouldBranch = (rs == rt && opcode == 4) || (rs != rt && opcode == 5)
		} else if opcode == 6 {
			shouldBranch = int32(rs) <= 0
		} else if opcode == 7 {
			shouldBranch = int32(rs) > 0
		} else if opcode == 1 {
			rtv := ((insn >> 16) & 0x1F)
			if rtv == 0 {
				shouldBranch = int32(rs) < 0
			} else if rtv == 1 {
				shouldBranch = int32(rs) >= 0
			}
		}
		ram[REG_PC] |= 0x80000000
		if shouldBranch {
			ram[REG_PENDPC] = pc + (SE(insn&0xFFFF, 16) << 2)
		} else {
			ram[REG_PENDPC] = pc + 4
		}
	}
	if opcode == 0 && (mfunc == 8 || mfunc == 9) {
		ram[REG_PC] |= 0x80000000
		ram[REG_PENDPC] = rs
	}
}

func RegSerialize(ram map[uint32](uint32)) []uint32 {
	// len(ram) is still failing, need nextpc
	//ret := []uint32{ram[0xc0000080] & 0x7FFFFFFF, uint32(len(ram))}
	//ret := []uint32{ram[0xc0000080] & 0x7FFFFFFF}
	//ret := []uint32{ram[0xc0000080], ram[REG_PENDPC], uint32(len(ram))}
	ret := []uint32{ram[0xc0000080], ram[REG_PENDPC]}
	for i := uint32(0xc0000000); i < 0xc0000000+32*4; i += 4 {
		ret = append(ret, ram[i])
	}
	return ret
}

func TestCompare(t *testing.T) {
	fn := "../mipigeth/test.bin"
	//fn := "../mipigeth/minigeth.bin"

	steps := 10000000

	cevm := make(chan []uint32)
	cuni := make(chan []uint32)

	ram := make(map[uint32](uint32))
	LoadMappedFile(fn, ram, 0)
	inputFile := fmt.Sprintf("/tmp/eth/%d", 13284469)
	LoadMappedFile(inputFile, ram, 0x30000000)

	go RunWithRam(ram, steps, 0, func(step int, ram map[uint32](uint32)) {
		//fmt.Printf("%d evm %x\n", step, ram[0xc0000080])
		cevm <- RegSerialize(ram)
	})

	go RunUnicorn(fn, steps, func(step int, mu uc.Unicorn, ram map[uint32](uint32)) {
		pc, _ := mu.RegRead(uc.MIPS_REG_PC)
		//fmt.Printf("%d uni %x\n", step, pc)

		addr := uint32(0xc0000000)
		for i := uc.MIPS_REG_ZERO; i < uc.MIPS_REG_ZERO+32; i++ {
			reg, _ := mu.RegRead(i)
			WriteRam(ram, addr, uint32(reg))
			addr += 4
		}
		WriteRam(ram, 0xc0000080, uint32(pc))
		FixBranchDelay(ram)
		cuni <- RegSerialize(ram)
	})

	for i := 0; i < steps; i++ {
		x, y := <-cevm, <-cuni
		if x[0] == 0x5ead0000 && y[0] == 0x5ead0000 {
			fmt.Println("both processes exited")
			break
		}
		if i%1000 == 0 {
			fmt.Println(i, x[0:9], y[0:9])
		}
		for j := 0; j < len(x); j++ {
			if x[j] != y[j] {
				fmt.Println(i, x, y)
				t.Fatal("value mismatch")
			}
		}
	}

}
