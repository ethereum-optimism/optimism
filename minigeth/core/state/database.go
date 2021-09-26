package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/oracle"
	"github.com/ethereum/go-ethereum/trie"
)

// TODO: add oracle calls here
// wrapper for the oracle

type Database struct {
	db          *trie.Database
	BlockNumber *big.Int
	StateRoot   common.Hash
}

func NewDatabase(header types.Header) Database {
	//triedb := trie.Database{BlockNumber: header.Number, Root: header.Root}
	//triedb.Preseed()
	triedb := trie.NewDatabase(header)
	return Database{db: &triedb, BlockNumber: header.Number, StateRoot: header.Root}
}

// ContractCode retrieves a particular contract's code.
func (db *Database) ContractCode(addrHash common.Hash, codeHash common.Hash) ([]byte, error) {
	oracle.PrefetchCode(db.BlockNumber, addrHash)
	code := oracle.Preimage(codeHash)
	return code, nil
}

// ContractCodeSize retrieves a particular contracts code's size.
func (db *Database) ContractCodeSize(addrHash common.Hash, codeHash common.Hash) (int, error) {
	oracle.PrefetchCode(db.BlockNumber, addrHash)
	code := oracle.Preimage(codeHash)
	return len(code), nil
}

func (db *Database) CopyTrie(trie Trie) Trie {
	panic("don't copy tries")
}

// OpenTrie opens the main account trie at a specific root hash.
func (db *Database) OpenTrie(root common.Hash) (Trie, error) {
	tr, err := trie.NewSecure(root, db.db)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

// OpenStorageTrie opens the storage trie of an account.
func (db *Database) OpenStorageTrie(addrHash, root common.Hash) (Trie, error) {
	//return SimpleTrie{db.BlockNumber, root, true, addrHash}, nil
	tr, err := trie.NewSecure(root, db.db)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

type Trie interface {
	// TryGet returns the value for key stored in the trie. The value bytes must
	// not be modified by the caller. If a node was not found in the database, a
	// trie.MissingNodeError is returned.
	TryGet(key []byte) ([]byte, error)

	// TryUpdate associates key with value in the trie. If value has length zero, any
	// existing value is deleted from the trie. The value bytes must not be modified
	// by the caller while they are stored in the trie. If a node was not found in the
	// database, a trie.MissingNodeError is returned.
	TryUpdate(key, value []byte) error

	// TryDelete removes any existing value for key from the trie. If a node was not
	// found in the database, a trie.MissingNodeError is returned.
	TryDelete(key []byte) error

	// Hash returns the root hash of the trie. It does not write to the database and
	// can be used even if the trie doesn't have one.
	Hash() common.Hash

	// Commit writes all nodes to the trie's memory database, tracking the internal
	// and external (for account tries) references.
	Commit(onleaf trie.LeafCallback) (common.Hash, error)
}

// stubbed: we don't prefetch

type triePrefetcher struct {
}

func (p *triePrefetcher) prefetch(root common.Hash, keys [][]byte) {
}

func (p *triePrefetcher) used(root common.Hash, used [][]byte) {
}

func (p *triePrefetcher) close() {
}

func (p *triePrefetcher) copy() *triePrefetcher {
	return p
}

func (p *triePrefetcher) trie(root common.Hash) Trie {
	return nil
}
