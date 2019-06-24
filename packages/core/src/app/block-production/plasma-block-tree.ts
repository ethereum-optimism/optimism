
import { MerkleIntervalTree, MerkleIntervalTreeNode, MerkleStateIntervalTree }  from './'
import { AbiStateUpdate } from '../'

export interface SubtreeContents {
    address: Buffer
    stateUpdates: AbiStateUpdate[]
}

export class PlasmaBlock extends MerkleIntervalTree {
public subtrees: MerkleStateIntervalTree[]

public parseLeaves() {
    this.subtrees = []
    super.parseLeaves()
}

public parseLeaf(subtree: SubtreeContents): MerkleIntervalTreeNode {
    const merkleStateIntervalTree = new MerkleStateIntervalTree(subtree.stateUpdates)
    this.subtrees.push(merkleStateIntervalTree)
    return new MerkleIntervalTreeNode(merkleStateIntervalTree.root().hash, subtree.address)
}

public getStateUpdateInclusionProof(
    stateUpdatePosition: number,
    addressPosition: number
): any {
    return {
    stateTreeInclusionProof: this.subtrees[addressPosition].getInclusionProof(stateUpdatePosition),
    addressTreeInclusionProof: this.getInclusionProof(addressPosition)
    }
}

public static verifyStateUpdateInclusionProof(
    stateUpdate: AbiStateUpdate,
    stateTreeInclusionProof: MerkleIntervalTreeNode[],
    stateUpdatePosition: number,
    addressTreeInclusionProof: MerkleIntervalTreeNode[],
    addressPosition: number,
    blockRootHash: Buffer
): any {
    const leafNodeHash: Buffer = MerkleIntervalTree.hash(Buffer.from(stateUpdate.encoded))
    const leafNodeIndex: Buffer = stateUpdate.range.start.toBuffer('be', MerkleStateIntervalTree.STATE_ID_LENGTH)
    const stateLeafNode: MerkleIntervalTreeNode = new MerkleIntervalTreeNode(leafNodeHash, leafNodeIndex)
    const stateUpdateRootAndBounds = MerkleIntervalTree.getRootAndBounds(
    stateLeafNode,
    stateUpdatePosition,
    stateTreeInclusionProof
    )

    const addressLeafHash: Buffer = stateUpdateRootAndBounds.root.hash
    const addressLeafIndex: Buffer = Buffer.from(stateUpdate.depositAddress.slice(2), 'hex')
    const addressLeafNode: MerkleIntervalTreeNode = new MerkleIntervalTreeNode(addressLeafHash, addressLeafIndex)
    return MerkleIntervalTree.verify(
    addressLeafNode,
    addressPosition,
    addressTreeInclusionProof,
    blockRootHash
    )
}
}