package main

import (
	"fmt"
	"io/ioutil"

	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

func WriteCheckpoint(ram map[uint32](uint32), root string, step int) {
	trieroot := RamToTrie(ram)
	dat := TrieToJson(trieroot)
	fn := fmt.Sprintf("%s/checkpoint_%d.json", root, step)
	fmt.Printf("writing %s len %d with root %s\n", fn, len(dat), trieroot)
	ioutil.WriteFile(fn, dat, 0644)
}

func main() {
	root := fmt.Sprintf("/tmp/eth/%d", 13284469)
	// step 1, generate the checkpoints every million steps using unicorn
	ram := make(map[uint32](uint32))

	lastStep := 0
	mu := GetHookedUnicorn(root, ram, func(step int, mu uc.Unicorn, ram map[uint32](uint32)) {
		// this can be raised to 10,000,000 if the files are too large
		if step%1000000 == 0 {
			SyncRegs(mu, ram)
			WriteCheckpoint(ram, root, step)
		}
		lastStep = step
	})

	ZeroRegisters(ram)
	LoadMappedFileUnicorn(mu, "../mipigo/golden/minigeth.bin", ram, 0)
	WriteCheckpoint(ram, root, -1)
	LoadMappedFileUnicorn(mu, fmt.Sprintf("%s/input", root), ram, 0xB0000000)

	mu.Start(0, 0x5ead0004)
	WriteCheckpoint(ram, root, lastStep)

	// step 2 (optional), validate each 1 million chunk in EVM

	// step 3 (super optional) validate each 1 million chunk on chain

	//RunWithRam(ram, steps, debug, nil)

}
