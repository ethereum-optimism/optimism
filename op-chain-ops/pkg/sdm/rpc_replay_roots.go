package sdm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

type replayWitness struct {
	State []hexutil.Bytes `json:"state"`
	Keys  []hexutil.Bytes `json:"keys"`
}
type canonicalReplayAccount struct {
	Balance     *hexutil.Big    `json:"balance"`
	Nonce       *hexutil.Uint64 `json:"nonce"`
	StorageRoot *common.Hash    `json:"storageRoot"`
	CodeHash    *common.Hash    `json:"codeHash"`
}

// Reconstruct each transaction-boundary account trie from the parent witness
// and executor-backed accountAt records. Do not equate the final intermediate
// root with the block header: block finalization can legitimately change state.
func checkIntermediateRoots(ctx context.Context, rpc Caller, header *replayHeader) error {
	var roots []common.Hash
	if err := rpc.CallContext(ctx, &roots, "debug_intermediateRoots", header.Hash); err != nil {
		return err
	}
	if len(roots) != len(header.Transactions) {
		return fmt.Errorf("intermediate roots count %d, want %d", len(roots), len(header.Transactions))
	}
	var parent *struct {
		Hash      common.Hash `json:"hash"`
		StateRoot common.Hash `json:"stateRoot"`
	}
	if err := rpc.CallContext(ctx, &parent, "eth_getBlockByHash", header.ParentHash, false); err != nil {
		return err
	}
	if parent == nil || parent.Hash != header.ParentHash || parent.StateRoot == (common.Hash{}) {
		return errors.New("missing parent state root")
	}
	var witness replayWitness
	if err := rpc.CallContext(ctx, &witness, "debug_executionWitness", header.Hash); err != nil {
		return err
	}
	disk := rawdb.NewMemoryDatabase()
	defer disk.Close()
	for _, node := range witness.State {
		rawdb.WriteLegacyTrieNode(disk, crypto.Keccak256Hash(node), node)
	}
	db := triedb.NewDatabase(disk, triedb.HashDefaults)
	defer db.Close()
	tree, err := trie.New(trie.TrieID(parent.StateRoot), db)
	if err != nil {
		return fmt.Errorf("open witness-backed parent trie: %w", err)
	}
	addresses := make(map[common.Address]struct{})
	for _, key := range witness.Keys {
		if len(key) == common.AddressLength {
			addresses[common.BytesToAddress(key)] = struct{}{}
		} else if len(key) != common.HashLength {
			return errors.New("invalid witness key preimage length")
		}
	}
	if len(addresses) == 0 {
		return errors.New("witness has no account key preimages")
	}
	ordered := make([]common.Address, 0, len(addresses))
	for address := range addresses {
		ordered = append(ordered, address)
	}
	slices.SortFunc(ordered, func(a, b common.Address) int { return bytes.Compare(a[:], b[:]) })
	var failures []error
	for i, root := range roots {
		for _, address := range ordered {
			var account *canonicalReplayAccount
			if err := rpc.CallContext(ctx, &account, "debug_accountAt", header.Hash, i, address); err != nil {
				return err
			}
			key := crypto.Keccak256(address[:])
			encoded, err := encodeCanonicalAccount(account)
			if err != nil {
				return fmt.Errorf("accountAt %d/%s: %w", i, address, err)
			}
			if encoded == nil {
				err = tree.Delete(key)
			} else {
				err = tree.Update(key, encoded)
			}
			if err != nil {
				return fmt.Errorf("update witness trie %d/%s: %w", i, address, err)
			}
		}
		expected := tree.Hash()
		if root != expected {
			failures = append(failures, fmt.Errorf("root after tx[%d] %s, canonical reconstructed %s", i, root, expected))
		}
	}
	return errors.Join(failures...)
}

func encodeCanonicalAccount(account *canonicalReplayAccount) ([]byte, error) {
	if account == nil {
		return nil, nil
	}
	if account.Balance == nil || account.Nonce == nil || account.StorageRoot == nil || account.CodeHash == nil {
		return nil, errors.New("incomplete canonical account")
	}
	balance := new(uint256.Int)
	if overflow := balance.SetFromBig(account.Balance.ToInt()); overflow || account.Balance.ToInt().Sign() < 0 {
		return nil, errors.New("invalid account balance")
	}
	// Preserve existence: an explicitly returned empty account is not the same
	// as null (e.g. an untouched empty account inherited from genesis).
	return rlp.EncodeToBytes(&types.StateAccount{Nonce: uint64(*account.Nonce), Balance: balance, Root: *account.StorageRoot, CodeHash: account.CodeHash[:]})
}
