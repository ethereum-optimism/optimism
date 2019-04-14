/* External Imports */
import BigNum = require('bn.js')

/* Internal Imports */
import {
  KeyValueStore,
  ChainDB,
  Transaction,
  InclusionProof,
} from '../../../interfaces'
import {
  BaseKey,
  encode,
  decode,
  bnToUint256,
  getTransactionRangeEnd,
} from '../../common'

// TODO: Where should this stuff sit? Should it be customizeable?
const computeParent = (leftChild: string, rightChild: string): string => {
  return
}

/**
 * Checks whether a node is a left child.
 * @param nodeIndex Node index to check
 * @returns `true` if the node is a left child, `false` otherwise.
 */
const isLeftChild = (nodeIndex: number): boolean => {
  return nodeIndex % 2 === 1
}

/**
 * Computes the index of the parent of a node.
 * @param nodeIndex Index of the node to compute parent for.
 * @returns the index of the node's parent.
 */
const getParentIndex = (nodeIndex: number): number => {
  return Math.floor((nodeIndex - 1) / 2)
}

/**
 * Computes the index of the left child of a node.
 * @param nodeIndex Index of the node to compute left child for.
 * @returns the index of the node's left child.
 */
const getLeftChildIndex = (nodeIndex: number): number => {
  return 2 * nodeIndex + 1
}

/**
 * Computes the index of the right child of a node.
 * @param nodeIndex Index of the node to compute right child for.
 * @returns the index of the node's right child.
 */
const getRightChildIndex = (nodeIndex: number): number => {
  return 2 * nodeIndex + 2
}

/**
 * Computes the index of the sibling of a node.
 * @param nodeIndex Index of the node to compute sibling for.
 * @returns the index of the node's sibling.
 */
const getSiblingIndex = (nodeIndex: number): number => {
  return isLeftChild(nodeIndex) ? nodeIndex - 1 : nodeIndex + 1
}

/**
 * Computes the node index of a given leaf node. Used so that we can zero-index
 * leaf nodes `(0....2^n-1)` but still refer to the root node as `0`.
 * @param leafIndex Index of the leaf node to convert to a node index.
 * @param treeHeight Height of the binary tree.
 * @returns the index of the leaf node in the binary tree.
 */
const getLeafNodeIndex = (leafIndex: number, treeHeight: number): number => {
  return 2 ** treeHeight - 1 + leafIndex
}

/**
 * Computes the indices of each sibling node necessary to generate a Merkle
 * proof.
 * @param leafIndex Index of the leaf node to get siblings for.
 * @param treeHeight Height of the tree.
 * @returns the indices of each sibling going up the tree.
 */
const getMerkleSiblingIndices = (
  leafIndex: number,
  treeHeight: number
): number[] => {
  const siblingIndices: number[] = []
  let nodeIndex = getLeafNodeIndex(leafIndex, treeHeight)

  // Go until we're at the root.
  while (nodeIndex > 0) {
    // Compute the sibling and add it.
    const siblingIndex = getSiblingIndex(nodeIndex)
    siblingIndices.push(siblingIndex)

    // Go on to the parent.
    nodeIndex = getParentIndex(nodeIndex)
  }

  return siblingIndices
}

const KEYS = {
  BLOCKS: new BaseKey('b', ['uint32']),
  DEPOSITS: new BaseKey('d', ['uint256']),
  TRANSACTIONS: new BaseKey('t', ['uint32', 'uint256']),
  TREE_HEIGHTS: new BaseKey('h', ['uint32']),
  TREE_NODES: new BaseKey('n', ['uint32', 'uint32']),
  LEAF_INDICES: new BaseKey('l', ['uint32', 'uint256']),
}

/**
 * Basic ChainDB implementation that provides a nice interface to the chain
 * database.
 */
export class PGChainDB implements ChainDB {
  /**
   * Creates the wrapper.
   * @param db Database to interact with.
   */
  constructor(private db: KeyValueStore) {}

  /**
   * Queries transactions in a given range.
   * @param blockNumber Block to look in.
   * @param start Start of the range to query.
   * @param end End of the range to query.
   * @returns a list of transactions in that range.
   */
  public async getTransactions(
    blockNumber: number,
    start: BigNum,
    end: BigNum
  ): Promise<Transaction[]> {
    const iterator = this.db.iterator({
      gte: KEYS.TRANSACTIONS.encode([blockNumber, bnToUint256(start)]),
      lte: KEYS.TRANSACTIONS.encode([blockNumber, bnToUint256(end)]),
      values: true,
    })

    const values = await iterator.values()
    const transactions = values.map((value) => {
      return decode(value)
    })

    return transactions
  }

  /**
   * Queries deposits in a given range.
   * @param start Start of the range to query.
   * @param end End of the range to query.
   * @returns a list of deposits for the range.
   */
  public async getDeposits(start: BigNum, end: BigNum): Promise<Transaction[]> {
    const iterator = this.db.iterator({
      gte: KEYS.DEPOSITS.encode([bnToUint256(start)]),
      lte: KEYS.DEPOSITS.encode([bnToUint256(end)]),
    })

    const values = await iterator.values()
    const deposits = values.map((value) => {
      return decode(value)
    })

    return deposits
  }

  /**
   * Queries the block hash for a given block number.
   * @param blockNumber Block number to query.
   * @returns the hash of the given block.
   */
  public async getBlockHash(blockNumber: number): Promise<string> {
    const key = KEYS.BLOCKS.encode([blockNumber])
    const value = await this.db.get(key)
    return value.toString()
  }

  /**
   * Queries a specific stored Merkle tree node. Attempts to compute the node
   * from its children if the node doesn't exist.
   * @param blockNumber Block number to pull the node from.
   * @param nodeIndex Index of the specific node to query.
   * @returns the node or `null` if the node cannot be found.
   */
  public async getMerkleTreeNode(
    blockNumber: number,
    nodeIndex: number
  ): Promise<string> {
    // Reject attempts to query invalid nodes.
    const treeHeight = await this.getBlockTreeHeight(blockNumber)
    if (nodeIndex < 0 || nodeIndex >= treeHeight) {
      throw new Error('Invalid node index.')
    }

    const key = KEYS.TREE_NODES.encode([blockNumber, nodeIndex])
    const value = await this.db.get(key)

    // We have this node, return it.
    if (value !== null) {
      return value.toString()
    }

    // We don't have the node, but we might be able to compute the it from its
    // children. Recursively find each child and see if we can create the node.

    // We're at a leaf node so we can't go any deeper.
    if (nodeIndex > 2 ** treeHeight) {
      return null
    }

    // Compute left and right children indices.
    const leftChildIndex = getLeftChildIndex(nodeIndex)
    const rightChildIndex = getRightChildIndex(nodeIndex)

    // Recursively pull the left and right children.
    const leftChild = await this.getMerkleTreeNode(blockNumber, leftChildIndex)
    const rightChild = await this.getMerkleTreeNode(
      blockNumber,
      rightChildIndex
    )

    // We can't create a parent if either child is node.
    if (leftChild === null || rightChild === null) {
      return null
    }

    // We have both children so we can compute the parent!
    return computeParent(leftChild, rightChild)
  }

  /**
   * Adds a Merkle tree node for a specific block. Won't add the node if it
   * already exists in the database or can be computed from other nodes that
   * exist.
   * @param blockNumber Block to add the node for.
   * @param nodeIndex Index of the node to add.
   * @param nodeHash Hash of the node to add.
   */
  public async addMerkleTreeNode(
    blockNumber: number,
    nodeIndex: number,
    nodeHash: string
  ): Promise<void> {
    // Don't do anything if we already have the node.
    if ((await this.getMerkleTreeNode(blockNumber, nodeIndex)) !== null) {
      return
    }

    // Smart parent checks. We don't need to store the parent as long as we
    // have this node's sibling. Parent can simply be re-generated later on.
    // Skip if this is the root node because it doesn't have a sibling.
    if (nodeIndex !== 0) {
      const siblingIndex = getSiblingIndex(nodeIndex)
      if ((await this.getMerkleTreeNode(blockNumber, siblingIndex)) !== null) {
        const parentIndex = getParentIndex(nodeIndex)
        await this.removeMerkleTreeNode(blockNumber, parentIndex)
      }
    }

    // We don't have the node, add it.
    const key = KEYS.TREE_NODES.encode([blockNumber, nodeIndex])
    const value = Buffer.from(nodeHash)
    await this.db.put(key, value)
  }

  /**
   * Recursively deletes Merkle tree nodes. Intelligently deletes children
   * that could be used to re-generate the node.
   * @param blockNumber Block number to delete nodes from.
   * @param nodeIndex Index of the node to delete.
   */
  public async removeMerkleTreeNode(
    blockNumber: number,
    nodeIndex: number
  ): Promise<void> {
    // We don't have the node and we can't generate it from children.
    const node = await this.getMerkleTreeNode(blockNumber, nodeIndex)
    if (node === null) {
      return
    }

    // We have the node value, but we don't know whether we actually have the
    // value stored or if we have children that can re-generate the node.

    // First, check if we have the node in question.
    const key = KEYS.TREE_NODES.encode([blockNumber, nodeIndex])
    const value = await this.db.get(key)

    // We have this specific node, so operate on it.
    if (value !== null) {
      // Delete the node.
      await this.db.del(key)

      // Figure out if this node had a sibling. If so, we're going to want to
      // compute the parent and insert it since we'd be losing that info.
      const siblingIndex = getSiblingIndex(nodeIndex)
      const sibling = await this.getMerkleTreeNode(blockNumber, siblingIndex)

      // We have a sibling, so we're going to compute the parent and insert it.
      if (sibling !== null) {
        const parentIndex = getParentIndex(nodeIndex)
        const parent = isLeftChild(nodeIndex)
          ? computeParent(node, sibling)
          : computeParent(sibling, node)
        await this.addMerkleTreeNode(blockNumber, parentIndex, parent)
      }

      // Stop, don't need to delete anything else.
      return
    }

    // We don't have the node, but we have children that can be used to
    // re-generate it. Need to recursively delete any of these children.

    // We're at a leaf node so we can't go any deeper.
    const treeHeight = await this.getBlockTreeHeight(blockNumber)
    if (nodeIndex > 2 ** treeHeight) {
      return
    }

    // Compute left and right children.
    const leftChildIndex = getLeftChildIndex(nodeIndex)
    const rightChildIndex = getRightChildIndex(nodeIndex)

    // Delete the children.
    await this.removeMerkleTreeNode(blockNumber, leftChildIndex)
    await this.removeMerkleTreeNode(blockNumber, rightChildIndex)
  }

  /**
   * Smart method for removing Merkle proof nodes stored for a specific
   * transaction. Checks the ensure that deletion of the nodes wouldn't impact
   * other transaction proofs.
   * @param blockNumber Block number to delete nodes from.
   * @param leafIndex Index of the leaf to delete proof nodes for.
   */
  public async removeMerkleProofNodes(
    blockNumber: number,
    leafIndex: number
  ): Promise<void> {
    // Get the leaf indices of all other transactions in the same block.
    const otherLeafIndices: number[] = (await this.getAllLeafIndices(
      blockNumber
    )).filter((otherLeafIndex) => {
      return otherLeafIndex !== leafIndex
    })

    // Need the tree height to compute sibling/parent nodes.
    const treeHeight = await this.getBlockTreeHeight(blockNumber)

    // We don't want to delete any nodes that other transactions still need.
    // Compute the siblings for *all* other stored transactions so we're not
    // deleting anything critical.
    let allOtherSiblingIndices: number[] = []
    for (const otherLeafIndex of otherLeafIndices) {
      const otherSiblingIndices = getMerkleSiblingIndices(
        otherLeafIndex,
        treeHeight
      )
      allOtherSiblingIndices = allOtherSiblingIndices.concat(
        otherSiblingIndices
      )
    }

    // Now figure out which sibling indices we can safely get rid of by finding
    // anything that other nodes don't need. This is a safe operation because
    // `removeMerkleTreeNode` will insert any parent that could've been
    // computed by the removed node.
    const siblingIndices = getMerkleSiblingIndices(leafIndex, treeHeight)
    for (const siblingIndex of siblingIndices) {
      if (!allOtherSiblingIndices.includes(siblingIndex)) {
        await this.removeMerkleTreeNode(blockNumber, siblingIndex)
      }
    }
  }

  /**
   * Creates an inclusion proof for a given transaction.
   * @param transaction Transaction to create a proof for.
   * @returns the inclusion proof for that transaction.
   */
  public async getInclusionProof(
    transaction: Transaction
  ): Promise<InclusionProof> {
    const blockNumber = transaction.block
    const leafIndex = await this.getLeafIndex(transaction)
    const treeHeight = await this.getBlockTreeHeight(blockNumber)

    // Get the list of siblings up the tree.
    const siblingIndices: number[] = getMerkleSiblingIndices(
      leafIndex,
      treeHeight
    )

    // Get the nodes necessary to generate the proof.
    const proof: string[] = []
    for (const siblingIndex of siblingIndices) {
      const sibling = await this.getMerkleTreeNode(blockNumber, siblingIndex)

      // Don't have the sibling, can't compute the proof.
      if (sibling === null) {
        throw new Error('Cannot compute inclusion proof, missing sibling node.')
      }

      proof.push(sibling)
    }

    return proof
  }

  /**
   * Adds a block hash to the database.
   * @param blockNumber Block to add a hash for.
   * @param blockHash Hash to add.
   */
  public async addBlockHash(
    blockNumber: number,
    blockHash: string
  ): Promise<void> {
    const key = KEYS.BLOCKS.encode([blockNumber])
    const value = Buffer.from(blockHash)
    await this.db.put(key, value)
  }

  /**
   * Inserts the tree height for a given block.
   * @param blockNumber Block number to add tree height for.
   * @param treeHeight Height of the tree.
   */
  public async addBlockTreeHeight(
    blockNumber: number,
    treeHeight: number
  ): Promise<void> {
    const key = KEYS.TREE_HEIGHTS.encode([blockNumber])
    const value = Buffer.allocUnsafe(4)
    value.writeUInt32BE(treeHeight, 0)
    await this.db.put(key, value)
  }

  /**
   * Queries the tree height for a given block.
   * @param blockNumber Block number to query.
   * @returns the tree height for that block.
   */
  public async getBlockTreeHeight(blockNumber: number): Promise<number> {
    const key = KEYS.TREE_HEIGHTS.encode([blockNumber])
    const value = await this.db.get(key)
    return value.readUInt32BE(0)
  }

  /**
   * Adds a transaction to the database.
   * @param transaction Transaction to add.
   */
  public async addTransaction(transaction: Transaction): Promise<void> {
    const end = getTransactionRangeEnd(transaction)
    const key = KEYS.TRANSACTIONS.encode([transaction.block, end])
    const value = encode(transaction)
    await this.db.put(key, value)
  }

  /**
   * Sets the leaf index for a given transaction.
   * @param transaction Transaction to set.
   * @param leafIndex Leaf index for the transaction.
   */
  public async addLeafIndex(
    transaction: Transaction,
    leafIndex: number
  ): Promise<void> {
    const end = getTransactionRangeEnd(transaction)
    const key = KEYS.LEAF_INDICES.encode([transaction.block, end])
    const value = Buffer.allocUnsafe(4)
    value.writeUInt32BE(leafIndex, 0)
    await this.db.put(key, value)
  }

  /**
   * Gets the leaf index for a transaction.
   * @param transaction Transaction to query.
   * @returns the leaf index for that transaction.
   */
  public async getLeafIndex(transaction: Transaction): Promise<number> {
    const end = getTransactionRangeEnd(transaction)
    const key = KEYS.LEAF_INDICES.encode([transaction.block, end])
    const value = await this.db.get(key)
    return value.readUInt32BE(0)
  }

  /**
   * Gets all leaf indices for a given block.
   * @param blockNumber Block to get leaf indices for.
   * @returns all leaf indices for that block.
   */
  public async getAllLeafIndices(blockNumber: number): Promise<number[]> {
    const iterator = this.db.iterator({
      gte: KEYS.LEAF_INDICES.min([blockNumber]),
      lte: KEYS.LEAF_INDICES.max([blockNumber]),
    })

    const values = await iterator.values()
    return values.map((value) => {
      return value.readUInt32BE(0)
    })
  }

  /**
   * Adds a deposit to the database.
   * @param deposit Deposit to add.
   */
  public async addDeposit(deposit: Transaction): Promise<void> {
    const end = getTransactionRangeEnd(deposit)
    const key = KEYS.DEPOSITS.encode([end])
    const value = encode(deposit)
    await this.db.put(key, value)
  }
}
