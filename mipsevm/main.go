package main

import "fmt"

func main() {
	// step 1, generate the checkpoints every million steps using unicorn
	ram := make(map[uint32](uint32))
	LoadMappedFile("../mipigo/minigeth", ram, 0)
	LoadMappedFile(fmt.Sprintf("/tmp/eth/%d/input", 13284469), ram, 0xB0000000)
	ZeroRegisters(ram)

	// step 2 (optional), validate each 1 million chunk in EVM

	// step 3 (super optional) validate each 1 million chunk on chain

	//RunWithRam(ram, steps, debug, nil)

}
