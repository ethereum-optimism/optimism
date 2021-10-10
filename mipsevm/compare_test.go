package main

import (
	"fmt"
	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
	"testing"
)

func RegSerialize(ram map[uint32](uint32)) []uint32 {
	ret := []uint32{ram[0xc0000080], ram[0xc00000A0], uint32(len(ram))}
	for i := uint32(0xc0000000); i < 0xc0000000+37*4; i += 4 {
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
		SyncRegs(mu, ram)
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
				fmt.Println(i, "cevm", x, "cuni", y)
				t.Fatal("value mismatch")
			}
		}
	}

}
