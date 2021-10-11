package main

import (
	"fmt"
	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
	"log"
	"sync"
	"testing"
	"time"
)

func RegSerialize(ram map[uint32](uint32)) []uint32 {
	ret := []uint32{ram[0xc0000080], ram[0xc00000A0], uint32(len(ram))}
	for i := uint32(0xc0000000); i < 0xc0000000+37*4; i += 4 {
		ret = append(ret, ram[i])
	}
	return ret
}

var done sync.Mutex

func TestCompare(t *testing.T) {
	fn := "../mipigeth/test.bin"
	//fn := "test/bin/sc.bin"
	//fn := "../mipigeth/minigeth.bin"

	steps := 10000000
	//steps := 1165
	//steps := 1180

	cevm := make(chan []uint32, 1)
	cuni := make(chan []uint32, 1)

	ram := make(map[uint32](uint32))
	LoadMappedFile(fn, ram, 0)
	inputFile := fmt.Sprintf("/tmp/eth/%d", 13284469)
	LoadMappedFile(inputFile, ram, 0x30000000)

	go RunWithRam(ram, steps, 0, func(step int, ram map[uint32](uint32)) {
		//fmt.Printf("%d evm %x\n", step, ram[0xc0000080])
		cevm <- RegSerialize(ram)
		done.Lock()
		done.Unlock()
	})

	uniram := make(map[uint32](uint32))
	go RunUnicorn(fn, uniram, steps, func(step int, mu uc.Unicorn, ram map[uint32](uint32)) {
		SyncRegs(mu, ram)
		cuni <- RegSerialize(ram)
		done.Lock()
		done.Unlock()
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
				fmt.Println(i, "mismatch at", j, "cevm", x, "cuni", y)
				break
			}
		}
	}

	// final ram check
	done.Lock()
	time.Sleep(100 * time.Millisecond)
	mismatch := false
	for k, v := range ram {
		if uniram[k] != v {
			fmt.Printf("ram mismatch at %x, evm %x != uni %x\n", k, v, uniram[k])
			mismatch = true
		}
	}
	if mismatch {
		log.Fatal("RAM mismatch")
	}
}
