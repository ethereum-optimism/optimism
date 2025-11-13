package main

import (
	"fmt"
	"keccak-xorasm/xor"
)

func main() {
	a := make([]byte, 3)
	p := []byte{1, 2, 3}

	xor.XORBytes(a, a, p)

	fmt.Printf("keccak program. result=%x\n", a)
}
