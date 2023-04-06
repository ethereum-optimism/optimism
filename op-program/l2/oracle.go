package l2

import "github.com/ethereum/go-ethereum/common"

// StateOracle defines the high-level API used to retrieve L2 state data pre-images
// The returned data is always the preimage of the requested hash.
type StateOracle interface {
	// NodeByHash retrieves the merkle-patricia trie node pre-image for a given hash.
	// Trie nodes may be from the world state trie or any account storage trie.
	// Contract code is not stored as part of the trie and must be retrieved via CodeByHash
	// Returns an error if the pre-image is unavailable.
	NodeByHash(nodeHash common.Hash) ([]byte, error)

	// CodeByHash retrieves the contract code pre-image for a given hash.
	// codeHash should be retrieved from the world state account for a contract.
	// Returns an error if the pre-image is unavailable.
	CodeByHash(codeHash common.Hash) ([]byte, error)
}
