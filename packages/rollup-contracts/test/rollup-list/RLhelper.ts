/* External Imports */
import {
  hexStrToBuf,
  bufToHexString,
  BigNumber,
  keccak256,
} from '@eth-optimism/core-utils'

import { newInMemoryDB, SparseMerkleTreeImpl } from '@eth-optimism/core-db'

import { utils } from 'ethers'

interface TxChainBatchHeader {
  timestamp: number
  isL1ToL2Tx: boolean
  elementsMerkleRoot: string
  numElementsInBatch: number
  cumulativePrevElements: number
}

interface ElementInclusionProof {
  batchIndex: number
  batchHeader: TxChainBatchHeader
  indexInBatch: number
  siblings: string[]
}

/*
 * Helper class which provides all information requried for a particular
 * Rollup batch. This includes all of the transactions in readable form
 * as well as the merkle tree which it generates.
 */
export class DefaultRollupBatch {
  public timestamp: number
  public isL1ToL2Tx: boolean
  public batchIndex: number //index in
  public cumulativePrevElements: number //in batchHeader
  public elements: string[] //Rollup batch
  public elementsMerkleTree: SparseMerkleTreeImpl

  constructor(
    timestamp: number, // Ethereum batch this batch was submitted in
    isL1ToL2Tx: boolean,
    batchIndex: number, // index in batchs array (first batch has batchIndex of 0)
    cumulativePrevElements: number,
    elements: string[]
  ) {
    this.isL1ToL2Tx = isL1ToL2Tx
    this.timestamp = timestamp
    this.batchIndex = batchIndex
    this.cumulativePrevElements = cumulativePrevElements
    this.elements = elements
  }
  /*
   * Generate the elements merkle tree from this.elements
   */
  public async generateTree(): Promise<void> {
    // Create a tree!
    const treeHeight = Math.ceil(Math.log2(this.elements.length)) + 1 // The height should actually not be plus 1
    this.elementsMerkleTree = await SparseMerkleTreeImpl.create(
      newInMemoryDB(),
      undefined,
      treeHeight
    )
    for (let i = 0; i < this.elements.length; i++) {
      await this.elementsMerkleTree.update(
        new BigNumber(i, 10),
        hexStrToBuf(this.elements[i])
      )
    }
  }

  /*
   * Returns the absolute position of the element at elementIndex
   */
  public getPosition(elementIndex: number): number {
    return this.cumulativePrevElements + elementIndex
  }

  /*
   * elementIndex is the index in this batch of the element
   * that we want to get the siblings of
   */
  public async getSiblings(elementIndex: number): Promise<string[]> {
    const batchInclusion = await this.elementsMerkleTree.getMerkleProof(
      new BigNumber(elementIndex),
      hexStrToBuf(this.elements[elementIndex])
    )
    const path = bufToHexString(batchInclusion.key.toBuffer('B', 32))
    const siblings = batchInclusion.siblings.map((sibBuf) =>
      bufToHexString(sibBuf)
    )
    return siblings
  }

  public async hashBatchHeader(): Promise<string> {
    const bufferRoot = await this.elementsMerkleTree.getRootHash()
    return utils.solidityKeccak256(
      ['uint', 'bool', 'bytes32', 'uint', 'uint'],
      [
        this.timestamp,
        this.isL1ToL2Tx,
        bufToHexString(bufferRoot),
        this.elements.length,
        this.cumulativePrevElements,
      ]
    )
  }

  /*
   * elementIndex is the index in this batch of the element
   * that we want to create an inclusion proof for.
   */

  public async getElementInclusionProof(
    elementIndex: number
  ): Promise<ElementInclusionProof> {
    const bufferRoot = await this.elementsMerkleTree.getRootHash()
    return {
      batchIndex: this.batchIndex,
      batchHeader: {
        timestamp: this.timestamp,
        isL1ToL2Tx: this.isL1ToL2Tx,
        elementsMerkleRoot: bufToHexString(bufferRoot),
        numElementsInBatch: this.elements.length,
        cumulativePrevElements: this.cumulativePrevElements,
      },
      indexInBatch: elementIndex,
      siblings: await this.getSiblings(elementIndex),
    }
  }
}
/*
 * Helper class which provides all information requried for a particular
 * Rollup Queue Batch. This includes all of the transactions in readable form
 * as well as the merkle tree which it generates.
 */
export class RollupQueueBatch {
  public elements: string[]
  public elementsMerkleTree: SparseMerkleTreeImpl
  public timestamp: number

  constructor(tx: string, timestamp: number) {
    this.elements = [tx]
    this.timestamp = timestamp
  }
  /*
   * Generate the elements merkle tree from this.elements
   */
  public async generateTree(): Promise<void> {
    const treeHeight = Math.ceil(Math.log2(this.elements.length)) + 1 // The height should actually not be plus 1
    this.elementsMerkleTree = await SparseMerkleTreeImpl.create(
      newInMemoryDB(),
      undefined,
      treeHeight
    )
    for (let i = 0; i < this.elements.length; i++) {
      await this.elementsMerkleTree.update(
        new BigNumber(i, 10),
        hexStrToBuf(this.elements[i])
      )
    }
  }
  public async getMerkleRoot(): Promise<string> {
    const bufferRoot = await this.elementsMerkleTree.getRootHash()
    return bufToHexString(bufferRoot)
  }
}
