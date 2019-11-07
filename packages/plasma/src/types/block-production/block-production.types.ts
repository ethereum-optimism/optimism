/* External Imports */
import { DoubleMerkleInclusionProof, MerkleIntervalTree } from '@pigi/core-db'

// Array of state updates for which a subtree is created in our block structure.
import { AbiStateUpdate } from '../../app/serialization'

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
