import { GenericMerkleIntervalTree, GenericMerkleIntervalTreeNode } from './'
import { AbiStateUpdate } from '../'
import { MerkleIntervalTreeNode, MerkleIntervalInclusionProof } from '../../types';

export class MerkleStateIntervalTree extends GenericMerkleIntervalTree {
  public static STATE_ID_LENGTH = 16

  public generateLeafNode(dataBlock: AbiStateUpdate): GenericMerkleIntervalTreeNode {
    return MerkleStateIntervalTree.calculateStateUpdateLeaf(dataBlock)
  }

  // To create a state update tree from the generic interval tree,
  // we simply define how to generate a leaf from its SU data block.
  public static calculateStateUpdateLeaf(stateUpdate: AbiStateUpdate): GenericMerkleIntervalTreeNode {
    const hash = GenericMerkleIntervalTree.hash(Buffer.from(stateUpdate.encoded))
    const index = stateUpdate.range.start.toBuffer(
      'be',
      MerkleStateIntervalTree.STATE_ID_LENGTH
    )
    return new GenericMerkleIntervalTreeNode(hash, index)
  }

  // For a state interval tree to be valid, we have the additional condition that
  // the SU.end is less than its inclusion proof's implicitEnd.  This function checks for that,
  // and returns the root which it results in to be verified in the assetId tree.
  public static verifyExectedRoot(
    stateUpdate: AbiStateUpdate,
    inclusionProof: MerkleIntervalInclusionProof,
  ): MerkleIntervalTreeNode {
    const leafNode: MerkleIntervalTreeNode = MerkleStateIntervalTree.calculateStateUpdateLeaf(stateUpdate)
    const rootAndBound = GenericMerkleIntervalTree.getRootAndBounds(
      leafNode,
      inclusionProof
    )
    // Check that the bound agrees with the end.
    if (stateUpdate.range.end.gt(rootAndBound.maxEnd)) {
      throw new Error('State Update range.end exceeds the max for its inclusion proof.')
    }
    return rootAndBound.root
  }
}
