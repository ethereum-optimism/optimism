package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/oracle"
)

// TODO: add oracle calls here
// wrapper for the oracle

type Database struct {
	BlockNumber *big.Int
	StateRoot   common.Hash
}

var unhashMap = make(map[common.Hash]common.Address)

func unhash(addrHash common.Hash) common.Address {
	return unhashMap[addrHash]
}

// ContractCode retrieves a particular contract's code.
func (db *Database) ContractCode(addrHash common.Hash, codeHash common.Hash) ([]byte, error) {
	addr := unhash(addrHash)
	code := oracle.GetProvedCodeBytes(db.BlockNumber, addr, codeHash)
	return code, nil
}

func (db *Database) CopyTrie(trie Trie) Trie {
	// TODO: this is wrong
	return trie
}

// ContractCodeSize retrieves a particular contracts code's size.
func (db *Database) ContractCodeSize(addrHash common.Hash, codeHash common.Hash) (int, error) {
	addr := unhash(addrHash)
	code := oracle.GetProvedCodeBytes(db.BlockNumber, addr, codeHash)
	return len(code), nil
}

// OpenStorageTrie opens the storage trie of an account.
func (db *Database) OpenStorageTrie(addrHash, root common.Hash) (Trie, error) {
	return SimpleTrie{}, nil
}

type LeafCallback func(paths [][]byte, hexpath []byte, leaf []byte, parent common.Hash) error

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
	Commit(onleaf LeafCallback) (common.Hash, error)
}

type SimpleTrie struct {
	BlockNumber *big.Int
	StateRoot   common.Hash
}

func (trie SimpleTrie) Commit(onleaf LeafCallback) (common.Hash, error) {
	return trie.StateRoot, nil
}

func (trie SimpleTrie) Hash() common.Hash {
	return trie.StateRoot
}

func (trie SimpleTrie) TryUpdate(key, value []byte) error {
	return nil
}

func (trie SimpleTrie) TryDelete(key []byte) error {
	return nil
}

func (trie SimpleTrie) TryGet(key []byte) ([]byte, error) {
	address := common.BytesToAddress(key)
	addrHash := crypto.Keccak256Hash(address[:])
	unhashMap[addrHash] = address
	enc := oracle.GetProvedAccountBytes(trie.BlockNumber, trie.StateRoot, address)
	return enc, nil
}

// stubbed: we don't prefetch

type triePrefetcher struct {
}

func (p *triePrefetcher) prefetch(root common.Hash, keys [][]byte) {
}

func (p *triePrefetcher) used(root common.Hash, used [][]byte) {
}

func (p *triePrefetcher) trie(root common.Hash) Trie {
	return nil
}
