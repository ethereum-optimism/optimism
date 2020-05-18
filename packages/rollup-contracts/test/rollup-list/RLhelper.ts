/* External Imports */
import {
  hexStrToBuf,
  bufToHexString,
  BigNumber,
  keccak256,
} from '@eth-optimism/core-utils'

import { newInMemoryDB, SparseMerkleTreeImpl } from '@eth-optimism/core-db'

import { utils } from 'ethers'

interface BlockHeader {
  timestamp: number
  isL1ToL2Tx: boolean
  elementsMerkleRoot: string
  numElementsInBlock: number
  cumulativePrevElements: number
}

interface ElementInclusionProof {
  blockIndex: number
  blockHeader: BlockHeader
  indexInBlock: number
  siblings: string[]
}

/*
 * Helper class which provides all information requried for a particular
 * Rollup block. This includes all of the tranisitions in readable form
 * as well as the merkle tree which it generates.
 */
export class DefaultRollupBlock {
  public timestamp: number
  public isL1ToL2Tx: boolean
  public blockIndex: number //index in
  public cumulativePrevElements: number //in blockHeader
  public elements: string[] //Rollup block
  public elementsMerkleTree: SparseMerkleTreeImpl

  constructor(
    timestamp: number, // Ethereum block this block was submitted in
    isL1ToL2Tx: boolean,
    blockIndex: number, // index in blocks array (first block has blockIndex of 0)
    cumulativePrevElements: number,
    elements: string[]
  ) {
    this.isL1ToL2Tx = isL1ToL2Tx
    this.timestamp = timestamp
    this.blockIndex = blockIndex
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
   * elementIndex is the index in this block of the element
   * that we want to get the siblings of
   */
  public async getSiblings(elementIndex: number): Promise<string[]> {
    const blockInclusion = await this.elementsMerkleTree.getMerkleProof(
      new BigNumber(elementIndex),
      hexStrToBuf(this.elements[elementIndex])
    )
    const path = bufToHexString(blockInclusion.key.toBuffer('B', 32))
    const siblings = blockInclusion.siblings.map((sibBuf) =>
      bufToHexString(sibBuf)
    )
    return siblings
  }

  public async hashBlockHeader(): Promise<string> {
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
   * elementIndex is the index in this block of the element
   * that we want to create an inclusion proof for.
   */

  public async getElementInclusionProof(
    elementIndex: number
  ): Promise<ElementInclusionProof> {
    const bufferRoot = await this.elementsMerkleTree.getRootHash()
    return {
      blockIndex: this.blockIndex,
      blockHeader: {
        timestamp: this.timestamp,
        isL1ToL2Tx: this.isL1ToL2Tx,
        elementsMerkleRoot: bufToHexString(bufferRoot),
        numElementsInBlock: this.elements.length,
        cumulativePrevElements: this.cumulativePrevElements,
      },
      indexInBlock: elementIndex,
      siblings: await this.getSiblings(elementIndex),
    }
  }
}
/*
 * Helper class which provides all information requried for a particular
 * Rollup block. This includes all of the tranisitions in readable form
 * as well as the merkle tree which it generates.
 */
export class RollupQueueBatch {
  public elements: string[] //Rollup block
  public elementsMerkleTree: SparseMerkleTreeImpl

  constructor(elements: string[]) {
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

  public async hashBlockHeader(): Promise<string> {
    const bufferRoot = await this.elementsMerkleTree.getRootHash()
    return utils.solidityKeccak256(
      ['bytes32', 'uint'],
      [bufToHexString(bufferRoot), this.elements.length]
    )
  }
}
