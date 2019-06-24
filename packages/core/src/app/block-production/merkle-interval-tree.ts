/* External Imports */
import BigNum = require('bn.js')
import debug from 'debug'
const log = debug('info:merkle-interval-tree')

/* Internal Imports */
import { reverse, keccak256, AbiStateUpdate } from '../'

const STATE_ID_LENGTH = 16

function getHash(value: Buffer) {
  return keccak256(value)
}

/**
 * Computes the index of the sibling of a node.
 * @param index Index of a node.
 * @returns the index of the sibling of that node.
 */
const getSiblingIndex = (index: number): number => {
  return index + (index % 2 === 0 ? 1 : -1)
}

/**
 * Computes the index of the parent of a node
 * @param index Index of a node.
 * @returns the index of the parent of that node.
 */
const getParentIndex = (index: number): number => {
  return index === 0 ? 0 : Math.floor(index / 2)
}

export class MerkleIntervalTreeNode {
  public data: Buffer

  constructor (readonly hash: Buffer, readonly index: Buffer) {
    this.data = Buffer.concat([this.hash, this.index])
  }
}

export class MerkleIntervalTree {
  public levels: MerkleIntervalTreeNode[][] = [[]]
  public numLeaves: number

  constructor (readonly dataBlocks: any) {
    this.parseNumLeaves()
    this.parseLeaves()
    this.generateFromLeaves()
  }

  public root(): MerkleIntervalTreeNode {
    return this.levels[this.levels.length - 1][0]
  }

  public static parent (left: MerkleIntervalTreeNode, right: MerkleIntervalTreeNode): MerkleIntervalTreeNode {
    if (Buffer.compare(left.index, right.index) >= 0) {
      throw new Error('Left index (0x' + left.index.toString('hex') + ') not less than right index (0x' + right.index.toString('hex') + ')')
    }
    const concatenated = Buffer.concat([left.data, right.data])
    return new MerkleIntervalTreeNode(getHash(concatenated), left.index)
  }

  public static emptyNode (ofLength: number): MerkleIntervalTreeNode {
    const hash = Buffer.from(new Array(32).fill(0))
    const filledArray = new Array(ofLength).fill(255)
    const index = Buffer.from(filledArray)
    return new MerkleIntervalTreeNode(hash, index)
  }

  private parseNumLeaves() {
    this.numLeaves = this.dataBlocks.length
  }

  public parseLeaf(dataBlock: any): MerkleIntervalTreeNode {
    return dataBlock
  }

  public parseLeaves() {
    for (let i = 0; i < this.dataBlocks.length; i++) {
      this.levels[0][i] = this.parseLeaf(this.dataBlocks[i])
    }
  }

  private generateFromLeaves() {
    // Calculate the depth of the tree
    const numInternalLevels = Math.ceil(Math.log2(this.numLeaves))
    for (let level = 0; level < numInternalLevels; level++) {
      this.generateLevel(level)
    }
  }

  // leaves are level 0 in this model, so that level = height - depth
  private calculateNumNodesinLevel(level: number) {
    return Math.ceil(this.numLeaves / (2**level))
  }

  private generateLevel(level: number) {
    this.levels[level+1] = []
    const numNodesInLevel: number = this.calculateNumNodesinLevel(level)
    for (let i = 0; i < numNodesInLevel; i += 2) {
      const left = this.levels[level][i]
      const right = 
        i + 1 === numNodesInLevel ? MerkleIntervalTree.emptyNode(left.index.length) : this.levels[level][i + 1]
      const parent = MerkleIntervalTree.parent(left, right)
      const parentIndex = getParentIndex(i)
      this.levels[level+1][parentIndex] = parent
    }
  }

  public getInclusionProof(leafPosition: number): MerkleIntervalTreeNode[] {
    if (!(leafPosition in this.levels[0])) {
      throw new Error('Leaf index ' + leafPosition + ' not in bottom level of tree')
    }

    const inclusionProof: MerkleIntervalTreeNode[] = []
    let parentIndex: number
    let siblingIndex = getSiblingIndex(leafPosition)
    for (let i = 0; i < this.levels.length - 1; i++) {
      const level = this.levels[i]
      const node = level[siblingIndex] || MerkleIntervalTree.emptyNode(level[0].index.length)
      inclusionProof.push(node)

      // Figure out the parent and then figure out the parent's sibling.
      parentIndex = getParentIndex(siblingIndex)
      siblingIndex = getSiblingIndex(parentIndex)
    }
    return inclusionProof
  }

  /**
   * Checks a Merkle proof.
   * @param leafNode Leaf node to check.
   * @param leafPosition Position of the leaf in the tree.
   * @param inclusionProof Inclusion proof for that transaction.
   * @param root The root node of the tree to check.
   * @returns the implicit bounds covered by the leaf if the proof is valid.
   */
  public static verify(
    leafNode: MerkleIntervalTreeNode,
    leafPosition: number,
    inclusionProof: MerkleIntervalTreeNode[],
    rootHash: Buffer
  ): any {
    const rootAndBounds = MerkleIntervalTree.getRootAndBounds(
      leafNode,
      leafPosition,
      inclusionProof
    )
    // Check that the roots match.
    if (Buffer.compare(rootAndBounds.root.hash, rootHash) !== 0) {
      throw new Error('Invalid Merkle Index Tree roothash.')
    } else {
      return rootAndBounds.bounds
    }
  }

  public static getRootAndBounds(
    leafNode: MerkleIntervalTreeNode,
    leafPosition: number,
    inclusionProof: MerkleIntervalTreeNode[],
  ): any {
    if (leafPosition < 0) {
      throw new Error('Invalid leaf position.')
    }

    // Compute the path based on the leaf index.
    const path = reverse(
      new BigNum(leafPosition).toString(2, inclusionProof.length)
    )

    // Need the first right sibling to ensure
    // that the tree is monotonically increasing.
    const firstRightSiblingIndex = path.indexOf('0')
    const firstRightSibling = 
      firstRightSiblingIndex >= 0
        ? inclusionProof[firstRightSiblingIndex]
        : undefined

    let computed: MerkleIntervalTreeNode = leafNode
    let left: MerkleIntervalTreeNode
    let right: MerkleIntervalTreeNode
    for (let i = 0; i < inclusionProof.length; i++) {
      const sibling = inclusionProof[i]

      if (path[i] === '1') {
        left = sibling
        right = computed
      } else {
        left = computed
        right = sibling

        // If some right node further up the tree
        // is less than the first right node,
        // the tree construction must be invalid.
        if (
            firstRightSibling && // if it's the last leaf in tree, this doesn't exist
            Buffer.compare(right.index, firstRightSibling.index) === -1)
          {
          throw new Error('Invalid Merkle Index Tree proof--potential intersection detected.')
        }
      }

      computed = this.parent(left, right) // note: this checks left.index < right.index
    }

    return {
      root: computed,
      bounds: {
        implicitStart: leafPosition == 0 ? new BigNum(0) : leafNode.index,
        implicitEnd: firstRightSibling ? firstRightSibling.index : MerkleIntervalTree.emptyNode(leafNode.index.length).index // messy way to get the max index, TODO clean
      }
    }
  }
}

export class MerkleStateIntervalTree extends MerkleIntervalTree {
  public parseLeaf(stateUpdate: AbiStateUpdate): MerkleIntervalTreeNode {
    const hash = getHash(Buffer.from(stateUpdate.encoded))
    const index = stateUpdate.range.start.toBuffer('be', STATE_ID_LENGTH)
    return new MerkleIntervalTreeNode(hash, index)
  }
}

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
    const leafNodeHash: Buffer = getHash(Buffer.from(stateUpdate.encoded))
    const leafNodeIndex: Buffer = stateUpdate.range.start.toBuffer('be', STATE_ID_LENGTH)
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