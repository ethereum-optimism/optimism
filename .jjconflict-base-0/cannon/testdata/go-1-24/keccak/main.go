package main

import (
	"fmt"

	"golang.org/x/crypto/sha3"
)

func main() {
	var result []byte
	state := sha3.NewLegacyKeccak256()
	state.Write([]byte{1, 2, 3})
	result = state.Sum(result)

	fmt.Printf("keccak program. result=%x\n", result)
}
