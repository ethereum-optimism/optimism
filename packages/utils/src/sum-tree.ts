/* External Imports */
import BigNum = require('bn.js')

/* Internal Imports */
import { keccak256 } from './eth'
import { reverse } from './utils'
import { NULL_HASH } from './constants'

export interface ImplicitBounds {
  implicitStart: BigNum
  implicitEnd: BigNum
}

export interface MerkleTreeNode {
  end: BigNum
  data: string
}

export interface MerkleSumTreeOptions {
  leaves?: MerkleTreeNode[]
  hash?: (value: string) => string
  maxTreeSize?: BigNum
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
 * Computes the index of the sibling of a node.
 * @param index Index of a node.
 * @returns the index of the sibling of that node.
 */
const getSiblingIndex = (index: number): number => {
  return index + (index % 2 === 0 ? 1 : -1)
}

/**
 * Basic MerkleSumTree implementation.
 */
export class MerkleSumTree {
  private tree: MerkleTreeNode[][] = []
  private hash: (value: string) => string
  private maxTreeSize: BigNum

  constructor({
    hash = keccak256,
    leaves = [],
    maxTreeSize = new BigNum('ffffffffffffffffffffffffffffffff', 16),
  }: MerkleSumTreeOptions = {}) {
    this.hash = hash
    this.maxTreeSize = new BigNum(maxTreeSize, 'hex')
    this.generateTree(this.parseLeaves(leaves))
  }

  /**
   * @returns the root of the tree.
   */
  get root(): string {
    return this.tree[0].length > 0
      ? this.computeHash(this.tree[this.tree.length - 1][0])
      : null
  }

  /**
   * @returns the leaf nodes in the tree.
   */
  get leaves(): MerkleTreeNode[] {
    return this.tree[0]
  }

  /**
   * @returns all levels in the tree.
   */
  get levels(): MerkleTreeNode[][] {
    return this.tree
  }

  /**
   * Checks a Merkle proof.
   * @param leaf Leaf node to check.
   * @param leafIndex Position of the leaf in the tree.
   * @param inclusionProof Inclusion proof for that transaction.
   * @param root The root node of the tree to check.
   * @returns the implicit bounds covered by the leaf if the proof is valid.
   */
  public verify(
    leaf: MerkleTreeNode,
    leafIndex: number,
    inclusionProof: MerkleTreeNode[],
    root: string
  ): ImplicitBounds {
    if (leafIndex < 0) {
      throw new Error('Invalid leaf index.')
    }

    // Leaf data is unhashed, so hash it.
    leaf.data = this.hash(leaf.data)

    // Compute the path based on the leaf index.
    const path = reverse(
      new BigNum(leafIndex).toString(2, inclusionProof.length)
    )

    // Need the first left sibling to ensure
    // that the tree is monotonically increasing.
    const firstLeftSiblingIndex = path.indexOf('1')
    const firstLeftSibling =
      firstLeftSiblingIndex >= 0
        ? inclusionProof[firstLeftSiblingIndex]
        : undefined

    let computed = leaf
    let left: MerkleTreeNode
    let right: MerkleTreeNode
    for (let i = 0; i < inclusionProof.length; i++) {
      const sibling = inclusionProof[i]

      if (path[i] === '0') {
        left = computed
        right = sibling
      } else {
        left = sibling
        right = computed

        // Some left node further up the tree
        // is greater than the first left node
        // so tree construction must be invalid.
        if (left.end.gt(firstLeftSibling.end)) {
          throw new Error('Invalid Merkle Sum Tree proof.')
        }
      }

      // Values at left nodes must always be
      // less than values at right nodes.
      if (left.end.gt(right.end)) {
        throw new Error('Invalid Merkle Sum Tree proof.')
      }

      computed = this.computeParent(left, right)
    }

    // Check that the roots match.
    if (this.computeHash(computed) !== root) {
      throw new Error('Invalid Merkle Sum Tree proof.')
    }

    const isLastLeaf = new BigNum(2)
      .pow(new BigNum(inclusionProof.length))
      .subn(1)
      .eqn(leafIndex)

    return {
      implicitStart: firstLeftSibling ? firstLeftSibling.end : new BigNum(0),
      implicitEnd: isLastLeaf ? this.maxTreeSize : leaf.end,
    }
  }

  /**
   * Returns an inclusion proof for the leaf at a given index.
   * @param leafIndex Index of the leaf to generate a proof for.
   * @returns an inclusion proof for that leaf.
   */
  public getInclusionProof(leafIndex: number): MerkleTreeNode[] {
    if (leafIndex >= this.leaves.length || leafIndex < 0) {
      throw new Error('Invalid leaf index.')
    }

    const inclusionProof: MerkleTreeNode[] = []
    let parentIndex: number
    let siblingIndex = getSiblingIndex(leafIndex)
    for (let i = 0; i < this.tree.length - 1; i++) {
      const node = this.tree[i][siblingIndex] || this.createEmptyNode()
      inclusionProof.push(node)

      // Figure out the parent and then figure out the parent's sibling.
      parentIndex = getParentIndex(siblingIndex)
      siblingIndex = getSiblingIndex(parentIndex)
    }

    return inclusionProof
  }

  /**
   * @returns an empty Merkle tree node.
   */
  private createEmptyNode(): MerkleTreeNode {
    return {
      ...{
        end: this.maxTreeSize,
        data: NULL_HASH,
      },
    }
  }

  /**
   * Parses leaf nodes by hashing their data.
   * @param leaves Leaf nodes to parse.
   * @returns the parsed leaves.
   */
  private parseLeaves(leaves: MerkleTreeNode[]): MerkleTreeNode[] {
    return leaves.map((leaf) => {
      return {
        end: leaf.end,
        data: this.hash(leaf.data),
      }
    })
  }

  /**
   * Computes the unique hash for a node.
   * @param node Node to hash.
   * @returns the hash of the node.
   */
  private computeHash(node: MerkleTreeNode): string {
    const data = node.end.toString('hex') + node.data
    return this.hash(data)
  }

  /**
   * Computes the parent of two nodes.
   * @param left Left child node.
   * @param right Right child node.
   * @returns the parent of the two nodes.
   */
  private computeParent(
    left: MerkleTreeNode,
    right: MerkleTreeNode
  ): MerkleTreeNode {
    return {
      end: right.end,
      data: this.hash(this.computeHash(left) + this.computeHash(right)),
    }
  }

  /**
   * Recursively generates the Merkle tree.
   * @param children Nodes in the last generated level.
   */
  private generateTree(children: MerkleTreeNode[]): void {
    this.tree.push(children)
    if (children.length <= 1) {
      return
    }

    const parents: MerkleTreeNode[] = []
    for (let i = 0; i < children.length; i += 2) {
      const left = children[i]
      const right =
        i + 1 === children.length ? this.createEmptyNode() : children[i + 1]
      parents.push(this.computeParent(left, right))
    }

    this.generateTree(parents)
  }
}
