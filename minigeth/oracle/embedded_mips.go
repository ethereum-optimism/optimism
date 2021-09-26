//go:build mips
// +build mips

package oracle

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var preimages = make(map[common.Hash][]byte)
var inputs [6]common.Hash
var inputsLoaded bool = false

func Input(index int) common.Hash {
	if index < 0 || index > 5 {
		panic("bad input index")
	}
	if !inputsLoaded {
		blockNumber, _ := strconv.Atoi(os.Args[1])
		f, err := os.Open(fmt.Sprintf("/tmp/eth/%d", blockNumber))
		if err != nil {
			panic("missing inputs")
		}
		defer f.Close()
		ret, err := ioutil.ReadAll(f)

		for i := 0; i < len(inputs); i++ {
			inputs[i] = common.BytesToHash(ret[i*0x20 : i*0x20+0x20])
		}

		inputsLoaded = true
	}
	return inputs[index]
}

func Output(output common.Hash) {
	if output == inputs[5] {
		fmt.Println("good transition")
	} else {
		fmt.Println(output, "!=", inputs[5])
		panic("BAD transition :((")
	}
}

func Preimage(hash common.Hash) []byte {
	val, ok := preimages[hash]
	if !ok {
		f, err := os.Open(fmt.Sprintf("/tmp/eth/%s", hash))
		if err != nil {
			panic("missing preimage")
		}

		defer f.Close()
		ret, err := ioutil.ReadAll(f)
		if err != nil {
			panic("preimage read failed")
		}

		realhash := crypto.Keccak256Hash(ret)
		if realhash != hash {
			panic("preimage has wrong hash")
		}

		preimages[hash] = ret
		return ret
	}
	return val
}

// these are stubs in embedded world
func PrefetchStorage(blockNumber *big.Int, addr common.Address, skey common.Hash)  {}
func PrefetchAccount(blockNumber *big.Int, addr common.Address)                    {}
func PrefetchCode(blockNumber *big.Int, addrHash common.Hash)                      {}
func PrefetchBlock(blockNumber *big.Int, startBlock bool, hasher types.TrieHasher) {}
