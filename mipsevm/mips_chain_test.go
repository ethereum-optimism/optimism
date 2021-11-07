package main

import (
	"fmt"
	"log"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestMipsChain(t *testing.T) {
	// only need one ram
	ram := LoadRam()
	root := RamToTrie(ram)

	interpreter, statedb := GetInterpreter(1, true, "")
	DeployChain(interpreter, statedb)

	// load chain trie node
	for _, v := range Preimages {
		//fmt.Println("AddTrieNode", k)
		AddTrieNode(v, interpreter, statedb)
	}

	steps := 50
	input := crypto.Keccak256Hash([]byte("Steps(bytes32,uint256)")).Bytes()[:4]
	input = append(input, root.Bytes()...)
	input = append(input, common.BigToHash(big.NewInt(int64(steps))).Bytes()...)
	dat, gasUsed, err := RunWithInputAndGas(interpreter, statedb, input, uint64(steps*10000000))

	if err != nil {
		if len(dat) >= 0x24 {
			fmt.Println(string(dat[0x24:]))
		}
		log.Fatal(err)
	} else {
		root = common.BytesToHash(dat)
		fmt.Println("new state root", root, "gas used", gasUsed)
	}
}
