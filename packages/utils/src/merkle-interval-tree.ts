/* External Imports */
import BigNumber = require('bn.js')

/* Internal Imports */
import { keccak256 } from './eth/utils'
import { bnMin, bnMax, except, reverse } from './utils'

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

/**
 * Checks if two ranges overlap.
 * @param rangeA First range to check.
 * @param rangeB Second range to check.
 * @returns `true` if the ranges overlap, `false` otherwise.
 */
const intersects = (rangeA: Range, rangeB: Range): boolean => {
  const maxStart = bnMax(rangeA.start, rangeB.start)
  const minEnd = bnMin(rangeA.end, rangeB.end)
  return maxStart.lt(minEnd)
}

/**
 * Checks if a given index is out of bounds for an array.
 * @param list Array to check against.
 * @param index Index to check.
 * @returns `true` if the index is out of bounds, `false` otherwise.
 */
const outOfBounds = (list: any[], index: number): boolean => {
  return index < 0 || index >= list.length
}

export interface Range {
  start: BigNumber
  end: BigNumber
}

export interface MerkleIntervalTreeLeafNode {
  start: BigNumber
  end: BigNumber
  data: Buffer
}

export interface MerkleIntervalTreeInternalNode {
  index: BigNumber
  hash: Buffer
}

export type MerkleIntervalTreeInclusionProof = MerkleIntervalTreeInternalNode[]

export class MerkleIntervalTree {
  private levels: MerkleIntervalTreeInternalNode[][]

  constructor(
    private leaves: MerkleIntervalTreeLeafNode[] = [],
    private hashfn: (value: Buffer) => Buffer = keccak256
  ) {
    this.validateLeaves(this.leaves)

    this.leaves.sort((a, b) => {
      if (a.start.lt(b.start)) {
        return -1
      } else if (a.start.gt(b.start)) {
        return 1
      } else {
        return 0
      }
    })

    const bottom = this.leaves.map((leaf) => {
      return this.parseLeaf(leaf)
    })

    this.levels = [bottom]
    this.generateTree()
  }

  /**
   * @returns the root of the tree.
   */
  public getRoot(): MerkleIntervalTreeInternalNode {
    return this.levels[0].length > 0
      ? this.levels[this.levels.length - 1][0]
      : null
  }

  /**
   * @returns the levels of the tree.
   */
  public getLevels(): MerkleIntervalTreeInternalNode[][] {
    return this.levels
  }

  /**
   * Generates an inclusion proof for a given leaf node.
   * @param leafPosition Index of the leaf node in the list of leaves.
   * @returns an inclusion proof for the given leaf.
   */
  public getInclusionProof(
    leafPosition: number
  ): MerkleIntervalTreeInclusionProof {
    if (outOfBounds(this.leaves, leafPosition)) {
      throw new Error('Leaf position is out of bounds.')
    }

    const inclusionProof: MerkleIntervalTreeInclusionProof = []
    let childIndex = leafPosition
    let siblingIndex = getSiblingIndex(childIndex)

    for (let i = 0; i < this.levels.length - 1; i++) {
      const currentLevel = this.levels[i]
      const childNode = currentLevel[childIndex]
      const siblingNode = outOfBounds(currentLevel, siblingIndex)
        ? currentLevel[siblingIndex]
        : this.createEmptyNode(childNode.index)

      inclusionProof.push(siblingNode)

      childIndex = getParentIndex(childIndex)
      siblingIndex = getSiblingIndex(childIndex)
    }

    return inclusionProof
  }

  /**
   * Checks an inclusion proof. Throws if the proof is invalid at any point.
   * @param leafNode Leaf node to check inclusion of.
   * @param leafPosition Index of the leaf in the list of leaves.
   * @param inclusionProof Inclusion proof for the leaf node.
   * @param rootHash Hash of the root of the tree.
   * @returns the "implicit range" covered by leaf node if the proof is valid.
   */
  public checkInclusionProof(
    leafNode: MerkleIntervalTreeLeafNode,
    leafPosition: number,
    inclusionProof: MerkleIntervalTreeInclusionProof,
    rootHash: Buffer
  ): Range {
    if (leafPosition < 0) {
      throw new Error('Invalid leaf index.')
    }

    const path = reverse(
      new BigNumber(leafPosition).toString(2, inclusionProof.length)
    )

    const firstRightSiblingIndex = path.indexOf('0')
    const firstRightSibling =
      firstRightSiblingIndex >= 0
        ? inclusionProof[firstRightSiblingIndex]
        : null

    let computedNode = this.parseLeaf(leafNode)
    let leftChild: MerkleIntervalTreeInternalNode
    let rightChild: MerkleIntervalTreeInternalNode

    for (let i = 0; i < inclusionProof.length; i++) {
      const siblingNode = inclusionProof[i]

      if (path[i] === '1') {
        leftChild = siblingNode
        rightChild = computedNode
      } else {
        leftChild = computedNode
        rightChild = siblingNode

        if (
          firstRightSibling !== null &&
          rightChild.index.lt(firstRightSibling.index)
        ) {
          throw new Error(
            'Invalid Merkle Interval Tree proof -- potential intersection detected.'
          )
        }
      }

      computedNode = this.computeParent(leftChild, rightChild)
    }

    if (Buffer.compare(computedNode.hash, rootHash) !== 0) {
      throw new Error(
        'Invalid Merkle Interval Tree proof -- invalid root hash.'
      )
    }

    const implicitStart = leafPosition === 0 ? new BigNumber(0) : leafNode.start
    const implicitEnd =
      firstRightSibling !== null ? firstRightSibling.index : null

    return {
      start: implicitStart,
      end: implicitEnd,
    }
  }

  /**
   * Validates that a set of leaf nodes are valid by checking that there are no
   * overlapping leaves. Throws if any two leaves are overlapping.
   * @param leaves Set of leaf nodes to check.
   */
  private validateLeaves(leaves: MerkleIntervalTreeLeafNode[]): void {
    const valid = leaves.every((leaf) => {
      const others = except(leaves, leaf)
      return others.every((other) => {
        return !intersects(leaf, other)
      })
    })

    if (!valid) {
      throw new Error('Merkle Interval Tree leaves must not overlap.')
    }
  }

  /**
   * Parses a leaf node into an internal node.
   * @param leaf Leaf to parse.
   * @returns the parsed internal node.
   */
  private parseLeaf(
    leaf: MerkleIntervalTreeLeafNode
  ): MerkleIntervalTreeInternalNode {
    return {
      index: leaf.start,
      hash: this.hashfn(
        Buffer.concat([
          leaf.start.toBuffer('be', 16),
          leaf.end.toBuffer('be', 16),
          leaf.data,
        ])
      ),
    }
  }

  /**
   * Creates an empty node for when there are an odd number of elements in a
   * specific layer in the tree.
   * @param siblingIndex Index of the node's sibling.
   * @returns the empty node.
   */
  private createEmptyNode(
    siblingIndex: BigNumber
  ): MerkleIntervalTreeInternalNode {
    return {
      index: siblingIndex,
      hash: this.hashfn(Buffer.from('0')),
    }
  }

  /**
   * Computes the parent of two internal nodes.
   * @param leftChild Left child of the parent.
   * @param rightChild Right child of the parent.
   * @returns the parent of the two children.
   */
  private computeParent(
    leftChild: MerkleIntervalTreeInternalNode,
    rightChild: MerkleIntervalTreeInternalNode
  ): MerkleIntervalTreeInternalNode {
    const data = Buffer.concat([
      leftChild.index.toBuffer('be', 16),
      leftChild.hash,
      rightChild.index.toBuffer('be', 16),
      rightChild.hash,
    ])
    const hash = this.hashfn(data)
    const index = leftChild.index

    return {
      index,
      hash,
    }
  }

  /**
   * Generates the tree recursively.
   */
  private generateTree(): void {
    const children = this.levels[this.levels.length - 1]

    if (children.length <= 1) {
      return
    }

    const parents: MerkleIntervalTreeInternalNode[] = []

    for (let i = 0; i < children.length; i += 2) {
      const leftChild = children[i]
      const rightChild = outOfBounds(children, i + 1)
        ? this.createEmptyNode(leftChild.index)
        : children[i + 1]
      const parent = this.computeParent(leftChild, rightChild)
      parents.push(parent)
    }

    this.levels.push(parents)
    this.generateTree()
  }
}
