package eth

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

type StorageProofEntry struct {
	Key   common.Hash     `json:"key"`
	Value hexutil.Bytes   `json:"value"`
	Proof []hexutil.Bytes `json:"proof"`
}

type AccountResult struct {
	AccountProof []hexutil.Bytes `json:"accountProof"`

	Address     common.Address `json:"address"`
	Balance     *hexutil.Big   `json:"balance"`
	CodeHash    common.Hash    `json:"codeHash"`
	Nonce       hexutil.Uint64 `json:"nonce"`
	StorageHash common.Hash    `json:"storageHash"`

	// Optional
	StorageProof []StorageProofEntry `json:"storageProof,omitempty"`
}

// Verify an account (and optionally storage) proof from the getProof RPC. See https://eips.ethereum.org/EIPS/eip-1186
func (res *AccountResult) Verify(stateRoot common.Hash) error {
	// verify storage proof values, if any, against the storage trie root hash of the account
	if len(res.StorageProof) > 0 {
		// load all MPT nodes into a DB
		db := memorydb.New()
		for i, entry := range res.StorageProof {
			for j, encodedNode := range entry.Proof {
				nodeKey := encodedNode
				if len(encodedNode) >= 32 { // small MPT nodes are not hashed
					nodeKey = crypto.Keccak256(encodedNode)
				}
				if err := db.Put(nodeKey, encodedNode); err != nil {
					return fmt.Errorf("failed to load storage proof node %d of storage value %d into mem db: %w", j, i, err)
				}
			}
		}
		// interpret the DB of MPT nodes as an MPT node database
		trieDB := trie.NewDatabase(db)
		// Now select the Trie anchored at the storage hash,
		// this should be able to resolve all the values, if they are in the trie with this root.
		proofTrie, err := trie.New(trie.StateTrieID(res.StorageHash), trieDB)
		if err != nil {
			return fmt.Errorf("failed to load db wrapper around storage trie kv store: %w", err)
		}
		// Now verify all the storage values are present in the trie at the path of their key.
		for i, entry := range res.StorageProof {
			path := crypto.Keccak256(entry.Key[:])
			val, err := proofTrie.TryGet(path)
			if err != nil {
				return fmt.Errorf("failed to find storage value %d with key %s (path %x) in storage trie %s: %w", i, entry.Key, path, res.StorageHash, err)
			}
			expectedNodeData, err := rlp.EncodeToBytes(entry.Value)
			if err != nil {
				return fmt.Errorf("failed to encode storage proof value %d as rlp string: %w", i, err)
			}
			if !bytes.Equal(val, expectedNodeData) {
				return fmt.Errorf("value %d in storage proof does not match proven value at key %s (path %x)", i, entry.Key, path)
			}
		}
	}

	accountClaimed := []any{uint64(res.Nonce), (*big.Int)(res.Balance).Bytes(), res.StorageHash, res.CodeHash}
	accountClaimedValue, err := rlp.EncodeToBytes(accountClaimed)
	if err != nil {
		return fmt.Errorf("failed to encode account from retrieved values: %w", err)
	}

	// create a db with all account trie nodes
	db := memorydb.New()
	for i, encodedNode := range res.AccountProof {
		nodeKey := encodedNode
		if len(encodedNode) >= 32 { // small MPT nodes are not hashed
			nodeKey = crypto.Keccak256(encodedNode)
		}
		if err := db.Put(nodeKey, encodedNode); err != nil {
			return fmt.Errorf("failed to load account proof node %d into mem db: %w", i, err)
		}
	}

	key := crypto.Keccak256Hash(res.Address[:])
	trieDB := trie.NewDatabase(db)

	// wrap our DB of trie nodes with a Trie interface, and anchor it at the trusted state root
	proofTrie, err := trie.New(trie.StateTrieID(stateRoot), trieDB)
	if err != nil {
		return fmt.Errorf("failed to load db wrapper around account trie kv store: %w", err)
	}

	// now get the full value from the account proof, and check that it matches the JSON contents
	accountProofValue, err := proofTrie.TryGet(key[:])
	if err != nil {
		return fmt.Errorf("failed to retrieve account value: %w", err)
	}

	if !bytes.Equal(accountClaimedValue, accountProofValue) {
		return fmt.Errorf("L1 RPC is tricking us, account proof does not match provided deserialized values:\n"+
			"  claimed: %x\n"+
			"  proof:   %x", accountClaimedValue, accountProofValue)
	}
	return err
}
