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

    // We're at a leaf node so we can't go any deeper.
    if (nodeIndex > 2 ** TREE_DEPTH) {
      return null
    }

    // Compute left and right children (full binary tree).
    const leftChildIndex = 2 * nodeIndex + 1
    const rightChildIndex = leftChildIndex + 1

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

    // We don't have the node, add it.
    const key = KEYS.TREE_NODES.encode([blockNumber, nodeIndex])
    const value = Buffer.from(nodeHash)
    await this.db.put(key, value)
  }

  /**
   * Creates an inclusion proof for a given transaction.
   * @param transaction Transaction to create a proof for.
   * @returns the inclusion proof for that transaction.
   */
  public async getInclusionProof(
    transaction: Transaction
  ): Promise<InclusionProof> {
    return
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
