package main

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// go test -run TestTrie

func TestToTrie(t *testing.T) {
	ram := make(map[uint32](uint32))
	LoadMappedFile("test/bin/oracle.bin", ram, 0)
	ZeroRegisters(ram)
	ram[0xC000007C] = 0x5EAD0000
	root := RamToTrie(ram)

	dat := TrieToJson(root, -1)
	fmt.Println("serialized length is", len(dat))
	ioutil.WriteFile("/tmp/cannon/oracletest.json", dat, 0644)
}

func TestTrie(t *testing.T) {
	ram := make(map[uint32](uint32))
	LoadMappedFile("../mipigo/test/test.bin", ram, 0)
	ZeroRegisters(ram)
	root := RamToTrie(ram)
	//ParseNode(root, 0)

	dat := TrieToJson(root, -1)
	fmt.Println("serialized length is", len(dat))
	ioutil.WriteFile("/tmp/cannon/ramtrie.json", dat, 0644)

	// load the trie
	oldPreLen := len(Preimages)
	Preimages = make(map[common.Hash][]byte)
	dat, err := ioutil.ReadFile("/tmp/cannon/ramtrie.json")
	check(err)
	newroot, _ := TrieFromJson(dat)
	if root != newroot {
		t.Fatal("loaded root mismatch")
	}
	if len(Preimages) != oldPreLen {
		t.Fatal("preimage length mismatch")
	}

	// load memory
	newram := RamFromTrie(newroot)

	if !reflect.DeepEqual(ram, newram) {
		t.Fatal("ram to/from mismatch")
	}
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

func TestToFromTrie(t *testing.T) {
	ram := make(map[uint32](uint32))
	ram[0] = 1
	ram[4] = 2

	trie := RamToTrie(ram)
	newram := RamFromTrie(trie)

	if !reflect.DeepEqual(ram, newram) {
		t.Fatal("ram to/from mismatch")
	}
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
