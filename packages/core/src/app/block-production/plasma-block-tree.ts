/*Internal Imports */
import {
  MerkleIntervalTree,
  MerkleIntervalTreeNode,
  MerkleStateIntervalTree,
} from './'
import { AbiStateUpdate } from '../'
import { SubtreeContents } from '../../types'

export class PlasmaBlock extends MerkleIntervalTree {
  public subtrees: MerkleStateIntervalTree[]

  public generateLeafNodes() {
    this.subtrees = []
    super.generateLeafNodes()
  }

  // The "leaf node" for the plasma block is itself the root hash of a state update tree.
  // Thus, its data blocks are in fact entire subtrees.
  public generateLeafNode(subtree: SubtreeContents): MerkleIntervalTreeNode {
    // Create a state subtree for these state updates.
    const merkleStateIntervalTree = new MerkleStateIntervalTree(
      subtree.stateUpdates
    )
    // Store the state subtree.
    this.subtrees.push(merkleStateIntervalTree)
    // Return a leaf node with the root of the state tree and an index of the depositAddress.
    return new MerkleIntervalTreeNode(
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
  ): any {
    return {
      stateTreeInclusionProof: this.subtrees[assetIdPosition].getInclusionProof(
        stateUpdatePosition
      ),
      addressTreeInclusionProof: this.getInclusionProof(assetIdPosition),
    }
  }

  /**
   * Verifies a double inclusion proof which demonstrates the existence of a state update within the plasma block.
   * @param stateUpdate
   * @param stateTreeInclusionProof
   * @param stateUpdatePosition
   * @param addressTreeInclusionProof
   * @param assetIdPosition
   * @param blockRootHash
   */
  public static verifyStateUpdateInclusionProof(
    stateUpdate: AbiStateUpdate,
    stateTreeInclusionProof: MerkleIntervalTreeNode[],
    stateUpdatePosition: number,
    addressTreeInclusionProof: MerkleIntervalTreeNode[],
    assetIdPosition: number,
    blockRootHash: Buffer
  ): any {
    const leafNodeHash: Buffer = MerkleIntervalTree.hash(
      Buffer.from(stateUpdate.encoded)
    )
    const leafNodeIndex: Buffer = stateUpdate.range.start.toBuffer(
      'be',
      MerkleStateIntervalTree.STATE_ID_LENGTH
    )
    const stateLeafNode: MerkleIntervalTreeNode = new MerkleIntervalTreeNode(
      leafNodeHash,
      leafNodeIndex
    )
    const stateUpdateRootAndBounds = MerkleIntervalTree.getRootAndBounds(
      stateLeafNode,
      stateUpdatePosition,
      stateTreeInclusionProof
    )

    const addressLeafHash: Buffer = stateUpdateRootAndBounds.root.hash
    const addressLeafIndex: Buffer = Buffer.from(
      stateUpdate.depositAddress.slice(2),
      'hex'
    )
    const addressLeafNode: MerkleIntervalTreeNode = new MerkleIntervalTreeNode(
      addressLeafHash,
      addressLeafIndex
    )
    return MerkleIntervalTree.verify(
      addressLeafNode,
      assetIdPosition,
      addressTreeInclusionProof,
      blockRootHash
    )
  }
}
