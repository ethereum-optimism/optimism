package utils

import (
	"bytes"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"
)

// NormalizeProofResponse standardizes an AccountResult obtained from eth_getProof
// across different client implementations (e.g., Geth, Reth) so that they can be
// compared meaningfully in tests.
//
// Ethereum clients may encode empty or zeroed data structures differently while
// still representing the same logical state. For example:
//   - An empty storage proof may appear as [] (Geth) or ["0x80"] (Reth).
//
// This function normalizes such differences by:
//   - Converting single-element proofs containing "0x80" to an empty proof slice.
func NormalizeProofResponse(res *eth.AccountResult) {
	for i := range res.StorageProof {
		if len(res.StorageProof[i].Proof) == 1 && bytes.Equal(res.StorageProof[i].Proof[0], []byte{0x80}) {
			res.StorageProof[i].Proof = []hexutil.Bytes{}
		}
	}

	// Normalize empty CodeHash
	// Geth returns 0x0000000000000000000000000000000000000000000000000000000000000000
	// Reth returns 0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470
	if res.CodeHash == (common.Hash{}) {
		res.CodeHash = crypto.Keccak256Hash(nil)
	}

	// Normalize empty StorageHash
	// Geth returns 0x0000000000000000000000000000000000000000000000000000000000000000
	// Reth returns 0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421
	if res.StorageHash == (common.Hash{}) {
		res.StorageHash = types.EmptyRootHash
	}
}

// VerifyProof verifies an account and its storage proofs against a given state root.
//
// This function extends the standard behavior of go-ethereum’s AccountResult.Verify()
// by gracefully handling the case where the account’s storage trie root is empty
// (0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421).
func VerifyProof(res *eth.AccountResult, stateRoot common.Hash) error {
	// Skip storage proof verification if the storage trie is empty.
	if res.StorageHash != types.EmptyRootHash {
		for i, entry := range res.StorageProof {
			// load all MPT nodes into a DB
			db := memorydb.New()
			for j, encodedNode := range entry.Proof {
				nodeKey := encodedNode
				if len(encodedNode) >= 32 { // small MPT nodes are not hashed
					nodeKey = crypto.Keccak256(encodedNode)
				}
				if err := db.Put(nodeKey, encodedNode); err != nil {
					return fmt.Errorf("failed to load storage proof node %d of storage value %d into mem db: %w", j, i, err)
				}
			}
			path := crypto.Keccak256(entry.Key)
			val, err := trie.VerifyProof(res.StorageHash, path, db)
			if err != nil {
				return fmt.Errorf("failed to verify storage value %d with key %s (path %x) in storage trie %s: %w", i, entry.Key.String(), path, res.StorageHash, err)
			}
			if val == nil && entry.Value.ToInt().Cmp(common.Big0) == 0 { // empty storage is zero by default
				continue
			}
			comparison, err := rlp.EncodeToBytes(entry.Value.ToInt().Bytes())
			if err != nil {
				return fmt.Errorf("failed to encode storage value %d with key %s (path %x) in storage trie %s: %w", i, entry.Key.String(), path, res.StorageHash, err)
			}
			if !bytes.Equal(val, comparison) {
				return fmt.Errorf("value %d in storage proof does not match proven value at key %s (path %x)", i, entry.Key.String(), path)
			}
		}
	}

	accountClaimed := []any{uint64(res.Nonce), res.Balance.ToInt().Bytes(), res.StorageHash, res.CodeHash}
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
	path := crypto.Keccak256(res.Address[:])
	accountProofValue, err := trie.VerifyProof(stateRoot, path, db)
	if err != nil {
		return fmt.Errorf("failed to verify account value with key %s (path %x) in account trie %s: %w", res.Address, path, stateRoot, err)
	}

	// If the proof demonstrates non-existence (nil value), we must check if the RPC claimed the account is empty.
	if len(accountProofValue) == 0 {
		isEmpty := res.Nonce == 0 &&
			res.Balance.ToInt().Sign() == 0 &&
			(res.StorageHash == types.EmptyRootHash || res.StorageHash == common.Hash{}) &&
			(res.CodeHash == crypto.Keccak256Hash(nil) || res.CodeHash == common.Hash{})

		if isEmpty {
			return nil
		}
	}

	if !bytes.Equal(accountClaimedValue, accountProofValue) {
		return fmt.Errorf("L1 RPC is tricking us, account proof does not match provided deserialized values:\n"+
			"  claimed: %x\n"+
			"  proof:   %x", accountClaimedValue, accountProofValue)
	}
	return nil
}

// FetchAndVerifyProofs fetches account proofs from both L2EL and L2ELB for the given address
func FetchAndVerifyProofs(t devtest.T, sys *MixedOpProofPreset, address common.Address, slots []common.Hash, block uint64) {
	ctx := t.Ctx()
	blockInfo, err := sys.L2ELSequencerNode().Escape().L2EthClient().InfoByNumber(ctx, block)
	require.NoError(t, err, "failed to get block info for block %d", block)

	seqProofRes, err := sys.L2ELSequencerNode().Escape().L2EthClient().GetProof(ctx, address, slots, hexutil.Uint64(block).String())
	require.NoError(t, err, "failed to get proof from L2EL at block %d", block)

	valProofRes, err := sys.L2ELValidatorNode().Escape().L2EthClient().GetProof(ctx, address, slots, hexutil.Uint64(block).String())
	require.NoError(t, err, "failed to get proof from L2ELB at block %d", block)

	NormalizeProofResponse(seqProofRes)
	NormalizeProofResponse(valProofRes)

	require.Equal(t, seqProofRes, valProofRes, "sequencer and validator proofs should match")

	err = VerifyProof(seqProofRes, blockInfo.Root())
	require.NoError(t, err, "geth proof verification failed at block %d", block)

	err = VerifyProof(valProofRes, blockInfo.Root())
	require.NoError(t, err, "reth proof verification failed at block %d", block)
}
