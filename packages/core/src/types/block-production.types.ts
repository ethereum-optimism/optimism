/* External Imports */
import BigNumber = require('bn.js')

/* Internal Imports */
import { AbiStateUpdate } from '../app'

export interface MerkleIntervalTreeNode {
  hash: Buffer // Hash of the sibling or leaf data
  start: Buffer // The start interval value for this node
  data: Buffer // concatenation of (hash, index)
}

export type MerkleIntervalInclusionProof = MerkleIntervalTreeNode[]

export interface MerkleIntervalProofOutput {
  root: MerkleIntervalTreeNode
  maxEnd: BigNumber
}

export interface MerkleIntervalTree {
  dataBlocks: any
  levels: Array<Array<MerkleIntervalTreeNode>>
  root(): MerkleIntervalTreeNode
  getInclusionProof(leafposition: number): MerkleIntervalInclusionProof
}

export interface SubtreeContents {
  assetId: Buffer
  stateUpdates: AbiStateUpdate[]
}

export interface DoubleMerkleIntervalTree extends MerkleIntervalTree {
  dataBlocks: SubtreeContents
  subtrees: MerkleIntervalTree[]
  getStateUpdateInclusionProof(stateUpdatePosition: number, assetIdPosition: number): DoubleMerkleInclusionProof
}

export interface DoubleMerkleInclusionProof {
  stateTreeInclusionProof: MerkleIntervalInclusionProof
  assetTreeInclusionProof: MerkleIntervalInclusionProof
}