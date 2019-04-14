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
const TREE_DEPTH = 16
const computeParent = (leftChild: string, rightChild: string): string => {
  return
}

// TODO: Maybe this should go in a utils file?

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
  return nodeIndex % 2 === 0 ? nodeIndex - 1 : nodeIndex + 1
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

// TODO: Where should this sit? Probably OK to be here.
const KEYS = {
  BLOCKS: new BaseKey('b', ['uint32']),
  DEPOSITS: new BaseKey('d', ['uint256']),
  TRANSACTIONS: new BaseKey('t', ['uint32', 'uint256']),
  TREE_NODES: new BaseKey('n', ['uint32', 'uint32']),
}

/**
 * Basic ChainDB implementation that provides a
 * nice interface to the chain database.
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
    const key = KEYS.TREE_NODES.encode([blockNumber, nodeIndex])
    const value = await this.db.get(key)

    // We have this node, return it.
    if (value !== null) {
      return value.toString()
    }

    // We don't have the node, but we might be able to compute the it from its
    // children. Recursively find each child and see if we can create the node.

    // We're at a leaf node so we can't go any deeper.
    if (nodeIndex > 2 ** TREE_DEPTH) {
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

    // TODO: Figure out where to store max tree depth since it varies per block.
    // TODO: Add check to make sure nodeIndex isn't greater than max tree depth.

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
    // TODO: Smart transaction removal should check that we're not removing
    // proof elements that might be necessary for other transactions. Need to
    // find a formula that, given the indices of all other leaf nodes, computes
    // the nodes that can be safely removed.

    // We don't have the node and we can't generate it from children.
    if ((await this.getMerkleTreeNode(blockNumber, nodeIndex)) === null) {
      return
    }

    // We have the node value, but we don't know whether we actually have the
    // value stored or if we have children that can re-generate the node.

    // First, check if we have the node in question.
    const key = KEYS.TREE_NODES.encode([blockNumber, nodeIndex])
    const value = await this.db.get(key)

    // We have this specific node, delete it and stop.
    if (value !== null) {
      await this.db.del(key)
      return
    }

    // We don't have the node, but we have children that can be used to
    // re-generate it. Need to recursively delete any of these children.

    // We're at a leaf node so we can't go any deeper.
    if (nodeIndex > 2 ** TREE_DEPTH) {
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
   * Creates an inclusion proof for a given transaction.
   * @param transaction Transaction to create a proof for.
   * @returns the inclusion proof for that transaction.
   */
  public async getInclusionProof(
    transaction: Transaction
  ): Promise<InclusionProof> {
    // TODO: Figure out how to compute block number?
    // TODO: Figure out how to compute the leaf index?
    // TODO: Figure out how to pull the tree height?

    const blockNumber = 0
    const leafIndex = 0
    const treeHeight = 0

    const proof: string[] = []
    let nodeIndex = getLeafNodeIndex(leafIndex, treeHeight)

    // Generate the proof as long as we're not already at the root.
    while (nodeIndex > 0) {
      // Get the sibling node.
      const siblingIndex = getSiblingIndex(nodeIndex)
      const sibling = await this.getMerkleTreeNode(blockNumber, siblingIndex)

      // Don't have the sibling, can't compute the proof.
      if (sibling === null) {
        throw new Error('Cannot compute inclusion proof, missing sibling node.')
      }

      // Add the sibling to the proof and go on to the parent.
      proof.push(sibling)
      nodeIndex = getParentIndex(nodeIndex)
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
