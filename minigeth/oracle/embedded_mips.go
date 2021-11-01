//go:build mips
// +build mips

package oracle

import (
	"math/big"
	"os"
	"reflect"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var preimages = make(map[common.Hash][]byte)

func byteAt(addr uint64, length int) []byte {
	var ret []byte
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&ret))
	bh.Data = uintptr(addr)
	bh.Len = length
	bh.Cap = length
	return ret
}

func InputHash() common.Hash {
	ret := byteAt(0x30000000, 0x20)
	os.Stderr.WriteString("********* on chain starts here *********\n")
	return common.BytesToHash(ret)
}

func Halt() {
	//os.Stderr.WriteString("THIS SHOULD BE PATCHED OUT\n")
	// the exit syscall is a jump to 0x5ead0000 now
	os.Exit(0)
}

func Output(output common.Hash, receipts common.Hash) {
	ret := byteAt(0x30000804, 0x20)
	copy(ret, output.Bytes())
	rret := byteAt(0x30000824, 0x20)
	copy(rret, receipts.Bytes())
	magic := byteAt(0x30000800, 4)
	copy(magic, []byte{0x13, 0x37, 0xf0, 0x0d})
	Halt()
}

func Preimage(hash common.Hash) []byte {
	val, ok := preimages[hash]
	if !ok {
		// load in hash
		preImageHash := byteAt(0x30001000, 0x20)
		copy(preImageHash, hash.Bytes())

		// used in unicorn emulator to trigger the load
		// in onchain mips, it's instant
		os.Getpid()

		// ready
		rawSize := common.CopyBytes(byteAt(0x31000000, 4))
		size := (int(rawSize[0]) << 24) | (int(rawSize[1]) << 16) | (int(rawSize[2]) << 8) | int(rawSize[3])
		ret := common.CopyBytes(byteAt(0x31000004, size))

		// this is 20% of the exec instructions, this speedup is always an option
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
func SetNodeUrl(newNodeUrl string)                                                        {}
func SetRoot(newRoot string)                                                              {}
func PrefetchStorage(*big.Int, common.Address, common.Hash, func(map[common.Hash][]byte)) {}
func PrefetchAccount(*big.Int, common.Address, func(map[common.Hash][]byte))              {}
func PrefetchCode(blockNumber *big.Int, addrHash common.Hash)                             {}
func PrefetchBlock(blockNumber *big.Int, startBlock bool, hasher types.TrieHasher)        {}

// KeyValueWriter wraps the Put method of a backing data store.
type PreimageKeyValueWriter struct{}

func (kw PreimageKeyValueWriter) Put(key []byte, value []byte) error { return nil }
func (kw PreimageKeyValueWriter) Delete(key []byte) error            { return nil }
