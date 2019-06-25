import { MerkleIntervalTree, GenericMerkleIntervalTreeNode } from './'
import { AbiStateUpdate } from '../'

export class MerkleStateIntervalTree extends MerkleIntervalTree {
  public static STATE_ID_LENGTH = 16

  // To create a state update tree from the generic interval tree,
  // we simply define how to generate a leaf from its SU data block.
  public generateLeafNode(stateUpdate: AbiStateUpdate): GenericMerkleIntervalTreeNode {
    const hash = MerkleIntervalTree.hash(Buffer.from(stateUpdate.encoded))
    const index = stateUpdate.range.start.toBuffer(
      'be',
      MerkleStateIntervalTree.STATE_ID_LENGTH
    )
    return new GenericMerkleIntervalTreeNode(hash, index)
  }
}
