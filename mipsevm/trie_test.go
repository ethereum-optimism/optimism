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

	dat := TrieToJson(root)
	fmt.Println("serialized length is", len(dat))
	ioutil.WriteFile("/tmp/cannon/ramtrie.json", dat, 0644)
}

func printRoot(ram map[uint32](uint32)) {
	root := RamToTrie(ram)
	fmt.Println("root =", root)
}

func printTrie(ram map[uint32](uint32)) {
	root := RamToTrie(ram)
	fmt.Println("root =", root)
	ParseNode(root, 0, func(t common.Hash) []byte {
		return Preimages[t]
	})
}

func TestBuggedTrie(t *testing.T) {
	ram := make(map[uint32](uint32))

	ram[0] = 1
	ram[4] = 2
	printTrie(ram)

	ram[0x40] = 3
	printTrie(ram)

	ram = make(map[uint32](uint32))
	ram[0x7fffd00c] = 1
	ram[0x7fffd010] = 2
	printTrie(ram)
	ram[0x7fffcffc] = 3
	printTrie(ram)
}
