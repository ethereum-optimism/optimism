/* Internal Imports */
import { AbiStateUpdate, BigNumber } from '../app'

export interface MerkleIntervalTreeNode {
  hash: Buffer // Hash of the sibling or leaf data.
  lowerBound: Buffer // The interval lower bound value for this node.
  data: Buffer // Concatenation of (hash, index)
}

export interface MerkleIntervalInclusionProof {
  siblings: MerkleIntervalTreeNode[] // The siblings along the merkle path leading from the leaf to the root.
  leafPosition: BigNumber // The index of the leaf we are proving inclusion of.
}

export interface MerkleIntervalProofOutput {
  root: MerkleIntervalTreeNode // the root node resulting from a merkle index tree inclusion proof
  upperBound: BigNumber // The upper bound that an inclusion proof is valid for.
  // For a single MerkleIntervalTree, it is mathematically impossible for two branches to exist
  // such that their [leaf.lowerBound, proofOutput.upperBound) intersect.
}

export interface MerkleIntervalTree {
  dataBlocks: any // The blocks of data we are constructing a merkle interval tree for.
  levels: MerkleIntervalTreeNode[][] // The 'MerkleIntervalTreeNode's which make up the tree.
  // E.g. levels[0].length == numLeaves (the leaves), levels[levels.length-1].length == 1 (the root).
  root(): MerkleIntervalTreeNode
  getInclusionProof(leafposition: number): MerkleIntervalInclusionProof
}

// Array of state updates for which a subtree is created in our block structure.
export interface SubtreeContents {
  assetId: Buffer // Each subtree of state updates will be given a leaf.lowerBound = assetId in the top-level tree.
  stateUpdates: AbiStateUpdate[] // The state updates from which to build our subtree, whose root will become the leaf.hash = root() in the top-level tree.
}

// Our plasma blocks are a nested merkle interval tree.  See http://spec.plasma.group/en/latest/src/01-core/double-layer-tree.html
export interface DoubleMerkleIntervalTree extends MerkleIntervalTree {
  dataBlocks: SubtreeContents
  subtrees: MerkleIntervalTree[]
  getStateUpdateInclusionProof(
    stateUpdatePosition: number,
    assetIdPosition: number
  ): DoubleMerkleInclusionProof
}

export interface DoubleMerkleInclusionProof {
  stateTreeInclusionProof: MerkleIntervalInclusionProof
  assetTreeInclusionProof: MerkleIntervalInclusionProof
}
