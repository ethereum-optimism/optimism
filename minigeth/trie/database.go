package trie

import (
	"fmt"
	"io"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/oracle"
)

// rawNode is a simple binary blob used to differentiate between collapsed trie
// nodes and already encoded RLP binary blobs (while at the same time store them
// in the same cache fields).
type rawNode []byte

func (n rawNode) cache() (hashNode, bool)   { panic("this should never end up in a live trie") }
func (n rawNode) fstring(ind string) string { panic("this should never end up in a live trie") }

func (n rawNode) EncodeRLP(w io.Writer) error {
	_, err := w.Write(n)
	return err
}

type Database struct {
	BlockNumber *big.Int
	Root        common.Hash
	lock        sync.RWMutex
}

func NewDatabase(header types.Header) Database {
	triedb := Database{BlockNumber: header.Number, Root: header.Root}
	//triedb.preimages = make(map[common.Hash][]byte)
	fmt.Println("init database")
	oracle.PrefetchAccount(header.Number, common.Address{})

	//panic("preseed")
	return triedb
}

// Node retrieves an encoded cached trie node from memory. If it cannot be found
// cached, the method queries the persistent database for the content.
func (db *Database) Node(hash common.Hash) ([]byte, error) {
	panic("no Node function")
}

// node retrieves a cached trie node from memory, or returns nil if none can be
// found in the memory cache.
func (db *Database) node(hash common.Hash) node {
	//fmt.Println("node", hash)
	return mustDecodeNode(hash[:], oracle.Preimage(hash))
}

// insert inserts a collapsed trie node into the memory database.
// The blob size must be specified to allow proper size tracking.
// All nodes inserted by this function will be reference tracked
// and in theory should only used for **trie nodes** insertion.
func (db *Database) insert(hash common.Hash, size int, node node) {
	// can put things in the oracle here if we care
	//fmt.Println("insert", hash, size)
}
