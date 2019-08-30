/* External Imports */
import { Mutex } from 'async-mutex'
import * as assert from 'assert'

import {
  BIG_ENDIAN,
  BigNumber,
  DB,
  HashFunction,
  MerkleTreeInclusionProof,
  MerkleTreeNode,
  SparseMerkleTree,
  TWO,
  ZERO,
} from '../../types'
import { keccak256 } from './crypto'

/**
 * SparseMerkleTree implementation built using the optimizations implemented by Vitalik
 * here: https://github.com/ethereum/research/blob/master/sparse_merkle_tree/new_bintrie_optimized.py
 *
 * Namely, no intermediate nodes are persisted between a leaf node and its first ancestor
 * with a non-zero-hash sibling. In order to not store these intermediate nodes but
 * still make the leaf reachable through traversal, this ancestor node's value is
 * stored as its leaf node descendent's hash as well as the key that represents
 * the path to the leaf starting from this ancestor.
 *
 * To indicate that a node's value is a pointer to a leaf node instead of the typical
 * value of (leftChildHash + rightChildHash), the value is stored as 65 bytes instead
 * of the typical 64 bytes, with the first byte being a dummy byte.
 */
export class OptimizedSparseMerkleTree implements SparseMerkleTree {
  public static readonly emptyBuffer: Buffer = new Buffer(32).fill('\x00')
  private static readonly siblingBuffer: Buffer = new Buffer(1).fill('\x00')
  private static readonly leafPointerByte: string = '\x01'

  private root: MerkleTreeNode
  private zeroHashes: Buffer[]
  private readonly treeMutex: Mutex = new Mutex()
  private readonly hashBuffer: Buffer = new Buffer(64)

  constructor(
    private readonly db: DB,
    rootHash?: Buffer,
    private readonly height: number = 160,
    private readonly hashFunction: HashFunction = keccak256
  ) {
    assert(!rootHash || rootHash.length === 32, 'Root hash must be 32 bytes')
    assert(height > 0, 'SMT height needs to be > 0')
    this.populateZeroHashesAndRoot(rootHash)
  }

  public async getRootHash(): Promise<Buffer> {
    const copy: Buffer = new Buffer(this.root.hash.length)
    this.root.hash.copy(copy)
    return copy
  }

  public async getLeaf(leafKey: BigNumber, rootHash?: Buffer): Promise<Buffer> {
    return this.treeMutex.runExclusive(async () => {
      if (!!rootHash && !rootHash.equals(this.root.hash)) {
        return undefined
      }

      const nodesInPath: MerkleTreeNode[] = await this.getNodesInPath(leafKey)
      if (!nodesInPath || !nodesInPath.length) {
        return undefined
      }
      const leaf: MerkleTreeNode = nodesInPath[nodesInPath.length]
      // Will only match if we were able to traverse all the way to the leaf
      return leaf.key.equals(leafKey) ? leaf.value : undefined
    })
  }

  public async verifyAndStore(
    inclusionProof: MerkleTreeInclusionProof
  ): Promise<boolean> {
    // There should be one sibling for every node except the root.
    if (inclusionProof.siblings.length !== this.height - 1) {
      return false
    }

    return this.treeMutex.runExclusive(async () => {
      const leafHash: Buffer = this.hashFunction(inclusionProof.value)
      if (!!(await this.getNode(leafHash, inclusionProof.key))) {
        return true
      }

      let intermediateZeroHashNode: boolean = true
      let child: MerkleTreeNode = this.createNode(
        leafHash,
        inclusionProof.value,
        inclusionProof.key
      )
      let parent: MerkleTreeNode = child
      const nodesToStore: MerkleTreeNode[] = [child]
      for (let parentDepth = this.height - 2; parentDepth >= 0; parentDepth--) {
        child = parent

        const childDepth: number = parentDepth + 1
        // Since there's no root sibling, each sibling is one index lower
        const childSiblingHash: Buffer = inclusionProof.siblings[childDepth - 1]
        parent = this.calculateParentNode(
          child,
          childSiblingHash,
          inclusionProof.key,
          parentDepth
        )

        intermediateZeroHashNode =
          intermediateZeroHashNode &&
          childSiblingHash.equals(this.zeroHashes[childDepth]) &&
          parentDepth !== 0 &&
          inclusionProof.siblings[parentDepth - 1].equals(
            this.zeroHashes[parentDepth]
          )

        // Don't store nodes that can be re-calculated from key, leaf, and zeroHashes
        if (intermediateZeroHashNode) {
          continue
        }

        // If there were any zero-hash intermediate nodes we didn't persist, make the current node a shortcut to the leaf.
        if (nodesToStore.length === 1 && parentDepth < this.height - 2) {
          parent = this.createLeafShortcutNode(
            parent.hash,
            inclusionProof.value,
            inclusionProof.key,
            parentDepth
          )
        }
        nodesToStore.push(parent)

        // Store sibling node, but don't overwrite it if it's in the db.
        const siblingNode: MerkleTreeNode = await this.createProofSiblingNodeIfDoesntExist(
          childSiblingHash,
          inclusionProof.key,
          childDepth
        )
        if (!!siblingNode) {
          nodesToStore.push(siblingNode)
        }
      }

      if (!parent.hash.equals(this.root.hash)) {
        return false
      }
      await Promise.all(
        (await this.getNodesInPath(inclusionProof.key)).map((n) =>
          this.db.del(this.getNodeID(n))
        )
      )

      // Root hash will not change, but it might have gone from a shortcut to regular node.
      this.root = parent

      await Promise.all(
        nodesToStore.map((n) => this.db.put(this.getNodeID(n), n.value))
      )
      return true
    })
  }

  public async update(key: BigNumber, value: Buffer): Promise<boolean> {
    return this.treeMutex.runExclusive(async () => {
      const nodesToUpdate: MerkleTreeNode[] = await this.getNodesInPath(key)
      if (!nodesToUpdate) {
        return false
      }

      const leaf: MerkleTreeNode = nodesToUpdate[nodesToUpdate.length - 1]
      const idsToDelete: Buffer[] = [this.getNodeID(leaf)]
      leaf.hash = this.hashFunction(value)
      leaf.value = value

      let updatedChild: MerkleTreeNode = leaf
      let depth: number = nodesToUpdate.length - 2 // -2 because this array also contains the leaf

      // If we're not updating all nodes, there's a shortcut node to update.
      if (this.height !== nodesToUpdate.length) {
        const ancestorHash: Buffer = this.getZeroHashAncestorFromLeaf(
          key,
          value,
          this.height - (nodesToUpdate.length - 1)
        )

        updatedChild = nodesToUpdate[depth]
        idsToDelete.push(this.getNodeID(updatedChild))
        // Update the node pointing to the leaf
        updatedChild.hash = ancestorHash
        updatedChild.value = this.createLeafShortcutNode(
          ancestorHash,
          value,
          key,
          depth--
        ).value
      }

      // Iteratively update all nodes from the leaf-pointer node up to the root
      for (; depth >= 0; depth--) {
        idsToDelete.push(this.getNodeID(nodesToUpdate[depth]))
        updatedChild = this.updateNode(
          nodesToUpdate[depth],
          updatedChild,
          key,
          depth
        )
      }

      await Promise.all([
        ...nodesToUpdate.map((n) => this.db.put(this.getNodeID(n), n.value)),
        ...idsToDelete.map((id) => this.db.del(id)),
      ])

      this.root = nodesToUpdate[0]
      return true
    })
  }

  /**
   * Gets the hash of the ancestor from the provided leaf value up numberOfZeroHashes levels.
   * all node hashes will be calculated as hash(last_calculated_hash + zero_hashes[depth]) or
   * hash(zero_hashes[depth] + last_calculated_hash) depending on the bit in the key
   * associated with the depth in question.
   *
   * @param key The full path to the leaf node
   * @param leafValue The leaf node value
   * @param numberOfZeroHashes The number of times to iterate
   * @returns the resulting ancestor hash
   */
  private getZeroHashAncestorFromLeaf(
    key: BigNumber,
    leafValue: Buffer,
    numberOfZeroHashes: number
  ): Buffer {
    let nodeHash: Buffer = this.hashFunction(leafValue)
    for (let i = numberOfZeroHashes; i > 0; i--) {
      const depth = this.height - 1 - (numberOfZeroHashes - i)
      nodeHash = this.calculateParentHash(
        nodeHash,
        this.zeroHashes[depth],
        key,
        depth - 1 // Since it's the parent depth
      )
    }
    return nodeHash
  }

  /**
   * Gets an array of MerkleTreeNodes starting at the root and iterating down to the leaf
   * following the path in the provided key. The returned array will omit any nodes that
   * are not persisted because they can be calculated from the leaf and the zeroHashes.
   *
   * NOTE: If the tree is modified in parallel with a call to this function,
   * results are non-deterministic.
   *
   * @param leafKey The key describing the path to the leaf in question
   * @returns The array of MerkleTreeNodes from root to leaf
   */
  private async getNodesInPath(leafKey: BigNumber): Promise<MerkleTreeNode[]> {
    if (!this.root || !this.root.hash) {
      return []
    }
    if (!this.root.value) {
      return [this.root]
    }

    let node: MerkleTreeNode = this.root
    const nodesToUpdate: MerkleTreeNode[] = [node]

    let depth
    for (depth = 0; depth < this.height - 1; depth++) {
      const childDepth: number = depth + 1
      switch (node.value.length) {
        case 64:
          // This is a standard node
          node = this.isLeft(leafKey, depth)
            ? await this.getNode(
                node.value.subarray(0, 32),
                this.getNodeKey(leafKey, childDepth)
              )
            : await this.getNode(
                node.value.subarray(32),
                this.getNodeKey(leafKey, childDepth)
              )
          break
        case 65:
          // This is a pointer to a leaf node hash (0x01 + key_as_buffer + node_hash)
          const storedLeafKey: BigNumber = OptimizedSparseMerkleTree.bufferToKey(
            node.value.subarray(1, 33),
            depth
          )

          if (
            this.isLeft(storedLeafKey, depth) !== this.isLeft(leafKey, depth)
          ) {
            // We want the non-shortcut child-node, which is a zero-hash
            node = await this.getNode(
              this.zeroHashes[depth],
              this.getNodeKey(leafKey, childDepth)
            )
          } else {
            // skip to the leaf
            node = await this.getNode(
              node.value.subarray(33),
              this.getNodeKey(leafKey, this.height - 1)
            )
            depth = this.height
          }

          break
        default:
          // This is malformed or a disconnected sibling node
          return undefined
      }
      if (!node || !node.value) {
        return nodesToUpdate
      }
      nodesToUpdate.push(node)
    }
    return nodesToUpdate
  }

  /**
   * Updates the provided MerkleTreeNode based on the provided updated child node.
   *
   * @param node The node to update
   * @param updatedChild The child of the node to update that has changed
   * @param key The key for the updated leaf
   * @param depth the depth of the
   * @returns A reference to the provided node to update
   */
  private updateNode(
    node: MerkleTreeNode,
    updatedChild: MerkleTreeNode,
    key: BigNumber,
    depth: number
  ): MerkleTreeNode {
    const isLeft: boolean = this.isLeft(key, depth)
    if (isLeft) {
      node.value.fill(updatedChild.hash, 0, 32)
    } else {
      node.value.fill(updatedChild.hash, 32)
    }
    node.hash = this.hashFunction(node.value)
    return node
  }

  /**
   * Gets the portion of the provided key that describes the path at and below
   * the provided depth as a buffer.
   *
   * @param key The key
   * @param depth The depth
   * @returns The resulting buffer of the key
   */
  private static keyToBuffer(key: BigNumber, depth: number): Buffer {
    return key.shiftLeft(depth).toBuffer(BIG_ENDIAN, 32)
  }

  /**
   * Converts the provided buffer to a key specifying how to reach a specific
   * leaf in a sub-tree of the provided depth.
   *
   * @param buffer The buffer that contains key information
   * @param depth The depth
   * @returns the key.
   */
  private static bufferToKey(buffer: Buffer, depth: number): BigNumber {
    return new BigNumber(buffer).shiftRightInPlace(depth)
  }

  /**
   * Creates a shortcut node that has a hash that results from a single leaf-node
   * iteratively hashed up the tree with the zero hashes values up to the provided depth,
   * and a value that contains the leaf hash for quick look-up.
   *
   * @param nodeHash The hash identifying the node to store.
   * @param leafValue The leaf value that this is a shortcut to
   * @param leafKey The key associated with the leaf
   * @param depth The depth of the node to create
   * @returns The MerkleTreeNode
   */
  private createLeafShortcutNode(
    nodeHash: Buffer,
    leafValue: Buffer,
    leafKey: BigNumber,
    depth: number
  ): MerkleTreeNode {
    const value: Buffer = new Buffer(65)
      .fill(OptimizedSparseMerkleTree.leafPointerByte, 0, 1)
      .fill(OptimizedSparseMerkleTree.keyToBuffer(leafKey, depth), 1, 33)
      .fill(this.hashFunction(leafValue), 33)
    return this.createNode(nodeHash, value, this.getNodeKey(leafKey, depth))
  }

  /**
   * Creates a Merkle Proof sibling node if a node with this hash has not already been stored
   * in the DB.
   *
   * NOTE: If the tree is modified in parallel with a call to this function,
   * results are non-deterministic.
   *
   * @param nodeHash The hash of the node to create if not already present.
   * @param leafKey The key detailing how to get to this node from the root
   * @param depth The depth of this node in the tree
   * @returns The created node if one was created or undefined if one already exists.
   */
  private async createProofSiblingNodeIfDoesntExist(
    nodeHash: Buffer,
    leafKey: BigNumber,
    depth: number
  ): Promise<MerkleTreeNode> {
    const nodeKey: BigNumber = this.getNodeKey(leafKey, depth)
    const node: MerkleTreeNode = await this.getNode(nodeHash, nodeKey)
    if (!!node) {
      return undefined
    }
    return this.createNode(
      nodeHash,
      OptimizedSparseMerkleTree.siblingBuffer,
      nodeKey
    )
  }

  /**
   * Gets the MerkleTreeNode with the provided hash from the DB, if one exists.
   *
   * @param nodeHash The node hash uniquely identifying the node
   * @param nodeKey The key identifying the location of the node in question
   * @returns The node, if one was found
   */
  private async getNode(
    nodeHash: Buffer,
    nodeKey: BigNumber
  ): Promise<MerkleTreeNode> {
    const value: Buffer = await this.db.get(
      this.getNodeIDFromHashAndKey(nodeHash, nodeKey)
    )
    if (!value) {
      return undefined
    }
    return this.createNode(nodeHash, value, nodeKey)
  }

  /**
   * Calculates the parent hash from the provided node and sibling, using the key and depth
   * to determine whether the node is the left node or the sibling is the left node.
   *
   * @param nodeHash The node hash used as 1/2 input to parent calculation
   * @param siblingHash The sibling node hash used as 1/2 input to parent calculation
   * @param key The key representing the path to this node
   * @param depth The depth of this node
   * @returns The parent hash
   */
  private calculateParentHash(
    nodeHash: Buffer,
    siblingHash: Buffer,
    key: BigNumber,
    depth: number
  ): Buffer {
    return this.isLeft(key, depth)
      ? this.hashFunction(
          this.hashBuffer.fill(nodeHash, 0, 32).fill(siblingHash, 32)
        )
      : this.hashFunction(
          this.hashBuffer.fill(siblingHash, 0, 32).fill(nodeHash, 32)
        )
  }

  /**
   * Calculates the parent hash from the provided node and sibling hash, using the key and depth
   * to determine whether the node is the left node or the sibling is the left node.
   *
   * @param node The node whose hash is used as 1/2 input to parent calculation
   * @param siblingHash The sibling node hash used as 1/2 input to parent calculation
   * @param leafKey The key representing the path to a leaf from which we started
   * @param depth The depth of this node
   * @returns The parent node
   */
  private calculateParentNode(
    node: MerkleTreeNode,
    siblingHash: Buffer,
    leafKey: BigNumber,
    depth: number
  ): MerkleTreeNode {
    const value = new Buffer(64)
    if (this.isLeft(leafKey, depth)) {
      this.hashBuffer
        .fill(node.hash, 0, 32)
        .fill(siblingHash, 32)
        .copy(value)
    } else {
      this.hashBuffer
        .fill(siblingHash, 0, 32)
        .fill(node.hash, 32)
        .copy(value)
    }

    return this.createNode(
      this.hashFunction(value),
      value,
      this.getNodeKey(leafKey, depth)
    )
  }

  /**
   * Populates the zero-hash array for each level of the Sparse Merkle Tree
   * and stores the resulting root.
   *
   * @param rootHash The optional root hash to assign the tree
   */
  private populateZeroHashesAndRoot(rootHash?: Buffer): void {
    const hashes: Buffer[] = [
      this.hashFunction(OptimizedSparseMerkleTree.emptyBuffer),
    ]

    for (let i = 1; i < this.height; i++) {
      hashes[i] = this.hashFunction(
        this.hashBuffer.fill(hashes[i - 1], 0, 32).fill(hashes[i - 1], 32)
      )
    }

    this.zeroHashes = hashes.reverse()
    this.root = this.createNode(rootHash || this.zeroHashes[0], undefined, ZERO)
  }

  /**
   * Helper function to create a MerkleTreeNode from the provided hash, value, and key
   *
   * @param hash The hash
   * @param value The value
   * @param key The key that describes how to get to this node from the tree root
   * @returns The resulting MerkleTreeNode
   */
  private createNode(
    hash: Buffer,
    value: Buffer,
    key: BigNumber
  ): MerkleTreeNode {
    return { hash, value, key }
  }

  /**
   * Determines whether or not the key at the provided depth points to the left child or right child.
   *
   * @param key The key
   * @param depth The depth
   * @returns true if the key points to the left child at the provided depth, false if right
   */
  private isLeft(key: BigNumber, depth: number): boolean {
    return key
      .shiftLeft(depth)
      .shiftRight(this.height - 2)
      .mod(TWO)
      .equals(ZERO)
  }

  private getNodeKey(leafKey: BigNumber, depth: number): BigNumber {
    return leafKey.shiftRight(this.height - depth - 1)
  }

  /**
   * Gets the unique ID for the provided node used for lookup in the DB.
   *
   * @param node The node in question
   */
  private getNodeID(node: MerkleTreeNode): Buffer {
    return this.getNodeIDFromHashAndKey(node.hash, node.key)
  }

  private getNodeIDFromHashAndKey(
    nodeHash: Buffer,
    nodeKey: BigNumber
  ): Buffer {
    return this.hashFunction(
      this.hashBuffer
        .fill(nodeHash, 0, 32)
        .fill(this.hashFunction(nodeKey.toBuffer(BIG_ENDIAN)), 32)
    )
  }
}
