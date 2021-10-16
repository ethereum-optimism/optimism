package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

type PreimageKeyValueWriter struct{}

var Preimages = make(map[common.Hash][]byte)

type Jtree struct {
	Root      common.Hash            `json:"root"`
	Preimages map[common.Hash][]byte `json:"preimages"`
}

// TODO: replace with JSON
func SerializeTrie(root common.Hash) []byte {
	b := new(bytes.Buffer)
	e := gob.NewEncoder(b)
	check(e.Encode(root))
	check(e.Encode(Preimages))
	return b.Bytes()
}

func TrieToJson(root common.Hash) []byte {
	b, err := json.Marshal(Jtree{Preimages: Preimages, Root: root})
	check(err)
	return b
}

// TODO: this is copied from the oracle
func (kw PreimageKeyValueWriter) Put(key []byte, value []byte) error {
	hash := crypto.Keccak256Hash(value)
	if hash != common.BytesToHash(key) {
		panic("bad preimage value write")
	}
	Preimages[hash] = common.CopyBytes(value)
	return nil
}

func (kw PreimageKeyValueWriter) Delete(key []byte) error {
	delete(Preimages, common.BytesToHash(key))
	return nil
}

// full nodes / BRANCH_NODE have 17 values, each a hash
// LEAF or EXTENSION nodes have 2 values, a path and value
func ParseNode(node common.Hash, depth int, callback func(common.Hash) []byte) {
	if depth > 3 {
		return
	}
	sprefix := strings.Repeat("  ", depth)
	buf := callback(node)
	elems, _, err := rlp.SplitList(buf)
	check(err)
	c, _ := rlp.CountValues(elems)
	fmt.Println(sprefix, "parsing", node, depth, "elements", c)
	rest := elems
	for i := 0; i < c; i++ {
		kind, val, lrest, err := rlp.Split(rest)
		rest = lrest
		check(err)
		fmt.Println(sprefix, i, kind, val, len(val))
		if len(val) == 32 {
			hh := common.BytesToHash(val)
			fmt.Println(sprefix, "node found with len", len(Preimages[hh]))
			ParseNode(hh, depth+1, callback)
		}
	}
}

func RamToTrie(ram map[uint32](uint32)) common.Hash {
	mt := trie.NewStackTrie(PreimageKeyValueWriter{})

	sram := make([]uint64, len(ram))

	i := 0
	for k, v := range ram {
		sram[i] = (uint64(k) << 32) | uint64(v)
		i += 1
	}
	sort.Slice(sram, func(i, j int) bool { return sram[i] < sram[j] })

	for _, kv := range sram {
		k, v := uint32(kv>>32), uint32(kv)
		k >>= 2
		//fmt.Printf("insert %x = %x\n", k, v)
		tk := make([]byte, 4)
		tv := make([]byte, 4)
		binary.BigEndian.PutUint32(tk, k)
		binary.BigEndian.PutUint32(tv, v)
		mt.Update(tk, tv)
	}
	mt.Commit()
	/*fmt.Println("ram hash", mt.Hash())
	fmt.Println("hash count", len(Preimages))
	parseNode(mt.Hash(), 0)*/
	return mt.Hash()
}
