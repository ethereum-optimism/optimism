package merkle

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// BinaryMerkleTreeDepth is the depth of the merkle tree.
const BinaryMerkleTreeDepth = 16

// Proof is a list of [common.Hash]s that prove the merkle inclusion of a leaf.
// These are the sibling hashes of the leaf's path from the root to the leaf.
type Proof [BinaryMerkleTreeDepth]common.Hash

var (
	// MaxLeafCount is the maximum number of leaves in the merkle tree.
	MaxLeafCount = 1<<BinaryMerkleTreeDepth - 1 // 2^16 - 1
	// zeroHashes is a list of empty hashes in the binary merkle tree, indexed by height.
	zeroHashes [BinaryMerkleTreeDepth]common.Hash
	// rootHash is the known root hash of the empty binary merkle tree.
	rootHash common.Hash
)

func init() {
	// Initialize the zero hashes. These hashes are pre-computed for the starting state of the tree, where all leaves
	// are equal to `[32]byte{}`.
	for height := 0; height < BinaryMerkleTreeDepth-1; height++ {
		rootHash = crypto.Keccak256Hash(rootHash[:], zeroHashes[height][:])
		zeroHashes[height+1] = crypto.Keccak256Hash(zeroHashes[height][:], zeroHashes[height][:])
	}
	rootHash = crypto.Keccak256Hash(rootHash[:], zeroHashes[BinaryMerkleTreeDepth-1][:])
}

// merkleNode is a single node in the binary merkle tree.
type merkleNode struct {
	Label  common.Hash
	Parent *merkleNode
	Left   *merkleNode
	Right  *merkleNode
}

func (m *merkleNode) IsLeftChild(o *merkleNode) bool {
	return m.Left == o
}

func (m *merkleNode) IsRightChild(o *merkleNode) bool {
	return m.Right == o
}

// BinaryMerkleTree is a binary hash tree that uses the keccak256 hash function.
// It is an append-only tree, where leaves are added from left to right.
type BinaryMerkleTree struct {
	Root      *merkleNode
	LeafCount uint64
}

func NewBinaryMerkleTree() *BinaryMerkleTree {
	return &BinaryMerkleTree{
		Root:      &merkleNode{Label: rootHash},
		LeafCount: 0,
	}
}

// RootHash returns the root hash of the binary merkle tree.
func (m *BinaryMerkleTree) RootHash() (rootHash common.Hash) {
	return m.Root.Label
}

// walkDownToMaxLeaf walks down the tree to the max leaf node.
func (m *BinaryMerkleTree) walkDownToLeafCount(subtreeLeafCount uint64) *merkleNode {
	maxSubtreeLeafCount := uint64(MaxLeafCount) + 1
	levelNode := m.Root
	for height := 0; height < BinaryMerkleTreeDepth; height++ {
		if subtreeLeafCount*2 <= maxSubtreeLeafCount {
			if levelNode.Left == nil {
				levelNode.Left = &merkleNode{
					Label:  zeroHashes[height],
					Parent: levelNode,
				}
			}
			levelNode = levelNode.Left
		} else {
			if levelNode.Right == nil {
				levelNode.Right = &merkleNode{
					Label:  zeroHashes[height],
					Parent: levelNode,
				}
			}
			levelNode = levelNode.Right
			subtreeLeafCount -= maxSubtreeLeafCount / 2
		}
		maxSubtreeLeafCount /= 2
	}
	return levelNode
}

// AddLeaf adds a leaf to the binary merkle tree.
func (m *BinaryMerkleTree) AddLeaf(hash common.Hash) {
	// Walk down to the new max leaf node.
	m.LeafCount += 1
	levelNode := m.walkDownToLeafCount(m.LeafCount)

	// Set the leaf node data.
	levelNode.Label = hash

	// Walk back up the tree, updating the hashes with its sibling hash.
	for height := 0; height < BinaryMerkleTreeDepth; height++ {
		if levelNode.Parent.IsLeftChild(levelNode) {
			if levelNode.Parent.Right == nil {
				levelNode.Parent.Right = &merkleNode{
					Label:  zeroHashes[height],
					Parent: levelNode.Parent,
				}
			}
			levelNode.Parent.Label = crypto.Keccak256Hash(levelNode.Label[:], levelNode.Parent.Right.Label[:])
		} else {
			if levelNode.Parent.Left == nil {
				levelNode.Parent.Left = &merkleNode{
					Label:  zeroHashes[height],
					Parent: levelNode.Parent,
				}
			}
			levelNode.Parent.Label = crypto.Keccak256Hash(levelNode.Parent.Left.Label[:], levelNode.Label[:])
		}
		levelNode = levelNode.Parent
	}
}

// ProofAtIndex returns a merkle proof at the given leaf node index.
func (m *BinaryMerkleTree) ProofAtIndex(index uint64) (proof Proof) {
	if index >= uint64(MaxLeafCount) {
		panic("proof index out of bounds")
	}

	levelNode := m.walkDownToLeafCount(index + 1)
	for height := 0; height < BinaryMerkleTreeDepth; height++ {
		if levelNode.Parent.IsLeftChild(levelNode) {
			if levelNode.Parent.Right == nil {
				proof[height] = common.Hash{}
			} else {
				proof[height] = levelNode.Parent.Right.Label
			}
		} else {
			if levelNode.Parent.Left == nil {
				proof[height] = common.Hash{}
			} else {
				proof[height] = levelNode.Parent.Left.Label
			}
		}
		levelNode = levelNode.Parent
	}

	return proof
}
