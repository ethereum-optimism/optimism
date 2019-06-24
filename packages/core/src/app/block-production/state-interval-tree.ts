import { MerkleIntervalTree, MerkleIntervalTreeNode }  from './'
import { AbiStateUpdate } from '../'

export class MerkleStateIntervalTree extends MerkleIntervalTree {
    public static STATE_ID_LENGTH = 16

    public parseLeaf(stateUpdate: AbiStateUpdate): MerkleIntervalTreeNode {
        const hash = MerkleIntervalTree.hash(Buffer.from(stateUpdate.encoded))
        const index = stateUpdate.range.start.toBuffer('be', MerkleStateIntervalTree.STATE_ID_LENGTH)
        return new MerkleIntervalTreeNode(hash, index)
    }
}