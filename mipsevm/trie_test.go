package main

import (
	"fmt"
	"io/ioutil"
	"testing"
)

// go test -run TestTrie

func TestTrie(t *testing.T) {
	ram := make(map[uint32](uint32))
	LoadMappedFile("../mipigo/test/test.bin", ram, 0)
	ZeroRegisters(ram)
	root := RamToTrie(ram)
	ParseNode(root, 0)

	dat := SerializeTrie(root)
	fmt.Println("serialized length is", len(dat))
	ioutil.WriteFile("/tmp/eth/ramtrie", dat, 0644)
}
