package main

import (
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/trie"
)

type PreimageKeyValueWriter struct{}

func (kw PreimageKeyValueWriter) Put(key []byte, value []byte) error {
	//fmt.Println("preimage", key, value)
	// cache this
	return nil
}

func (kw PreimageKeyValueWriter) Delete(key []byte) error {
	return nil
}

func RamToTrie(ram map[uint32](uint32)) {
	mt := trie.NewStackTrie(PreimageKeyValueWriter{})

	tk := make([]byte, 4)
	tv := make([]byte, 4)

	sram := make([]uint64, len(ram))

	i := 0
	for k, v := range ram {
		sram[i] = (uint64(k) << 32) | uint64(v)
		i += 1
	}
	sort.Slice(sram, func(i, j int) bool { return sram[i] < sram[j] })

	for _, kv := range sram {
		k, v := uint32(kv>>32), uint32(kv)
		//fmt.Printf("insert %x = %x\n", k, v)
		binary.BigEndian.PutUint32(tk, k)
		binary.BigEndian.PutUint32(tv, v)
		mt.Update(tk, tv)
	}
	fmt.Println("ram hash", mt.Hash())
}
