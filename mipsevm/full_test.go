package main

import (
	"fmt"
	"testing"
)

func TestFull(t *testing.T) {
	RunFull()
}

func TestFullEvm(t *testing.T) {
	ram := make(map[uint32](uint32))
	LoadMappedFile("test/bin/add.bin", ram, 0)
	ZeroRegisters(ram)
	ram[0xC000007C] = 0x5EAD0000

	for step := 0; step < 12; step++ {
		RunWithRam(ram, 1, 0, nil)
		root := RamToTrie(ram)
		fmt.Println(step, root)
	}
}
