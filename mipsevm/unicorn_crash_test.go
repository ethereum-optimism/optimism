package main

import (
	"log"
	"testing"

	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

func TestUnicornCrash(t *testing.T) {
	mu, err := uc.NewUnicorn(uc.ARCH_MIPS, uc.MODE_32|uc.MODE_BIG_ENDIAN)
	if err != nil {
		log.Fatal(err)
	}

	// weird heap grooming (doesn't crash without this)
	junk := make(map[uint32](uint32))
	for i := 0; i < 1000000; i += 4 {
		junk[uint32(i)] = 0xaaaaaaaa
	}

	mu.Start(0, 4)
}
