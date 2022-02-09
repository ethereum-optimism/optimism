package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
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

	regfault := -1
	regfault_str, regfault_valid := os.LookupEnv("REGFAULT")
	if regfault_valid {
		regfault, _ = strconv.Atoi(regfault_str)
	}

	basedir := os.Getenv("BASEDIR")
	if len(basedir) == 0 {
		basedir = "/tmp/cannon"
	}
	if len(os.Args) > 1 {
		blockNumber, _ := strconv.Atoi(os.Args[1])
		root = fmt.Sprintf("%s/%d_%d", basedir, 0, blockNumber)
	}
	if len(os.Args) > 2 {
		target, _ = strconv.Atoi(os.Args[2])
	}
	evm := false
	if len(os.Args) > 3 && os.Args[3] == "evm" {
		evm = true
	}

	// step 1, generate the checkpoints every million steps using unicorn
	ram := make(map[uint32](uint32))

	lastStep := 1
	if evm {
		// TODO: fix this
		/*ZeroRegisters(ram)
		LoadMappedFile("mipigo/minigeth.bin", ram, 0)
		WriteCheckpoint(ram, "/tmp/cannon/golden.json", -1)
		LoadMappedFile(fmt.Sprintf("%s/input", root), ram, 0x30000000)
		RunWithRam(ram, target-1, 0, root, nil)
		lastStep += target - 1
		fn := fmt.Sprintf("%s/checkpoint_%d.json", root, lastStep)
		WriteCheckpoint(ram, fn, lastStep)*/
	} else {
		mu := GetHookedUnicorn(root, ram, func(step int, mu uc.Unicorn, ram map[uint32](uint32)) {
			// it seems this runs before the actual step happens
			// this can be raised to 10,000,000 if the files are too large
			//if (target == -1 && step%10000000 == 0) || step == target {
			// first run checkpointing is disabled for now since is isn't used
			if step == regfault {
				fmt.Printf("regfault at step %d\n", step)
				mu.RegWrite(uc.MIPS_REG_V0, 0xbabababa)
			}
			if step == target {
				SyncRegs(mu, ram)
				fn := fmt.Sprintf("%s/checkpoint_%d.json", root, step)
				WriteCheckpoint(ram, fn, step)
				if step == target {
					// done
					mu.RegWrite(uc.MIPS_REG_PC, 0x5ead0004)
				}
			}
			lastStep = step + 1
		})

		ZeroRegisters(ram)
		// not ready for golden yet
		LoadMappedFileUnicorn(mu, "mipigo/minigeth.bin", ram, 0)
		if root == "" {
			WriteCheckpoint(ram, fmt.Sprintf("%s/golden.json", basedir), -1)
			fmt.Println("exiting early without a block number")
			os.Exit(0)
		}

		LoadMappedFileUnicorn(mu, fmt.Sprintf("%s/input", root), ram, 0x30000000)

		mu.Start(0, 0x5ead0004)
		SyncRegs(mu, ram)
	}

	if target == -1 {
		if ram[0x30000800] != 0x1337f00d {
			log.Fatal("failed to output state root, exiting")
		}

		output_filename := fmt.Sprintf("%s/output", root)
		outputs, err := ioutil.ReadFile(output_filename)
		check(err)
		real := append([]byte{0x13, 0x37, 0xf0, 0x0d}, outputs...)

		output := []byte{}
		for i := 0; i < 0x44; i += 4 {
			t := make([]byte, 4)
			binary.BigEndian.PutUint32(t, ram[uint32(0x30000800+i)])
			output = append(output, t...)
		}

		if bytes.Compare(real, output) != 0 {
			fmt.Println("MISMATCH OUTPUT, OVERWRITING!!!")
			ioutil.WriteFile(output_filename, output[4:], 0644)
		} else {
			fmt.Println("output match")
		}

		WriteCheckpoint(ram, fmt.Sprintf("%s/checkpoint_final.json", root), lastStep)

	}

	// step 2 (optional), validate each 1 million chunk in EVM

	// step 3 (super optional) validate each 1 million chunk on chain

	//RunWithRam(ram, steps, debug, nil)

}
