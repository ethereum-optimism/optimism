package main

import (
	"fmt"
	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
	"testing"
)

func TestCompare(t *testing.T) {
	fn := "../mipigeth/test.bin"

	steps := 10000000

	cevm := make(chan []uint32)
	cuni := make(chan []uint32)

	ram := make(map[uint32](uint32))
	LoadMappedFile(fn, ram, 0)
	go RunWithRam(ram, steps, 0, func(step int, ram map[uint32](uint32)) {
		//fmt.Printf("%d evm %x\n", step, ram[0xc0000080])
		ret := []uint32{ram[0xc0000080] & 0x7FFFFFFF}
		for i := uint32(0xc0000000); i < 0xc0000000+32*4; i += 4 {
			ret = append(ret, ram[i])
		}
		cevm <- ret
	})

	go RunUnicorn(fn, steps, func(step int, mu uc.Unicorn) {
		pc, _ := mu.RegRead(uc.MIPS_REG_PC)
		//fmt.Printf("%d uni %x\n", step, pc)
		ret := []uint32{uint32(pc)}
		for i := uc.MIPS_REG_ZERO; i < uc.MIPS_REG_ZERO+32; i++ {
			reg, _ := mu.RegRead(i)
			ret = append(ret, uint32(reg))
		}
		cuni <- ret
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
