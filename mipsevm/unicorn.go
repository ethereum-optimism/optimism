package main

import (
	"fmt"
	"io/ioutil"
	"log"

	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var steps int = 0

func RunUnicorn(fn string) {
	mu, err := uc.NewUnicorn(uc.ARCH_MIPS, uc.MODE_32|uc.MODE_BIG_ENDIAN)
	check(err)

	mu.HookAdd(uc.HOOK_INTR, func(mu uc.Unicorn, intno uint32) {
		if intno != 17 {
			log.Fatal("invalid interrupt ", intno)
		}
		syscall_no, _ := mu.RegRead(uc.MIPS_REG_V0)
		fmt.Println("syscall", syscall_no)
	}, 1, 0)

	mu.HookAdd(uc.HOOK_CODE, func(mu uc.Unicorn, addr uint64, size uint32) {
		if steps%10000 == 0 {
			fmt.Printf("%6d Code: 0x%x, 0x%x\n", steps, addr, size)
		}
		steps += 1
	}, 1, 0)

	check(mu.MemMap(0, 0x80000000))

	dat, _ := ioutil.ReadFile(fn)
	mu.MemWrite(0, dat)

	mu.Start(0, 0xdead0000)

}
