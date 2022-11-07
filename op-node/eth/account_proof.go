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

type AccountResult struct {
	AccountProof []hexutil.Bytes `json:"accountProof"`

	Address     common.Address `json:"address"`
	Balance     *hexutil.Big   `json:"balance"`
	CodeHash    common.Hash    `json:"codeHash"`
	Nonce       hexutil.Uint64 `json:"nonce"`
	StorageHash common.Hash    `json:"storageHash"`
	// storageProof field is ignored, we only need to proof the account contents,
	// we do not access any individual storage values.
}

// Verify an account proof from the getProof RPC. See https://eips.ethereum.org/EIPS/eip-1186
func (res *AccountResult) Verify(stateRoot common.Hash) error {
	accountClaimed := []interface{}{uint64(res.Nonce), (*big.Int)(res.Balance).Bytes(), res.StorageHash, res.CodeHash}
	accountClaimedValue, err := rlp.EncodeToBytes(accountClaimed)
	if err != nil {
		return fmt.Errorf("failed to encode account from retrieved values: %w", err)
	}

	// create a db with all trie nodes
	db := memorydb.New()
	for i, encodedNode := range res.AccountProof {
		nodeKey := crypto.Keccak256(encodedNode)
		if err := db.Put(nodeKey, encodedNode); err != nil {
			return fmt.Errorf("failed to load proof value %d into mem db: %w", i, err)
		}
	}

	key := crypto.Keccak256Hash(res.Address[:])
	trieDB := trie.NewDatabase(db)

	// wrap our DB of trie nodes with a Trie interface, and anchor it at the trusted state root
	proofTrie, err := trie.New(trie.StateTrieID(stateRoot), trieDB)
	if err != nil {
		return fmt.Errorf("failed to load db wrapper around kv store")
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
