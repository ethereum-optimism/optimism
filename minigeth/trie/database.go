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

	preimages     map[common.Hash][]byte // Preimages of nodes from the secure trie
	preimagesSize common.StorageSize     // Storage size of the preimages cache
}

// insertPreimage writes a new trie node pre-image to the memory database if it's
// yet unknown. The method will NOT make a copy of the slice,
// only use if the preimage will NOT be changed later on.
//
// Note, this method assumes that the database's lock is held!
func (db *Database) insertPreimage(hash common.Hash, preimage []byte) {
	// Short circuit if preimage collection is disabled
	if db.preimages == nil {
		return
	}
	// Track the preimage if a yet unknown one
	if _, ok := db.preimages[hash]; ok {
		return
	}
	db.preimages[hash] = preimage
	db.preimagesSize += common.StorageSize(common.HashLength + len(preimage))
}

// preimage retrieves a cached trie node pre-image from memory. If it cannot be
// found cached, the method queries the persistent database for the content.
func (db *Database) preimage(hash common.Hash) []byte {
	// Short circuit if preimage collection is disabled
	if db.preimages == nil {
		return nil
	}
	return db.preimages[hash]
}

func NewDatabase(header types.Header) Database {
	triedb := Database{BlockNumber: header.Number, Root: header.Root}
	//triedb.preimages = make(map[common.Hash][]byte)
	fmt.Println("init database")
	oracle.PrefetchAddress(header.Number, common.Address{})

	//panic("preseed")
	return triedb
}

// Node retrieves an encoded cached trie node from memory. If it cannot be found
// cached, the method queries the persistent database for the content.
func (db *Database) Node(hash common.Hash) ([]byte, error) {
	/*fmt.Println("trie Node", hash)
	return []byte{}, nil*/
	panic("no Node")
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
	//panic("insert")
	//fmt.Println("insert", hash, size)
	// can put things in the oracle here if we care
}
