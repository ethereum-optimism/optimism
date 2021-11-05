package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/oracle"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

type PreimageKeyValueWriter struct{}

var Preimages = make(map[common.Hash][]byte)

type Jtree struct {
	Root      common.Hash            `json:"root"`
	Step      int                    `json:"step"`
	Preimages map[common.Hash][]byte `json:"preimages"`
}

func TrieToJson(root common.Hash, step int) []byte {
	b, err := json.Marshal(Jtree{Preimages: Preimages, Step: step, Root: root})
	check(err)
	return b
}

func TrieFromJson(dat []byte) (common.Hash, int) {
	var j Jtree
	err := json.Unmarshal(dat, &j)
	check(err)
	Preimages = j.Preimages
	return j.Root, j.Step
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

func ParseNodeInternal(elems []byte, depth int, callback func(common.Hash) []byte) {
	sprefix := strings.Repeat("  ", depth)
	c, _ := rlp.CountValues(elems)
	fmt.Println(sprefix, "parsing", depth, "elements", c)
	rest := elems
	for i := 0; i < c; i++ {
		kind, val, lrest, err := rlp.Split(rest)
		rest = lrest
		check(err)
		if len(val) > 0 {
			fmt.Println(sprefix, i, kind, val, len(val))
		}
		if len(val) == 32 {
			hh := common.BytesToHash(val)
			//fmt.Println(sprefix, "node found with len", len(Preimages[hh]))
			ParseNode(hh, depth+1, callback)
		}
		if kind == rlp.List && len(val) > 0 && len(val) < 32 {
			ParseNodeInternal(val, depth+1, callback)
		}
	}
}

// full nodes / BRANCH_NODE have 17 values, each a hash
// LEAF or EXTENSION nodes have 2 values, a path and value
func ParseNode(node common.Hash, depth int, callback func(common.Hash) []byte) {
	if depth > 4 {
		return
	}
	buf := callback(node)
	//fmt.Println("callback", node, len(buf), hex.EncodeToString(buf))
	elems, _, err := rlp.SplitList(buf)
	check(err)
	ParseNodeInternal(elems, depth, callback)
}

func RamFromTrie(root common.Hash) map[uint32](uint32) {
	ram := make(map[uint32](uint32))

	// load into oracle
	pp := oracle.Preimages()
	for k, v := range Preimages {
		pp[k] = v
	}

	triedb := trie.Database{Root: root}
	tt, err := trie.New(root, &triedb)
	check(err)
	tni := tt.NodeIterator([]byte{})
	for tni.Next(true) {
		if tni.Leaf() {
			tk := binary.BigEndian.Uint32(tni.LeafKey())
			tv := binary.BigEndian.Uint32(tni.LeafBlob())
			ram[tk*4] = tv
		}
	}
	return ram
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
