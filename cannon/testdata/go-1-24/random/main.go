package main

import (
	crand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

func main() {
	// Read a uint64
	var buf = make([]byte, 8)
	readRandomBytes(buf)
	randomInt := binary.BigEndian.Uint64(buf)
	fmt.Printf("Random int: %d\n", randomInt)

	// Read a large chunk of bytes all at once
	buf = make([]byte, 25)
	readRandomBytes(buf)
	printRandomBuffer(buf)

	// Read a small number of bytes 1 at a time
	buf = make([]byte, 5)
	for i := 0; i < len(buf); i++ {
		readRandomBytes(buf[i : i+1])
	}
	printRandomBuffer(buf)
}

func printRandomBuffer(buf []byte) {
	hexValue := hex.EncodeToString(buf)
	fmt.Printf("Random hex data: %v\n", hexValue)
}

func readRandomBytes(buf []byte) {
	n, err := crand.Read(buf)
	if err != nil {
		fmt.Printf("Error reading bytes: %v\n", err)
		panic(err)
	}
	if n != len(buf) {
		fmt.Printf("Read %d bytes, expected %d\n", n, len(buf))
		panic("Read wrong number of bytes")
	}
}
