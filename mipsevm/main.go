package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

func WriteCheckpoint(ram map[uint32](uint32), fn string, step int) {
	trieroot := RamToTrie(ram)
	dat := TrieToJson(trieroot, step)
	fmt.Printf("writing %s len %d with root %s\n", fn, len(dat), trieroot)
	ioutil.WriteFile(fn, dat, 0644)
}

func main() {
	root := ""
	target := -1
	if len(os.Args) > 1 {
		blockNumber, _ := strconv.Atoi(os.Args[1])
		root = fmt.Sprintf("/tmp/cannon/%d_%d", 0, blockNumber)
	}
	if len(os.Args) > 2 {
		target, _ = strconv.Atoi(os.Args[2])
	}

	// step 1, generate the checkpoints every million steps using unicorn
	ram := make(map[uint32](uint32))

	lastStep := 0
	mu := GetHookedUnicorn(root, ram, func(step int, mu uc.Unicorn, ram map[uint32](uint32)) {
		// this can be raised to 10,000,000 if the files are too large
		if step%10000000 == 0 || step == target {
			SyncRegs(mu, ram)
			fn := fmt.Sprintf("%s/checkpoint_%d.json", root, step)
			WriteCheckpoint(ram, fn, step)
			if step == target {
				// done
				mu.RegWrite(uc.MIPS_REG_PC, 0x5ead0004)
			}
		}
		lastStep = step
	})

	ZeroRegisters(ram)
	// not ready for golden yet
	LoadMappedFileUnicorn(mu, "mipigo/minigeth.bin", ram, 0)
	WriteCheckpoint(ram, "/tmp/cannon/golden.json", -1)
	if root == "" {
		fmt.Println("exiting early without a block number")
		os.Exit(0)
	}

	// TODO: this is actually step 0->1. Renumber as appropriate
	LoadMappedFileUnicorn(mu, fmt.Sprintf("%s/input", root), ram, 0x30000000)

	mu.Start(0, 0x5ead0004)
	SyncRegs(mu, ram)
	if target == -1 {
		WriteCheckpoint(ram, fmt.Sprintf("%s/checkpoint_final.json", root), lastStep)
	}

	// step 2 (optional), validate each 1 million chunk in EVM

	// step 3 (super optional) validate each 1 million chunk on chain

	//RunWithRam(ram, steps, debug, nil)

}
