package main

import (
	"fmt"
	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
	"testing"
)

func WriteRam(ram map[uint32](uint32), addr uint32, value uint32) {
	if value != 0 {
		ram[addr] = value
	} else {
		delete(ram, addr)
	}
}

func RegSerialize(ram map[uint32](uint32)) []uint32 {
	// len(ram) is still failing, need nextpc
	//ret := []uint32{ram[0xc0000080] & 0x7FFFFFFF, uint32(len(ram))}
	ret := []uint32{ram[0xc0000080] & 0x7FFFFFFF}
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
