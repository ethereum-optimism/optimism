/* External Imports */
import {
  hexStrToBuf,
  bufToHexString,
  BigNumber,
} from '@eth-optimism/core-utils'

import { newInMemoryDB, SparseMerkleTreeImpl } from '@eth-optimism/core-db'

import { utils } from 'ethers'

const abi = new utils.AbiCoder()

interface TxChainBatchHeader {
  timestamp: number
  blocknumber: number
  isL1ToL2Tx: boolean
  elementsMerkleRoot: string
  numElementsInBatch: number
  cumulativePrevElements: number
}

interface TxElementInclusionProof {
  batchIndex: number
  batchHeader: TxChainBatchHeader
  indexInBatch: number
  siblings: string[]
}

interface StateBatchHeader {
  elementsMerkleRoot: string
  numElementsInBatch: number
  cumulativePrevElements: number
}

interface StateElementInclusionProof {
  batchIndex: number
  batchHeader: StateBatchHeader
  indexInBatch: number
  siblings: string[]
}

export class ChainBatch {
  public batchIndex: number //index in
  public cumulativePrevElements: number //in batchHeader
  public elements: string[] //Rollup batch
  public elementsMerkleTree: SparseMerkleTreeImpl

  constructor(
    batchIndex: number, // index in batchs array (first batch has batchIndex of 0)
    cumulativePrevElements: number,
    elements: string[]
  ) {
    this.batchIndex = batchIndex
    this.cumulativePrevElements = cumulativePrevElements
    this.elements = elements
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

  /*
   * elementIndex is the index in this batch of the element
   * that we want to create an inclusion proof for.
   */
  public async getElementInclusionProof(
    elementIndex: number
  ): Promise<StateElementInclusionProof> {
    const bufferRoot = await this.elementsMerkleTree.getRootHash()
    return {
      batchIndex: this.batchIndex,
      batchHeader: {
        elementsMerkleRoot: bufToHexString(bufferRoot),
        numElementsInBatch: this.elements.length,
        cumulativePrevElements: this.cumulativePrevElements,
      },
      indexInBatch: elementIndex,
      siblings: await this.getSiblings(elementIndex),
    }
  }
}

export function getL1ToL2MessageTxData(
  sender: any,
  to: any,
  gasLimit: any,
  data: any
): string {
  return abi.encode(
    ['address', 'address', 'uint32', 'bytes'],
    [sender, to, gasLimit, data]
  )
}

/*
 * Helper class which provides all information requried for a particular
 * Rollup batch. This includes all of the transactions in readable form
 * as well as the merkle tree which it generates.
 */
export class TxChainBatch extends ChainBatch {
  public timestamp: number
  public blocknumber: number
  public isL1ToL2Tx: boolean

  constructor(
    timestamp: number, // Ethereum timestamp this batch was submitted in
    blocknumber: number, // Same as above w/ blocknumber
    isL1ToL2Tx: boolean,
    batchIndex: number, // index in batchs array (first batch has batchIndex of 0)
    cumulativePrevElements: number,
    elements: any[],
    sender?: string
  ) {
    let elementsToMerklize: string[]
    if (isL1ToL2Tx) {
      if (!sender) {
        throw new Error(
          'To create a local L1->L2 batch, the msg.sender must be known.'
        )
      }
      elementsToMerklize = elements.map((l1ToL2Params) => {
        return getL1ToL2MessageTxData(
          sender,
          l1ToL2Params[0],
          l1ToL2Params[1],
          l1ToL2Params[2]
        )
      })
    } else {
      elementsToMerklize = elements
    }
    super(batchIndex, cumulativePrevElements, elementsToMerklize)
    this.isL1ToL2Tx = isL1ToL2Tx
    this.timestamp = timestamp
    this.blocknumber = blocknumber
  }

  public async hashBatchHeader(): Promise<string> {
    const bufferRoot = await this.elementsMerkleTree.getRootHash()
    return utils.solidityKeccak256(
      ['uint', 'uint', 'bool', 'bytes32', 'uint', 'uint'],
      [
        this.timestamp,
        this.blocknumber,
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
  ): Promise<TxElementInclusionProof> {
    const bufferRoot = await this.elementsMerkleTree.getRootHash()
    return {
      batchIndex: this.batchIndex,
      batchHeader: {
        timestamp: this.timestamp,
        blocknumber: this.blocknumber,
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

export class StateChainBatch extends ChainBatch {
  constructor(
    batchIndex: number, // index in batches array (first batch has batchIndex of 0)
    cumulativePrevElements: number,
    elements: string[]
  ) {
    super(batchIndex, cumulativePrevElements, elements)
  }

  public async hashBatchHeader(): Promise<string> {
    const bufferRoot = await this.elementsMerkleTree.getRootHash()
    return utils.solidityKeccak256(
      ['bytes32', 'uint', 'uint'],
      [
        bufToHexString(bufferRoot),
        this.elements.length,
        this.cumulativePrevElements,
      ]
    )
  }
}

/*
 * Helper class which provides all information requried for a particular
 * Rollup Queue Batch. This includes all of the transactions in readable form
 * as well as the merkle tree which it generates.
 */
export class TxQueueBatch {
  public elements: string[]
  public elementsMerkleTree: SparseMerkleTreeImpl
  public timestamp: number
  public blocknumber: number

  // TODO remove blocknumber optionality, just here for testing
  constructor(tx: string, timestamp: number, blocknumber: number) {
    this.elements = [tx]
    this.timestamp = timestamp
    this.blocknumber = blocknumber
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

  public async hashBatchHeader(
    isL1ToL2Tx: boolean,
    cumulativePrevElements: number = 0
  ): Promise<string> {
    const txHash = await this.getMerkleRoot()
    return utils.solidityKeccak256(
      ['uint', 'uint', 'bool', 'bytes32', 'uint', 'uint'],
      [this.timestamp, this.blocknumber, isL1ToL2Tx, txHash, 1, cumulativePrevElements]
    )
  }
}
