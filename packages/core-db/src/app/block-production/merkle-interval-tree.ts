/* External Imports */
import { BigNumber, keccak256, reverse, ZERO } from '@pigi/core-utils'

/* Internal Imports */
import {
  MerkleIntervalInclusionProof,
  MerkleIntervalProofOutput,
  MerkleIntervalTree,
  MerkleIntervalTreeNode,
} from '../../types/block-production'

export class GenericMerkleIntervalTreeNode implements MerkleIntervalTreeNode {
  public data: Buffer

  constructor(readonly hash: Buffer, readonly lowerBound: Buffer) {
    this.data = Buffer.concat([this.hash, this.lowerBound])
  }
}

/**
 * Computes the index of the sibling of a node in some level.
 * @param index Index of a node.
 * @returns the index of the sibling of that node.
 */
const getSiblingIndex = (index: number): number => {
  return index + (index % 2 === 0 ? 1 : -1)
}

/**
 * Computes the index of the parent of a node at the level above.
 * @param index Index of a node.
 * @returns the index of the parent of that node.
 */
const getParentIndex = (index: number): number => {
  return index === 0 ? 0 : Math.floor(index / 2)
}

export class GenericMerkleIntervalTree implements MerkleIntervalTree {
  public levels: GenericMerkleIntervalTreeNode[][] = [[]]
  public numLeaves: number

  constructor(readonly dataBlocks: any) {
    // Store the number of leaves so that generation can use it.
    this.parseNumLeaves()
    // Convert the data blocks into leaf nodes so that the tree can be built.
    this.generateLeafNodes()
    // Build the remaining levels of the tree.
    this.generateInternalNodes()
  }

  public root(): GenericMerkleIntervalTreeNode {
    return this.levels[this.levels.length - 1][0]
  }

  public static hash(value: Buffer): Buffer {
    return Buffer.from(keccak256(value.toString('hex')), 'hex')
  }
  /**
   * Computes the parent of two GenericMerkleIntervalTreeNode siblings in a tree.
   * @param left The left sibling to compute the parent of.
   * @param right The right sibling to compute the parent of.
   */
  public static parent(
    left: GenericMerkleIntervalTreeNode,
    right: GenericMerkleIntervalTreeNode
  ): GenericMerkleIntervalTreeNode {
    if (Buffer.compare(left.lowerBound, right.lowerBound) >= 0) {
      throw new Error(
        'Left lowerBound (0x' +
          left.lowerBound.toString('hex') +
          ') not less than right lowerBound (0x' +
          right.lowerBound.toString('hex') +
          ')'
      )
    }
    const concatenated = Buffer.concat([left.data, right.data])
    return new GenericMerkleIntervalTreeNode(
      GenericMerkleIntervalTree.hash(concatenated),
      left.lowerBound
    )
  }

  /**
   * Computes an "empty node" whose hash value is 0 and whose index is the max.
   * Used to pad a tree which has less  than 2^n nodes.
   * @param ofLength The length in bytes of the lowerBound value for the empty node.
   */
  public static emptyNode(ofLength: number): GenericMerkleIntervalTreeNode {
    const hash = Buffer.from(new Array(32).fill(0))
    const filledArray = new Array(ofLength).fill(255)
    const index = Buffer.from(filledArray)
    return new GenericMerkleIntervalTreeNode(hash, index)
  }

  /**
   * Returns the number of leaves the tree has.
   */
  private parseNumLeaves() {
    this.numLeaves = this.dataBlocks.length
  }

  /**
   * Calculates the leaf GenericMerkleIntervalTreeNode for a given data block.
   */
  public generateLeafNode(dataBlock: any): GenericMerkleIntervalTreeNode {
    return dataBlock
  }

  /**
   * Fills the bottom (level 0) of the tree by parsing each data block into a node.
   */
  public generateLeafNodes() {
    for (let i = 0; i < this.dataBlocks.length; i++) {
      this.levels[0][i] = this.generateLeafNode(this.dataBlocks[i])
    }
  }

  /**
   * Generates the other levels of the tree once the leaf nodes have been parsed.
   */
  private generateInternalNodes() {
    // Calculate the depth of the tree
    const numInternalLevels = Math.ceil(Math.log2(this.numLeaves))
    for (let level = 0; level < numInternalLevels; level++) {
      this.generateLevelAbove(level)
    }
  }

  /**
   * Calculates the number of nodes which will be used in a given level of the tree based on the number of leaves.
   * @param level the level of the tree, such that leaf nodes are at level 0, and the root is at the maximum level.
   */
  private calculateNumNodesinLevel(level: number) {
    return Math.ceil(this.numLeaves / 2 ** level)
  }

  /**
   * Generates and stores an individual level of the tree from its children.
   * @param level the level of the children nodes for which we are storing parents.
   */
  private generateLevelAbove(level: number) {
    this.levels[level + 1] = []
    const numNodesInLevel: number = this.calculateNumNodesinLevel(level)
    for (let i = 0; i < numNodesInLevel; i += 2) {
      const left = this.levels[level][i]
      const right =
        i + 1 === numNodesInLevel
          ? GenericMerkleIntervalTree.emptyNode(left.lowerBound.length)
          : this.levels[level][i + 1]
      const parent = GenericMerkleIntervalTree.parent(left, right)
      const parentIndex = getParentIndex(i)
      this.levels[level + 1][parentIndex] = parent
    }
  }

  /**
   * Gets an inclusion proof for the merkle interval tree.
   * @param leafPosition the index in the tree of the leaf we are generating a merkle proof for.
   */
  public getInclusionProof(leafPosition: number): MerkleIntervalInclusionProof {
    if (!(leafPosition in this.levels[0])) {
      throw new Error(
        'Leaf index ' + leafPosition + ' not in bottom level of tree'
      )
    }

    const inclusionProof: GenericMerkleIntervalTreeNode[] = []
    let parentIndex: number
    let siblingIndex = getSiblingIndex(leafPosition)
    for (let i = 0; i < this.levels.length - 1; i++) {
      const level = this.levels[i]
      const node =
        level[siblingIndex] ||
        GenericMerkleIntervalTree.emptyNode(level[0].lowerBound.length)
      inclusionProof.push(node)

      // Figure out the parent and then figure out the parent's sibling.
      parentIndex = getParentIndex(siblingIndex)
      siblingIndex = getSiblingIndex(parentIndex)
    }
    return {
      siblings: inclusionProof,
      leafPosition: new BigNumber(leafPosition),
    }
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
    leafNode: GenericMerkleIntervalTreeNode,
    inclusionProof: MerkleIntervalInclusionProof,
    rootHash: Buffer
  ): boolean {
    const rootAndBounds = GenericMerkleIntervalTree.getRootAndBounds(
      leafNode,
      inclusionProof
    )
    // Check that the roots match.
    if (Buffer.compare(rootAndBounds.root.hash, rootHash) !== 0) {
      throw new Error('Invalid Merkle Index Tree roothash.')
    } else {
      return true
    }
  }

  public static getRootAndBounds(
    leafNode: GenericMerkleIntervalTreeNode,
    inclusionProof: MerkleIntervalInclusionProof
  ): MerkleIntervalProofOutput {
    if (inclusionProof.leafPosition.lt(ZERO)) {
      throw new Error('Invalid leaf position.')
    }

    // Compute the path based on the leaf index.
    const path = reverse(
      new BigNumber(inclusionProof.leafPosition).toString(
        2,
        inclusionProof.siblings.length
      )
    )

    // Need the first right sibling to ensure
    // that the tree is monotonically increasing.
    const firstRightSiblingIndex = path.indexOf('0')
    const firstRightSibling =
      firstRightSiblingIndex >= 0
        ? inclusionProof.siblings[firstRightSiblingIndex]
        : undefined

    let computed: GenericMerkleIntervalTreeNode = leafNode
    let left: GenericMerkleIntervalTreeNode
    let right: GenericMerkleIntervalTreeNode
    for (let i = 0; i < inclusionProof.siblings.length; i++) {
      const sibling = inclusionProof.siblings[i]

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
          Buffer.compare(right.lowerBound, firstRightSibling.lowerBound) === -1
        ) {
          throw new Error(
            'Invalid Merkle Index Tree proof--potential intersection detected.'
          )
        }
      }

      computed = this.parent(left, right) // note: this checks left.index < right.index
    }

    const implicitEnd = firstRightSibling
      ? firstRightSibling.lowerBound
      : GenericMerkleIntervalTree.emptyNode(leafNode.lowerBound.length)
          .lowerBound // messy way to get the max index, TODO clean
    return {
      root: computed,
      upperBound: new BigNumber(implicitEnd),
    }
  }
}
