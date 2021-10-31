package main

import (
	"fmt"
	"io/ioutil"

	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

func main() {
	root := fmt.Sprintf("/tmp/eth/%d", 13284469)
	// step 1, generate the checkpoints every million steps using unicorn
	ram := make(map[uint32](uint32))

	ZeroRegisters(ram)

	mu := GetHookedUnicorn(root, ram, func(step int, mu uc.Unicorn, ram map[uint32](uint32)) {
		if step%1000000 == 0 {
			trieroot := RamToTrie(ram)
			dat := TrieToJson(trieroot)
			fn := fmt.Sprintf("%s/checkpoint_%d.json", root, step)
			fmt.Printf("writing %s len %d\n", fn, len(dat))
			ioutil.WriteFile(fn, dat, 0644)
		}
	})
	check(mu.MemMap(0, 0x80000000))

	LoadMappedFileUnicorn(mu, "../mipigo/minigeth.bin", ram, 0)
	LoadMappedFileUnicorn(mu, fmt.Sprintf("%s/input", root), ram, 0xB0000000)

	mu.Start(0, 0x5ead0004)

	// step 2 (optional), validate each 1 million chunk in EVM

	// step 3 (super optional) validate each 1 million chunk on chain

	//RunWithRam(ram, steps, debug, nil)

}
