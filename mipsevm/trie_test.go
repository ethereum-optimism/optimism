package main

import (
	"fmt"
	"io/ioutil"
	"testing"
)

// go test -run TestTrie

func TestTrie(t *testing.T) {
	fn := "../mipigo/test/test.bin"
	ram := make(map[uint32](uint32))

	// TODO: copied from compare_test.go
	LoadMappedFile(fn, ram, 0)
	/*inputFile := fmt.Sprintf("/tmp/eth/%d", 13284469)
	LoadMappedFile(inputFile, ram, 0xB0000000)*/
	for i := uint32(0xC0000000); i < 0xC0000000+36*4; i += 4 {
		WriteRam(ram, i, 0)
	}

	root := RamToTrie(ram)
	dat := SerializeTrie(root)
	fmt.Println("serialized length is", len(dat))
	ioutil.WriteFile("/tmp/eth/ramtrie", dat, 0644)
}
