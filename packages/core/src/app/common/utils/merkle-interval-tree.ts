/* External Imports */
import BigNumber = require('bn.js')

/* Internal Imports */
import {
  Range,
  MerkleIntervalTreeLeafNode,
  MerkleIntervalTreeInternalNode,
  MerkleIntervalTreeInclusionProof,
} from '../../../interfaces'
import {  bnMin, bnMax, except, reverse, keccak256 } from 'src/app'

/**
 * Computes the position of the sibling of a node.
 * @param position Position of a node.
 * @returns the position of the sibling of that node.
 */
const getSiblingPosition = (position: number): number => {
  return position + (position % 2 === 0 ? 1 : -1)
}

/**
 * Computes the position of the parent of a node
 * @param position Position of a node.
 * @returns the position of the parent of that node.
 */
const getParentPosition = (position: number): number => {
  return position === 0 ? 0 : Math.floor(position / 2)
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
 * Checks if a given position is out of bounds for an array.
 * @param list Array to check against.
 * @param index Index to check.
 * @returns `true` if the index is out of bounds, `false` otherwise.
 */
const outOfBounds = (list: any[], index: number): boolean => {
  return index < 0 || index >= list.length
}

/**
 * Merkle Interval Tree implementation.
 */
export class MerkleIntervalTree {
  private levels: MerkleIntervalTreeInternalNode[][]

  /**
   * Creates the tree.
   * @param leaves Leaves of the tree.
   * @param hashfn Hash function to use for the tree.
   */
  constructor(
    private leaves: MerkleIntervalTreeLeafNode[] = [],
    private hashfn: (value: Buffer) => Buffer = keccak256
  ) {
    this.validateLeaves(this.leaves)

    // Sort leaves by start value.
    this.leaves.sort((a, b) => {
      if (a.start.lt(b.start)) {
        return -1
      } else if (a.start.gt(b.start)) {
        return 1
      } else {
        return 0
      }
    })

    // Parse leaves into the first layer of the tree.
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
   * @param leafPosition Position of the leaf node in the list of leaves.
   * @returns an inclusion proof for the given leaf.
   */
  public getInclusionProof(
    leafPosition: number
  ): MerkleIntervalTreeInclusionProof {
    if (outOfBounds(this.leaves, leafPosition)) {
      throw new Error('Leaf position is out of bounds.')
    }

    // Set up some initial values.
    const inclusionProof: MerkleIntervalTreeInclusionProof = []
    let computedNodePosition = leafPosition
    let siblingNodePosition = getSiblingPosition(computedNodePosition)

    // Add an inclusion proof for each level in the tree.
    for (let i = 0; i < this.levels.length - 1; i++) {
      const currentLevel = this.levels[i]

      // Find the computed node and its sibling.
      const computedNode = currentLevel[computedNodePosition]
      const siblingNode = outOfBounds(currentLevel, siblingNodePosition)
        ? this.createEmptyNode(computedNode.index)
        : currentLevel[siblingNodePosition]

      // Add the sibling to the inclusion proof.
      inclusionProof.push(siblingNode)

      // Move up to the next level.
      computedNodePosition = getParentPosition(computedNodePosition)
      siblingNodePosition = getSiblingPosition(computedNodePosition)
    }

    return inclusionProof
  }

  /**
   * Gets the root and implicit bounds for a given leaf.
   * @param leafNode Leaf to get root and bounds for.
   * @param leafPosition Position of the leaf in the list of leaves.
   * @param inclusionProof Inclusion proof for the leaf.
   * @returns the root and bounds for the leaf.
   */
  public getRootAndBounds(
    leafNode: MerkleIntervalTreeLeafNode,
    leafPosition: number,
    inclusionProof: MerkleIntervalTreeInclusionProof
  ): { root: MerkleIntervalTreeInternalNode; bounds: Range } {
    this.validateLeaves([leafNode])

    if (leafPosition < 0) {
      throw new Error('Invalid leaf position.')
    }

    /*
     * Converts the position of the leaf node to a Merkle branch path.
     * Branch paths in a binary tree can be computed by turning the leaf
     * position into a binary string of a length equal to the height of the
     * tree (left padded with zeroes). A '0' or '1' represents moving down the
     * tree left or right, respectively. For example, the path of the 3rd leaf
     * in an 8-leaf tree (height 3) would be '010' (left, right left). Finally,
     * we need to reverse the string to get the path *up* the tree since we're
     * moving from the leaf to the root.
     */
    const path = reverse(
      new BigNumber(leafPosition).toString(2, inclusionProof.length)
    )

    // Parse the leaf to get the first internal node.
    let computedNode = this.parseLeaf(leafNode)

    // Set up some initial values
    let leftChild: MerkleIntervalTreeInternalNode
    let rightChild: MerkleIntervalTreeInternalNode
    let prevRightSibling: MerkleIntervalTreeInternalNode = null

    // Compute the root node from the inclusion proof.
    for (let i = 0; i < inclusionProof.length; i++) {
      const siblingNode = inclusionProof[i]

      if (path[i] === '1') {
        // Sibling is on the right.
        leftChild = siblingNode
        rightChild = computedNode
      } else {
        // Sibling is on the left.
        leftChild = computedNode
        rightChild = siblingNode

        /*
         * We have two conditions under which the elements of an inclusion
         * proof are invalid. First, the index of each right sibling **MUST**
         * be greater than the index of the previous right sibling. Second, the
         * index of each right sibling **MUST** be greater than the end value
         * of the leaf node.
         */
        if (
          (prevRightSibling !== null &&
            rightChild.index.lt(prevRightSibling.index)) ||
          rightChild.index.lt(leafNode.end)
        ) {
          throw new Error(
            'Invalid Merkle Interval Tree proof -- potential intersection detected.'
          )
        }

        prevRightSibling = rightChild
      }

      computedNode = this.computeParent(leftChild, rightChild)
    }

    /*
     * Each leaf node covers some range (given by start and end) explicitly.
     * However, there's empty space between the end of one range and the start
     * of the next. We define the "implicit" range of a given leaf as the start
     * of the range to the end of the next range. A valid inclusion proof for a
     * range gives us the property that no overlapping ranges with valid proofs
     * exist.
     *
     * We have special cases for the first last leaves in the tree. The
     * implicit range of the first leaf starts at zero, and the implicit range
     * of the last leaf ends at "null", signifying that it extends to the rest
     * of the tree.
     */

    /*
     * We start computing implicit ranges by finding the first right sibling in
     * the inclusion proof (first instance of a '1' in the path). If there's no
     * right sibling, then the node must be the last node in the tree.
     */
    const firstRightSiblingPosition = path.indexOf('1')
    const firstRightSibling =
      firstRightSiblingPosition >= 0
        ? inclusionProof[firstRightSiblingPosition]
        : null

    /*
     * Now we compute implicit start and end, taking care to consider the
     * special cases mentioned for the first and last leaves in the tree.
     */
    const implicitStart = leafPosition === 0 ? new BigNumber(0) : leafNode.start
    const implicitEnd =
      firstRightSibling !== null ? firstRightSibling.index : null

    return {
      root: computedNode,
      bounds: {
        start: implicitStart,
        end: implicitEnd,
      },
    }
  }

  /**
   * Checks an inclusion proof. Throws if the proof is invalid at any point.
   * @param leafNode Leaf node to check inclusion of.
   * @param leafPosition Position of the leaf in the list of leaves.
   * @param inclusionProof Inclusion proof for the leaf node.
   * @param rootHash Hash of the root of the tree.
   * @returns the "implicit range" covered by leaf node if the proof is valid.
   */
  public checkInclusionProof(
    leafNode: MerkleIntervalTreeLeafNode,
    leafPosition: number,
    inclusionProof: MerkleIntervalTreeInclusionProof,
    rootHash: Buffer
  ): boolean {
    const { root, bounds } = this.getRootAndBounds(
      leafNode,
      leafPosition,
      inclusionProof
    )

    return Buffer.compare(root.hash, rootHash) === 0
  }

  /**
   * Validates that a set of leaf nodes are valid by checking that there are no
   * overlapping leaves. Throws if any two leaves are overlapping.
   * @param leaves Set of leaf nodes to check.
   */
  private validateLeaves(leaves: MerkleIntervalTreeLeafNode[]): void {
    // Make sure that no two leaves intersect.
    const valid = leaves.every((leaf) => {
      const others = except(leaves, leaf)
      return (
        others.every((other) => {
          return !intersects(leaf, other)
        }) && leaf.start.lte(leaf.end)
      )
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
   * @param siblingPosition Position of the node's sibling.
   * @returns the empty node.
   */
  private createEmptyNode(
    siblingPosition: BigNumber
  ): MerkleIntervalTreeInternalNode {
    return {
      index: siblingPosition,
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

    // Tree is empty or we're at the root.
    if (children.length <= 1) {
      return
    }

    const parents: MerkleIntervalTreeInternalNode[] = []

    // Compute parent for each pair of children.
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
