/*Internal Imports */
import {
  GenericMerkleIntervalTree,
  GenericMerkleIntervalTreeNode,
  MerkleStateIntervalTree
} from './'
import { AbiStateUpdate } from '../'
import { SubtreeContents, MerkleIntervalProofOutput, DoubleMerkleIntervalTree, DoubleMerkleInclusionProof } from '../../types'

export class PlasmaBlock extends GenericMerkleIntervalTree implements DoubleMerkleIntervalTree {
  public subtrees: MerkleStateIntervalTree[]

  public generateLeafNodes() {
    this.subtrees = []
    super.generateLeafNodes()
  }

  // The "leaf node" for the plasma block is itself the root hash of a state update tree.
  // Thus, its data blocks are in fact entire subtrees.
  public generateLeafNode(subtree: SubtreeContents): GenericMerkleIntervalTreeNode {
    // Create a state subtree for these state updates.
    const merkleStateIntervalTree = new MerkleStateIntervalTree(
      subtree.stateUpdates
    )
    // Store the state subtree.
    this.subtrees.push(merkleStateIntervalTree)
    // Return a leaf node with the root of the state tree and an index of the depositAddress.
    return new GenericMerkleIntervalTreeNode(
      merkleStateIntervalTree.root().hash,
      subtree.assetId
    )
  }

  /**
   * Returns a double inclusion proof which demonstrates the existence of a state update within the plasma block.
   * @param stateUpdatePosition index of the state udpate in the state subtree of the block.
   * @param assetIdPosition index of the assetId in the top-level asset id of the block
   */
  public getStateUpdateInclusionProof(
    stateUpdatePosition: number,
    assetIdPosition: number
  ): DoubleMerkleInclusionProof {
    return {
      stateTreeInclusionProof: this.subtrees[assetIdPosition].getInclusionProof(
        stateUpdatePosition
      ),
      assetTreeInclusionProof: this.getInclusionProof(assetIdPosition),
    }
  }

  /**
   * Verifies a double inclusion proof which demonstrates the existence of a state update within the plasma block.
   * @param stateUpdate
   * @param stateTreeInclusionProof
   * @param stateUpdatePosition
   * @param assetTreeInclusionProof
   * @param assetIdPosition
   * @param blockRootHash
   */
  public static verifyStateUpdateInclusionProof(
    stateUpdate: AbiStateUpdate,
    stateTreeInclusionProof: GenericMerkleIntervalTreeNode[],
    stateUpdatePosition: number,
    assetTreeInclusionProof: GenericMerkleIntervalTreeNode[],
    assetIdPosition: number,
    blockRootHash: Buffer
  ): any {
    const leafNodeHash: Buffer = GenericMerkleIntervalTree.hash(
      Buffer.from(stateUpdate.encoded)
    )
    const leafNodeIndex: Buffer = stateUpdate.range.start.toBuffer(
      'be',
      MerkleStateIntervalTree.STATE_ID_LENGTH
    )
    const stateLeafNode: GenericMerkleIntervalTreeNode = new GenericMerkleIntervalTreeNode(
      leafNodeHash,
      leafNodeIndex
    )
    const stateUpdateRootAndBounds: MerkleIntervalProofOutput = GenericMerkleIntervalTree.getRootAndBounds(
      stateLeafNode,
      stateUpdatePosition,
      stateTreeInclusionProof
    )

    if (stateUpdateRootAndBounds.bounds.implicitEnd.lt(stateUpdate.range.end)) {
      throw new Error(
        'state update inclusion failed: inclusion proof bounds: ' +
          stateUpdateRootAndBounds.bounds +
          ' disagrees with SU range: ' +
          stateUpdate.range
      )
    }

    const addressLeafHash: Buffer = stateUpdateRootAndBounds.root.hash
    const addressLeafIndex: Buffer = Buffer.from(
      stateUpdate.depositAddress.slice(2),
      'hex'
    )
    const addressLeafNode: GenericMerkleIntervalTreeNode = new GenericMerkleIntervalTreeNode(
      addressLeafHash,
      addressLeafIndex
    )
    return GenericMerkleIntervalTree.verify(
      addressLeafNode,
      assetIdPosition,
      assetTreeInclusionProof,
      blockRootHash
    )
  }
}
