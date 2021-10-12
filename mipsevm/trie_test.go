package main

import (
	"testing"
)

// go test -run TestTrie

func TestTrie(t *testing.T) {
	fn := "../mipigo/minigeth.bin"
	ram := make(map[uint32](uint32))
	LoadMappedFile(fn, ram, 0)

	RamToTrie(ram)
}
