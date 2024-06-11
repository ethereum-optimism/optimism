package mpt

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/hashdb"
)

// ReadTrie takes a Merkle Patricia Trie (MPT) root of a "DerivableList", and a pre-image oracle getter,
// and traverses the implied MPT to collect all raw leaf nodes in order, which are then returned.
func ReadTrie(root common.Hash, getPreimage func(key common.Hash) []byte) []hexutil.Bytes {
	odb := &DB{db: Hooks{
		Get: func(key []byte) []byte {
			if len(key) != 32 {
				panic(fmt.Errorf("expected 32 byte key query, but got %d bytes: %x", len(key), key))
			}
			return getPreimage(*(*[32]byte)(key))
		},
		Put: func(key []byte, value []byte) {
			panic("put not supported")
		},
		Delete: func(key []byte) {
			panic("delete not supported")
		},
	}}

	// trie.New backed with a trie.NodeReader and trie.Reader seems really promising
	// for a simple node-fetching backend, but the interface is half-private,
	// while we already have the full database code for doing the same thing.
	// Maybe it's still worth a small diff in geth to expose it?
	// Diff would be:
	//
	//      type Node = node
	//
	//      func DecodeNode(hash, buf []byte) (node, error) {
	//      	return decodeNode(hash, buf)
	//      }
	//
	// And then still some code here to implement the trie.NodeReader and trie.Reader
	// interfaces to map to the getPreimageFunction.
	//
	// For now we just use the state DB trie approach.

	tdb := triedb.NewDatabase(odb, &triedb.Config{HashDB: hashdb.Defaults})
	tr, err := trie.New(trie.TrieID(root), tdb)
	if err != nil {
		panic(err)
	}
	iter, err := tr.NodeIterator(nil)
	if err != nil {
		panic(err)
	}

	// With small lists the iterator seems to use 0x80 (RLP empty string, unlike the others)
	// as key for item 0, causing it to come last.
	// Let's just remember the keys, and reorder them in the canonical order, to ensure it is correct.
	var values [][]byte
	var keys []uint64
	for iter.Next(true) {
		if iter.Leaf() {
			k := iter.LeafKey()
			var x uint64
			err := rlp.DecodeBytes(k, &x)
			if err != nil {
				panic(fmt.Errorf("invalid key: %w", err))
			}
			keys = append(keys, x)
			values = append(values, iter.LeafBlob())
		}
	}
	out := make([]hexutil.Bytes, len(values))
	for i, x := range keys {
		if x >= uint64(len(values)) {
			panic(fmt.Errorf("bad key: %d", x))
		}
		if out[x] != nil {
			panic(fmt.Errorf("duplicate key %d", x))
		}
		out[x] = values[i]
	}
	return out
}

type rawList []hexutil.Bytes

func (r rawList) Len() int {
	return len(r)
}

func (r rawList) EncodeIndex(i int, buf *bytes.Buffer) {
	buf.Write(r[i])
}

var _ types.DerivableList = rawList(nil)

type noResetHasher struct {
	*trie.StackTrie
}

// Reset is intercepted and is no-op, because we want to retain the writing function when calling types.DeriveSha
func (n noResetHasher) Reset() {}

// WriteTrie takes a list of values, and merkleizes them as a "DerivableList":
// a Merkle Patricia Trie (MPT) with values keyed by their RLP encoded index.
// This merkleization matches that of transactions, receipts, and withdrawals lists in the block header
// (at least up to the Shanghai L1 update).
// This then returns the MPT root and a list of pre-images of the trie.
// Note: empty values are illegal, and there may be less pre-images returned than values,
// if any values are less than 32 bytes and fit into branch-node slots that way.
func WriteTrie(values []hexutil.Bytes) (common.Hash, []hexutil.Bytes) {
	var out []hexutil.Bytes
	st := noResetHasher{
		trie.NewStackTrie(func(path []byte, hash common.Hash, blob []byte) {
			out = append(out, common.CopyBytes(blob)) // the stack hasher may mutate the blob bytes, so copy them.
		}),
	}
	root := types.DeriveSha(rawList(values), st)
	return root, out
}
