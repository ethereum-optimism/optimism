//go:build !mips
// +build !mips

package oracle

import (
	"fmt"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var preimages = make(map[common.Hash][]byte)

func Preimage(hash common.Hash) []byte {
	val, ok := preimages[hash]
	key := fmt.Sprintf("/tmp/eth/%s", hash)
	ioutil.WriteFile(key, val, 0644)
	if !ok {
		fmt.Println("can't find preimage", hash)
	}
	comphash := crypto.Keccak256Hash(val)
	if hash != comphash {
		panic("corruption in hash " + hash.String())
	}
	return val
}

// TODO: Maybe we will want to have a seperate preimages for next block's preimages?
func Preimages() map[common.Hash][]byte {
	return preimages
}

// KeyValueWriter wraps the Put method of a backing data store.
type PreimageKeyValueWriter struct{}

// Put inserts the given value into the key-value data store.
func (kw PreimageKeyValueWriter) Put(key []byte, value []byte) error {
	hash := crypto.Keccak256Hash(value)
	if hash != common.BytesToHash(key) {
		panic("bad preimage value write")
	}
	nval := make([]byte, len(value))
	copy(nval, value)
	preimages[hash] = nval
	//fmt.Println("tx preimage", hash, common.Bytes2Hex(value))
	return nil
}

// Delete removes the key from the key-value data store.
func (kw PreimageKeyValueWriter) Delete(key []byte) error {
	return nil
}
