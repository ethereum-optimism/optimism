package main

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// go test -run TestTrie

func TestTrie(t *testing.T) {
	ram := make(map[uint32](uint32))
	LoadMappedFile("../mipigo/test/test.bin", ram, 0)
	ZeroRegisters(ram)
	root := RamToTrie(ram)
	//ParseNode(root, 0)

	dat := SerializeTrie(root)
	fmt.Println("serialized length is", len(dat))
	ioutil.WriteFile("/tmp/eth/ramtrie", dat, 0644)
}

func TestBuggedTrie(t *testing.T) {
	ram := make(map[uint32](uint32))

	ram[0] = 1
	ram[4] = 2

	root := RamToTrie(ram)
	fmt.Println("root(0,4) =", root)
	ParseNode(root, 0, func(t common.Hash) []byte {
		return Preimages[t]
	})

	ram[0x40] = 3

	root = RamToTrie(ram)
	fmt.Println("root(0,4,0x40) =", root)
	ParseNode(root, 0, func(t common.Hash) []byte {
		return Preimages[t]
	})
}
