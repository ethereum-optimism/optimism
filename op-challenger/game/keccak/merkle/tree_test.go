package merkle

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestBinaryMerkleTree_AddLeaf(t *testing.T) {
	var tests []struct {
		name      string
		leafCount int
	}

	// Test only three leaf counts since this requires adding a lot of leaves.
	// To test more thoroughly, increase the divisor and run locally.
	for i := 0; i < MaxLeafCount; i += MaxLeafCount / 3 {
		tests = append(tests, struct {
			name      string
			leafCount int
		}{
			name:      fmt.Sprintf("AddLeaf-%d", i),
			leafCount: i,
		})
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tree := NewBinaryMerkleTree()
			expectedLeafHash := zeroHashes[BinaryMerkleTreeDepth-1]
			for i := 0; i < test.leafCount; i++ {
				input := ([types.BlockSize]byte)(bytes.Repeat([]byte{byte(i)}, types.BlockSize))
				lastLeaf := types.Leaf{
					Input:           input,
					Index:           big.NewInt(int64(i)),
					StateCommitment: common.Hash{},
				}
				tree.AddLeaf(lastLeaf)
				expectedLeafHash = lastLeaf.Hash()
			}
			leaf := tree.walkDownToLeafCount(tree.LeafCount)
			require.Equal(t, expectedLeafHash, leaf.Label)
		})
	}
}

func TestBinaryMerkleTree_RootHash(t *testing.T) {
	tests := []struct {
		name      string
		leafCount int
		rootHash  common.Hash
	}{
		{
			name:      "EmptyBinaryMerkleTree",
			leafCount: 0,
			rootHash:  common.HexToHash("2733e50f526ec2fa19a22b31e8ed50f23cd1fdf94c9154ed3a7609a2f1ff981f"),
		},
		{
			name:      "SingleLeaf",
			leafCount: 1,
			rootHash:  common.HexToHash("de8451f1c4f0153718b46951d0764a63e979fa13d496e709cceafcdbbe4ae68c"),
		},
		{
			name:      "TwoLeaves",
			leafCount: 2,
			rootHash:  common.HexToHash("caa0130e02ef997ebab07643394f7fa90767a68c49170669a9262573bfc46116"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tree := NewBinaryMerkleTree()
			for i := 0; i < test.leafCount; i++ {
				input := ([types.BlockSize]byte)(bytes.Repeat([]byte{byte(i)}, types.BlockSize))
				tree.AddLeaf(types.Leaf{
					Input:           input,
					Index:           big.NewInt(int64(i)),
					StateCommitment: common.Hash{},
				})
			}
			require.Equal(t, test.rootHash, tree.RootHash())
		})
	}
}

func TestBinaryMerkleTree_ProofAtIndex(t *testing.T) {
	proof := Proof{}
	for i := 0; i < BinaryMerkleTreeDepth; i++ {
		proof[i] = common.Hash{}
	}
	tests := []struct {
		name      string
		leafCount int
		index     int
		proof     Proof
	}{
		{
			name:      "EmptyBinaryMerkleTree",
			leafCount: 0,
			index:     0,
			proof:     proof,
		},
		{
			name:      "SingleLeaf",
			leafCount: 1,
			index:     0,
			proof: Proof{
				common.HexToHash("0000000000000000000000000000000000000000000000000000000000000000"),
				common.HexToHash("ad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"),
				common.HexToHash("b4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30"),
				common.HexToHash("21ddb9a356815c3fac1026b6dec5df3124afbadb485c9ba5a3e3398a04b7ba85"),
				common.HexToHash("e58769b32a1beaf1ea27375a44095a0d1fb664ce2dd358e7fcbfb78c26a19344"),
				common.HexToHash("0eb01ebfc9ed27500cd4dfc979272d1f0913cc9f66540d7e8005811109e1cf2d"),
				common.HexToHash("887c22bd8750d34016ac3c66b5ff102dacdd73f6b014e710b51e8022af9a1968"),
				common.HexToHash("ffd70157e48063fc33c97a050f7f640233bf646cc98d9524c6b92bcf3ab56f83"),
				common.HexToHash("9867cc5f7f196b93bae1e27e6320742445d290f2263827498b54fec539f756af"),
				common.HexToHash("cefad4e508c098b9a7e1d8feb19955fb02ba9675585078710969d3440f5054e0"),
				common.HexToHash("f9dc3e7fe016e050eff260334f18a5d4fe391d82092319f5964f2e2eb7c1c3a5"),
				common.HexToHash("f8b13a49e282f609c317a833fb8d976d11517c571d1221a265d25af778ecf892"),
				common.HexToHash("3490c6ceeb450aecdc82e28293031d10c7d73bf85e57bf041a97360aa2c5d99c"),
				common.HexToHash("c1df82d9c4b87413eae2ef048f94b4d3554cea73d92b0f7af96e0271c691e2bb"),
				common.HexToHash("5c67add7c6caf302256adedf7ab114da0acfe870d449a3a489f781d659e8becc"),
				common.HexToHash("da7bce9f4e8618b6bd2f4132ce798cdc7a60e7e1460a7299e3c6342a579626d2"),
			},
		},
		{
			name:      "ManyLeaves",
			leafCount: 20,
			index:     0,
			proof: Proof{
				common.HexToHash("1033d66892a2bbb829462e2ebf4d5d86f4a5cc4e41b62c9d1b98159ad3b6a6b6"),
				common.HexToHash("82918db2862f7e00087a3b96d66b0fddd9a667845dfea72c0553a89bd3c4b629"),
				common.HexToHash("2d5bbf3e663792ea111f05a3448a3aee485b408d689f817eaabfe9c740bcd598"),
				common.HexToHash("597dd11bdb5f6367416f3d44bb1f707dcf24e13e8429bcb6539d66e96203d472"),
				common.HexToHash("a063b1a2583a43114cd150b3c2320fe0bccf5f1b0f75c92c9d7e55433a291517"),
				common.HexToHash("0eb01ebfc9ed27500cd4dfc979272d1f0913cc9f66540d7e8005811109e1cf2d"),
				common.HexToHash("887c22bd8750d34016ac3c66b5ff102dacdd73f6b014e710b51e8022af9a1968"),
				common.HexToHash("ffd70157e48063fc33c97a050f7f640233bf646cc98d9524c6b92bcf3ab56f83"),
				common.HexToHash("9867cc5f7f196b93bae1e27e6320742445d290f2263827498b54fec539f756af"),
				common.HexToHash("cefad4e508c098b9a7e1d8feb19955fb02ba9675585078710969d3440f5054e0"),
				common.HexToHash("f9dc3e7fe016e050eff260334f18a5d4fe391d82092319f5964f2e2eb7c1c3a5"),
				common.HexToHash("f8b13a49e282f609c317a833fb8d976d11517c571d1221a265d25af778ecf892"),
				common.HexToHash("3490c6ceeb450aecdc82e28293031d10c7d73bf85e57bf041a97360aa2c5d99c"),
				common.HexToHash("c1df82d9c4b87413eae2ef048f94b4d3554cea73d92b0f7af96e0271c691e2bb"),
				common.HexToHash("5c67add7c6caf302256adedf7ab114da0acfe870d449a3a489f781d659e8becc"),
				common.HexToHash("da7bce9f4e8618b6bd2f4132ce798cdc7a60e7e1460a7299e3c6342a579626d2"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tree := NewBinaryMerkleTree()
			for i := 0; i < test.leafCount; i++ {
				input := ([types.BlockSize]byte)(bytes.Repeat([]byte{byte(i)}, types.BlockSize))
				tree.AddLeaf(types.Leaf{
					Input:           input,
					Index:           big.NewInt(int64(i)),
					StateCommitment: common.Hash{},
				})
			}
			proof, err := tree.ProofAtIndex(uint64(test.index))
			require.NoError(t, err)
			require.Equal(t, test.proof, proof)
		})
	}
}

func TestBinaryMerkleTree_VerifyMerkleProof(t *testing.T) {
	proof := [BinaryMerkleTreeDepth]common.Hash{}
	for i := 0; i < BinaryMerkleTreeDepth; i++ {
		proof[i] = common.Hash{}
	}
	tests := []struct {
		name      string
		leafCount int
		index     int
	}{
		{
			name:      "SingleLeaf",
			leafCount: 1,
			index:     0,
		},
		{
			name:      "TwoLeaves",
			leafCount: 2,
			index:     1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tree := NewBinaryMerkleTree()
			var lastLeaf types.Leaf
			for i := 0; i < test.leafCount; i++ {
				input := ([types.BlockSize]byte)(bytes.Repeat([]byte{byte(i)}, types.BlockSize))
				lastLeaf = types.Leaf{
					Input:           input,
					Index:           big.NewInt(int64(i)),
					StateCommitment: common.Hash{},
				}
				tree.AddLeaf(lastLeaf)
			}
			proof, err := tree.ProofAtIndex(uint64(test.index))
			require.NoError(t, err)
			require.True(t, tree.VerifyMerkleProof(uint64(test.index), lastLeaf, proof))
		})
	}
}
