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
		fmt.Printf("%d evm %x\n", step, ram[0xc0000080])
		cevm <- []uint32{ram[0xc0000080] & 0x7FFFFFFF, ram[0xc0000008]}
	})

	go RunUnicorn(fn, steps, func(step int, mu uc.Unicorn) {
		pc, _ := mu.RegRead(uc.MIPS_REG_PC)
		v0, _ := mu.RegRead(uc.MIPS_REG_V0)
		fmt.Printf("%d uni %x\n", step, pc)
		cuni <- []uint32{uint32(pc), uint32(v0)}
	})

	for i := 0; i < steps; i++ {
		x, y := <-cevm, <-cuni
		if x[0] == 0x5ead0000 && y[0] == 0x5ead0000 {
			fmt.Println("both processes exited")
			break
		}
		if i%1000 == 0 {
			fmt.Println(i, x, y)
		}
		for j := 0; j < len(x); j++ {
			if x[j] != y[j] {
				fmt.Println(i, x, y)
				t.Fatal("value mismatch")
			}
		}
	}

}
