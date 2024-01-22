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
			name:      "SingleLeaf_ZeroIndex",
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
			name:      "SingleLeaf_FirstIndex",
			leafCount: 1,
			index:     1,
			proof: Proof{
				common.HexToHash("cfd23b6298abaea12ade48cd472295893b7facf37c92f425e50722a72ed084ac"),
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
			name:      "PartialTree_EvenMiddleIndex",
			leafCount: 20,
			index:     10,
			proof: Proof{
				common.HexToHash("30009b412aade7d8309511a4408ae2f4b72573dde601905693af9f3abb2e1dc8"),
				common.HexToHash("a7ca9b77295eadcfe6d58ef5e86e88432a0417b36ac6e4ab2d2e9d45702292e5"),
				common.HexToHash("0a8eb56a75e17742db02f5de94120005cfe95e26c80891de57e390b3e6a3ebc5"),
				common.HexToHash("c9909a93d2c0248ef490da737dabda9e41eb3d9a379ddb004cfe66c60f3072df"),
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
		{
			name:      "PartialTree_OddMiddleIndex",
			leafCount: 20,
			index:     11,
			proof: Proof{
				common.HexToHash("536c2dfcd1e4d209e13b2e17323dc3d71171114b4ea9481dcfaac0361eeaffae"),
				common.HexToHash("a7ca9b77295eadcfe6d58ef5e86e88432a0417b36ac6e4ab2d2e9d45702292e5"),
				common.HexToHash("0a8eb56a75e17742db02f5de94120005cfe95e26c80891de57e390b3e6a3ebc5"),
				common.HexToHash("c9909a93d2c0248ef490da737dabda9e41eb3d9a379ddb004cfe66c60f3072df"),
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
		{
			name:      "PartialTree_SecondLastIndex",
			leafCount: MaxLeafCount,
			index:     MaxLeafCount - 2,
			proof: Proof{
				common.HexToHash("a7ead3155af3540785a09632e1a9a0e905700dddc29ca3531643209d11abde34"),
				common.HexToHash("2510bc5d56f49d7f5cdce4e37424936b31f7a01ba27c0201b40bd00d7241f525"),
				common.HexToHash("69251e33d3beeac7d24417e095adf8994eca90f7b220477b7dbdf7833ed5c646"),
				common.HexToHash("37e281b275aa30e45a87ad2cfb8fbd6402fca47633dd698c519595e1360714dd"),
				common.HexToHash("5fdae72b17eea7ab12066760cfe8dff858665ece3b5ec7ea30fb88737bc63e00"),
				common.HexToHash("6657aa64e042b55b6915c5ac65c9f07e29e1e67466d159ab8432f15e7088ded6"),
				common.HexToHash("a689859b6b6f0b586d267b5228b3d07c169889b219a49231386c4ed6c934df6e"),
				common.HexToHash("b9d17d1a81bcc1e9434e517008e3f68e2e19ab73b464ca13885c9e3b1072fd0c"),
				common.HexToHash("af79e21764536f60b54298ba40da1d16fe432510c40d54e8d6d12f4c5491418d"),
				common.HexToHash("0071a9f78633e4bf4f008134694b97783ffdf7651bf09096bce74b1cc77c9254"),
				common.HexToHash("7d0475c42ed689be225ec7de3d6f620f5afa50d17223d3cf3b5faebe6b8eab63"),
				common.HexToHash("7efcb6ac59e3ebd6cb62985827c634e88a9cf118a906d2d25f0facf44f8ce48d"),
				common.HexToHash("0c2264f5c766dafd9ed23277449b2affe176e7ffeef29dad949932654df90f17"),
				common.HexToHash("0622a9974a31b7774c1b89e494663f008a82d525dd00ac6fbaf322ae9845261a"),
				common.HexToHash("882284d7f31dec4cadb4b46d2209ef7806e440d541f1d6efd2e1b53a32ee9e65"),
				common.HexToHash("fbcf92d99fd2e103e15b1647b177895b0529dd715e551a80c90744818458a8e3"),
			},
		},
		{
			name:      "FullTree_LastIndex",
			leafCount: MaxLeafCount,
			index:     MaxLeafCount - 1,
			proof: Proof{
				common.HexToHash("0000000000000000000000000000000000000000000000000000000000000000"),
				common.HexToHash("5d5992fd072e73c425c84efbd29e9e4a87756a67d972f096661c967583809c8f"),
				common.HexToHash("69251e33d3beeac7d24417e095adf8994eca90f7b220477b7dbdf7833ed5c646"),
				common.HexToHash("37e281b275aa30e45a87ad2cfb8fbd6402fca47633dd698c519595e1360714dd"),
				common.HexToHash("5fdae72b17eea7ab12066760cfe8dff858665ece3b5ec7ea30fb88737bc63e00"),
				common.HexToHash("6657aa64e042b55b6915c5ac65c9f07e29e1e67466d159ab8432f15e7088ded6"),
				common.HexToHash("a689859b6b6f0b586d267b5228b3d07c169889b219a49231386c4ed6c934df6e"),
				common.HexToHash("b9d17d1a81bcc1e9434e517008e3f68e2e19ab73b464ca13885c9e3b1072fd0c"),
				common.HexToHash("af79e21764536f60b54298ba40da1d16fe432510c40d54e8d6d12f4c5491418d"),
				common.HexToHash("0071a9f78633e4bf4f008134694b97783ffdf7651bf09096bce74b1cc77c9254"),
				common.HexToHash("7d0475c42ed689be225ec7de3d6f620f5afa50d17223d3cf3b5faebe6b8eab63"),
				common.HexToHash("7efcb6ac59e3ebd6cb62985827c634e88a9cf118a906d2d25f0facf44f8ce48d"),
				common.HexToHash("0c2264f5c766dafd9ed23277449b2affe176e7ffeef29dad949932654df90f17"),
				common.HexToHash("0622a9974a31b7774c1b89e494663f008a82d525dd00ac6fbaf322ae9845261a"),
				common.HexToHash("882284d7f31dec4cadb4b46d2209ef7806e440d541f1d6efd2e1b53a32ee9e65"),
				common.HexToHash("fbcf92d99fd2e103e15b1647b177895b0529dd715e551a80c90744818458a8e3"),
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
