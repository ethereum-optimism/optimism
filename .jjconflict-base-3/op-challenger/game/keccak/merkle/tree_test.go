package merkle

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/proofs.json
var refTests []byte

type testData struct {
	Name      string      `json:"name"`
	LeafCount uint64      `json:"leafCount"`
	RootHash  common.Hash `json:"rootHash"`
	Index     uint64      `json:"index"`
	Proofs    Proof       `json:"proofs"`
}

func TestBinaryMerkleTree_AddLeaf(t *testing.T) {
	var tests []testData
	require.NoError(t, json.Unmarshal(refTests, &tests))

	for i, test := range tests {
		test := test
		t.Run(fmt.Sprintf("%s-LeafCount-%v-Ref-%v", test.Name, test.LeafCount, i), func(t *testing.T) {
			tree := NewBinaryMerkleTree()
			expectedLeafHash := zeroHashes[BinaryMerkleTreeDepth-1]
			for i := 0; i < int(test.LeafCount); i++ {
				expectedLeafHash = leafHash(i)
				tree.AddLeaf(expectedLeafHash)
			}
			leaf := tree.walkDownToLeafCount(tree.LeafCount)
			require.Equal(t, expectedLeafHash, leaf.Label)
		})
	}
}

func TestBinaryMerkleTree_RootHash(t *testing.T) {
	var tests []testData
	require.NoError(t, json.Unmarshal(refTests, &tests))

	for i, test := range tests {
		test := test
		t.Run(fmt.Sprintf("%s-LeafCount-%v-Ref-%v", test.Name, test.LeafCount, i), func(t *testing.T) {
			tree := NewBinaryMerkleTree()
			for i := 0; i < int(test.LeafCount); i++ {
				tree.AddLeaf(leafHash(i))
			}
			require.Equal(t, test.RootHash, tree.RootHash())
		})
	}
}

func TestBinaryMerkleTree_ProofAtIndex(t *testing.T) {
	var tests []testData
	require.NoError(t, json.Unmarshal(refTests, &tests))

	for i, test := range tests {
		test := test
		t.Run(fmt.Sprintf("%s-Index-%v-Ref-%v", test.Name, test.LeafCount, i), func(t *testing.T) {
			tree := NewBinaryMerkleTree()
			for i := 0; i < int(test.LeafCount); i++ {
				tree.AddLeaf(leafHash(i))
			}
			proof := tree.ProofAtIndex(test.Index)
			require.Equal(t, test.Proofs, proof)
		})
	}
}

func leafHash(idx int) common.Hash {
	return common.Hash{0xff, byte(idx)}
}
