package main

import (
	"fmt"
	"log"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func LoadRam() map[uint32](uint32) {
	ram := make(map[uint32](uint32))
	fn := "../mipigo/test/test.bin"
	//fn := "test/bin/add.bin"
	LoadMappedFile(fn, ram, 0)
	ZeroRegisters(ram)
	ram[0xC000007C] = 0x5EAD0000
	return ram
}

// go test -run TestCompareEvmChain

func TestCompareEvmChain(t *testing.T) {
	totalSteps := 20

	cchain := make(chan common.Hash, 1)
	cuni := make(chan common.Hash, 1)

	// only need one ram
	ram := LoadRam()

	root := RamToTrie(ram)
	fmt.Println("state root", root, "nodes", len(Preimages))

	// deploy chain
	interpreter, statedb := GetInterpreter(0, true, "")
	DeployChain(interpreter, statedb)

	// load chain trie node
	for _, v := range Preimages {
		//fmt.Println("AddTrieNode", k)
		AddTrieNode(v, interpreter, statedb)
	}

	// run on (fake) chain
	go func(root common.Hash) {
		for step := 0; step < totalSteps; step++ {
			steps := 1
			input := crypto.Keccak256Hash([]byte("Steps(bytes32,uint256)")).Bytes()[:4]
			input = append(input, root.Bytes()...)
			input = append(input, common.BigToHash(big.NewInt(int64(steps))).Bytes()...)
			dat, _, err := RunWithInputAndGas(interpreter, statedb, input, uint64(100000000))
			if err != nil {
				if len(dat) >= 0x24 {
					fmt.Println(string(dat[0x24:]))
				}
				log.Fatal(err)
			} else {
				root = common.BytesToHash(dat)
				//fmt.Println("new state root", step, root, "gas used", gasUsed)
				cchain <- root
			}
		}
	}(root)

	// run on evm
	go func() {
		for step := 0; step < totalSteps; step++ {
			RunWithRam(ram, 1, 0, "", nil)
			root = RamToTrie(ram)
			cuni <- root
		}
	}()

	for i := 0; i < totalSteps; i++ {
		x, y := <-cchain, <-cuni
		fmt.Println(i, x, y)
		if x != y {
			log.Fatal("mismatch at step", i)
		}
	}

	/*ParseNode(root, 0, func(t common.Hash) []byte {
		return Preimages[t]
	})*/
}
