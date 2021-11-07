package main

import (
	"encoding/binary"
	"io/ioutil"
	"log"
	"testing"

	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

func TestUnicornCrash(t *testing.T) {
	mu, err := uc.NewUnicorn(uc.ARCH_MIPS, uc.MODE_32|uc.MODE_BIG_ENDIAN)
	if err != nil {
		log.Fatal(err)
	}

	// program
	fn := "../mipigo/test/test.bin"
	dat, err := ioutil.ReadFile(fn)
	if err != nil {
		log.Fatal(err)
	}
	mu.MemWrite(0, dat)

	// load into a map, without this (completely unrelated), it doesn't crash
	ram := make(map[uint32](uint32))
	for i := 0; i < len(dat); i += 4 {
		value := binary.BigEndian.Uint32(dat[i : i+4])
		if value != 0 {
			ram[uint32(i)] = value
		}
	}

	mu.Start(0, 0x5ead0004)
}
