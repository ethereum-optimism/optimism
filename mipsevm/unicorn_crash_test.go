package main

import (
	"encoding/binary"
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
	dat := make([]byte, 1490944)
	for i := 0; i < len(dat); i++ {
		dat[i] = 0xaa
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
