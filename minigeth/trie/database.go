package trie

import (
	"bytes"
	"io"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/oracle"
	"github.com/ethereum/go-ethereum/rlp"
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
	//fmt.Println("init database")
	oracle.PrefetchAccount(header.Number, common.Address{}, nil)

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
	if val := oracle.Preimage(hash); val != nil {
		return mustDecodeNode(hash[:], val)
	}
	return nil
}

// insert inserts a collapsed trie node into the memory database.
// The blob size must be specified to allow proper size tracking.
// All nodes inserted by this function will be reference tracked
// and in theory should only used for **trie nodes** insertion.
func (db *Database) insert(hash common.Hash, size int, node node) {
	// can put things in the oracle here if we care
	//fmt.Println("insert", hash, size)
}

func GenPossibleShortNodePreimage(preimages map[common.Hash][]byte) {
	newPreimages := make(map[common.Hash][]byte)

	for _, val := range preimages {
		node, err := decodeNode(nil, val)
		if err != nil {
			continue
		}

		if node, ok := node.(*shortNode); ok {
			for i := len(node.Key) - 1; i > 0; i-- {
				n := shortNode{
					Key: hexToCompact(node.Key[i:]),
					Val: node.Val,
				}
				buf := new(bytes.Buffer)
				if err := rlp.Encode(buf, n); err != nil {
					panic("encode error: " + err.Error())
				}
				preimage := buf.Bytes()
				if len(preimage) < 32 {
					continue
				}
				newPreimages[crypto.Keccak256Hash(preimage)] = preimage
			}
		}
	}

	for hash, val := range newPreimages {
		preimages[hash] = val
	}
}
