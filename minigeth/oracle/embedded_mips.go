//go:build mips
// +build mips

package oracle

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var preimages = make(map[common.Hash][]byte)

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
