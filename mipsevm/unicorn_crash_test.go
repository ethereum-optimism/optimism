package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"testing"

	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

func TestUnicornCrash(t *testing.T) {
	root := "/tmp/cannon/0_13284469"
	mu, err := uc.NewUnicorn(uc.ARCH_MIPS, uc.MODE_32|uc.MODE_BIG_ENDIAN)
	check(err)

	// program
	fn := "../mipigo/test/test.bin"
	dat, err := ioutil.ReadFile(fn)
	check(err)
	mu.MemWrite(0, dat)

	// inputs
	inputs, _ := ioutil.ReadFile(fmt.Sprintf("%s/input", root))
	mu.MemWrite(0x30000000, inputs[0:0xc0])

	// load into ram, without this (completely unrelated), it doesn't crash
	ram := make(map[uint32](uint32))
	for i := 0; i < len(dat); i += 4 {
		value := binary.BigEndian.Uint32(dat[i : i+4])
		if value != 0 {
			ram[uint32(i)] = value
		}
	}

	mu.Start(0, 0x5ead0004)
}
