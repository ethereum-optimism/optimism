package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/merkle"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// VerifyMerkleProof verifies a merkle proof against the root hash and the leaf hash.
// Reference: https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#is_valid_merkle_branch
func VerifyMerkleProof(root, leaf common.Hash, index uint64, proof [merkle.BinaryMerkleTreeDepth]common.Hash) bool {
	value := leaf
	for i := 0; i < merkle.BinaryMerkleTreeDepth; i++ {
		if ((index >> i) & 1) == 1 {
			value = crypto.Keccak256Hash(proof[i][:], value[:])
		} else {
			value = crypto.Keccak256Hash(value[:], proof[i][:])
		}
	}
	return value == root
}

const (
	// GenProof generates a merkle proof for a given leaf index by reconstructing the merkle tree from the passed
	// leaves.
	genProof = "gen_proof"
)

var (
	rootAndProof, _ = abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "root", Type: "bytes32"},
		{Name: "proof", Type: "bytes32[]"},
	})

	merkleEncoder = abi.Arguments{
		{Type: rootAndProof},
	}
)

// DiffMerkle generates an abi-encoded `merkleTestCase` of a specified variant.
func DiffMerkle() {
	variant := os.Args[2]
	if len(variant) == 0 {
		log.Fatal("Must pass a variant to the merkle diff tester!")
	}

	switch variant {
	case genProof:
		if len(os.Args) < 5 {
			log.Fatal("Invalid arguments to `gen_proof` variant.")
		}

		rawLeaves, err := hexutil.Decode(os.Args[3])
		if err != nil {
			log.Fatal("Failed to decode leaves: ", err)
		}
		index, err := strconv.ParseInt(os.Args[4], 10, 64)
		if err != nil {
			log.Fatal("Failed to parse leaf index: ", err)
		}
		merkleTree := merkle.NewBinaryMerkleTree()

		// Append all leaves to the merkle tree.
		for i := 0; i < len(rawLeaves)/32; i++ {
			leaf := common.BytesToHash(rawLeaves[i<<5 : (i+1)<<5])
			merkleTree.AddLeaf(leaf)
		}

		// Generate the proof for the given index.
		proof := merkleTree.ProofAtIndex(uint64(index))

		// Generate the merkle root.
		root := merkleTree.RootHash()

		// Return "abi.encode(root, proof)"
		packed, err := merkleEncoder.Pack(struct {
			Root  common.Hash
			Proof [merkle.BinaryMerkleTreeDepth]common.Hash
		}{
			Root:  root,
			Proof: proof,
		})
		if err != nil {
			log.Fatal("Failed to ABI encode root and proof: ", err)
		}
		fmt.Print(hexutil.Encode(packed[32:]))
	default:
		log.Fatal("Invalid variant passed to merkle diff tester!")
	}
}
