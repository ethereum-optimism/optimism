import '../setup'

/* External Imports */
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import {
  hexStrToBuf,
  bufToHexString,
  BigNumber,
  keccak256,
} from '@pigi/core-utils'

import { newInMemoryDB, SparseMerkleTreeImpl } from '@pigi/core-db'

import { DefaultRollupBlock } from './RLhelper'

/* Logging */
import debug from 'debug'
const log = debug('test:info:rollup-list')

/* Contract Imports */
import * as RollupList from '../../build/RollupList.json'
import * as RollupMerkleUtils from '../../build/RollupMerkleUtils.json'
import { Contract, ContractFactory, Wallet, utils } from 'ethers'

/* Begin tests */
describe.only('RollupList', () => {
  const provider = createMockProvider()
  const [wallet1, wallet2] = getWallets(provider)
  let rollupList
  let rollupMerkleUtils
  let rollupCtLogFilter

  /* Link libraries before tests */
  before(async () => {
    rollupMerkleUtils = await deployContract(wallet1, RollupMerkleUtils, [], {
      gasLimit: 6700000,
    })
  })

  /* Deploy a new RollupChain before each test */
  beforeEach(async () => {
    rollupList = await deployContract(
      wallet1,
      RollupList,
      [rollupMerkleUtils.address],
      {
        gasLimit: 6700000,
      }
    )
    rollupCtLogFilter = {
      address: rollupList.address,
      fromBlock: 0,
      toBlock: 'latest',
    }
  })

  const enqueueAndGenerateBlock = async (
    block: string[],
    blockIndex: number,
    cumulativePrevElements: number
  ): Promise<DefaultRollupBlock> => {
    // Submit the rollup block on-chain
    const enqueueTx = await rollupList.enqueueBlock(block)
    const txReceipt = await provider.getTransactionReceipt(enqueueTx.hash)
    // Generate a local version of the rollup block
    const ethBlockNumber = txReceipt.blockNumber
    const localBlock = new DefaultRollupBlock(
      ethBlockNumber,
      blockIndex,
      cumulativePrevElements,
      block
    )
    await localBlock.generateTree()
    return localBlock
  }
  /*
   * Test enqueueBlock()
   */
  describe('enqueueBlock() ', async () => {
    it('should not throw as long as it gets a bytes array (even if its invalid)', async () => {
      const block = ['0x1234', '0x1234']
      await rollupList.enqueueBlock(block) // Did not throw... success!
    })

    it('should throw if submitting an empty block', async () => {
      const emptyBlock = []
      try {
        await rollupList.enqueueBlock(emptyBlock)
      } catch (err) {
        // Success we threw an error!
        return
      }
      throw new Error('Allowed an empty block to be appended')
    })

    it('should add to blocks array', async () => {
      const block = ['0x1234', '0x6578']
      const output = await rollupList.enqueueBlock(block)
      log('enqueue block output', JSON.stringify(output))
      const blocksLength = await rollupList.getBlocksLength()
      blocksLength.toNumber().should.equal(1)
    })

    it('should update cumulativeNumElements correctly', async () => {
      const block = ['0x1234', '0x5678']
      await rollupList.enqueueBlock(block)
      const cumulativeNumElements = await rollupList.cumulativeNumElements.call()
      cumulativeNumElements.toNumber().should.equal(2)
    })

    it('should calculate blockHeaderHash correctly', async () => {
      const block = ['0x1234', '0x5678']
      const blockIndex = 0
      const cumulativePrevElements = 0
      const localBlock = await enqueueAndGenerateBlock(
        block,
        blockIndex,
        cumulativePrevElements
      )
      //Check blockHeaderHash
      const expectedBlockHeaderHash = await localBlock.hashBlockHeader()
      const calculatedBlockHeaderHash = await rollupList.blocks(0)
      calculatedBlockHeaderHash.should.equal(expectedBlockHeaderHash)
    })

    it('should add multiple blocks correctly', async () => {
      const block = ['0x1234', '0x5678']
      const numBlocks = 10
      for (let blockIndex = 0; blockIndex < numBlocks; blockIndex++) {
        const cumulativePrevElements = block.length * blockIndex
        const localBlock = await enqueueAndGenerateBlock(
          block,
          blockIndex,
          cumulativePrevElements
        )
        //Check blockHeaderHash
        const expectedBlockHeaderHash = await localBlock.hashBlockHeader()
        const calculatedBlockHeaderHash = await rollupList.blocks(blockIndex)
        calculatedBlockHeaderHash.should.equal(expectedBlockHeaderHash)
      }
      //check cumulativeNumElements
      const cumulativeNumElements = await rollupList.cumulativeNumElements.call()
      cumulativeNumElements.toNumber().should.equal(numBlocks * block.length)
      //check blocks length
      const blocksLength = await rollupList.getBlocksLength()
      blocksLength.toNumber().should.equal(numBlocks)
    })
    //TODO test with actual transitions and actual state roots
    //TODO test above with multiple blocks with different # elements and different size elements
  })

  /*
   * Test verifyElement()
   */
  describe('verifyElement() ', async () => {
    it('should return true for valid elements for different blockIndexs', async () => {
      const maxBlockNumber = 5
      const minBlockNumber = 0
      const block = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
      // Create trees of multiple sizes tree
      for (
        let blockIndex = minBlockNumber;
        blockIndex < maxBlockNumber + 1;
        blockIndex++
      ) {
        log(`testing valid proof for block #: ${blockIndex}`)
        const cumulativePrevElements = block.length * blockIndex
        const localBlock = await enqueueAndGenerateBlock(
          block,
          blockIndex,
          cumulativePrevElements
        )
        // Create inclusion proof for the element at elementIndex
        const elementIndex = 3
        const element = block[elementIndex]
        const position = localBlock.getPosition(elementIndex)
        const elementInclusionProof = await localBlock.getElementInclusionProof(
          elementIndex
        )
        log(
          `trying to correctly verify this inclusion proof: ${JSON.stringify(
            elementInclusionProof
          )}`
        )
        //run verifyElement()
        //
        const isIncluded = await rollupList.verifyElement(
          element,
          position,
          elementInclusionProof
        )
        log('isIncluded: ', JSON.stringify(isIncluded))
        isIncluded.should.equal(true)
      }
    }).timeout(8000)

    it('should return false for wrong position with wrong indexInBlock', async () => {
      const block = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
      const cumulativePrevElements = 0
      const blockIndex = 0
      const localBlock = await enqueueAndGenerateBlock(
        block,
        blockIndex,
        cumulativePrevElements
      )
      const elementIndex = 1
      const element = block[elementIndex]
      const position = localBlock.getPosition(elementIndex)
      const elementInclusionProof = await localBlock.getElementInclusionProof(
        elementIndex
      )
      log(
        `trying to falsely verify this inclusion proof: ${JSON.stringify(
          elementInclusionProof
        )}`
      )
      //Give wrong position so inclusion proof is wrong
      const wrongPosition = position + 1
      //run verifyElement()
      //
      const isIncluded = await rollupList.verifyElement(
        element,
        wrongPosition,
        elementInclusionProof
      )
      log('isIncluded: ', JSON.stringify(isIncluded))
      isIncluded.should.equal(false)
    }).timeout(8000)

    it('should return false for wrong position and matching indexInBlock', async () => {
      const block = ['0x1234', '0x4567', '0x890a', '0xabcd']
      const cumulativePrevElements = 0
      const blockIndex = 0
      const localBlock = await enqueueAndGenerateBlock(
        block,
        blockIndex,
        cumulativePrevElements
      )
      //generate inclusion proof
      const elementIndex = 1
      const element = block[elementIndex]
      const position = localBlock.getPosition(elementIndex)
      const elementInclusionProof = await localBlock.getElementInclusionProof(
        elementIndex
      )
      //Give wrong position so inclusion proof is wrong
      const wrongPosition = position + 1
      //Change index to also be false (so position = index + cumulative)
      elementInclusionProof.indexInBlock++
      log(
        `trying to falsely verify this inclusion proof: ${JSON.stringify(
          elementInclusionProof
        )}`
      )
      //run verifyElement()
      //
      const isIncluded = await rollupList.verifyElement(
        element,
        wrongPosition,
        elementInclusionProof
      )
      log('isIncluded: ', JSON.stringify(isIncluded))
      isIncluded.should.equal(false)
    }).timeout(8000)
  })

  /*
   * Test deleteAfterInclusive()
   */
  describe('deleteAfterInclusive() ', async () => {
    it('should delete single block', async () => {
      const block = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
      const cumulativePrevElements = 0
      const blockIndex = 0
      const localBlock = await enqueueAndGenerateBlock(
        block,
        blockIndex,
        cumulativePrevElements
      )
      const blockHeader = {
        ethBlockNumber: localBlock.ethBlockNumber,
        elementsMerkleRoot: await localBlock.elementsMerkleTree.getRootHash(),
        numElementsInBlock: block.length,
        cumulativePrevElements,
      }
      // Submit the rollup block on-chain
      let blocksLength = await rollupList.getBlocksLength()
      log(`blocksLength before deletion: ${blocksLength}`)
      await rollupList.deleteAfterInclusive(
        blockIndex, // delete the single appended block
        blockHeader
      )
      blocksLength = await rollupList.getBlocksLength()
      log(`blocksLength after deletion: ${blocksLength}`)
      blocksLength.should.equal(0)
    }).timeout(8000)

    it('should delete many blocks', async () => {
      const block = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
      const localBlocks = []
      for (let blockIndex = 0; blockIndex < 5; blockIndex++) {
        const cumulativePrevElements = blockIndex * block.length
        const localBlock = await enqueueAndGenerateBlock(
          block,
          blockIndex,
          cumulativePrevElements
        )
        localBlocks.push(localBlock)
      }
      const deleteBlockNumber = 0
      const deleteBlock = localBlocks[deleteBlockNumber]
      const blockHeader = {
        ethBlockNumber: deleteBlock.ethBlockNumber,
        elementsMerkleRoot: deleteBlock.elementsMerkleTree.getRootHash(),
        numElementsInBlock: block.length,
        cumulativePrevElements: deleteBlock.cumulativePrevElements,
      }
      let blocksLength = await rollupList.getBlocksLength()
      log(`blocksLength before deletion: ${blocksLength}`)
      await rollupList.deleteAfterInclusive(
        deleteBlockNumber, // delete all blocks (including and after block 0)
        blockHeader
      )
      blocksLength = await rollupList.getBlocksLength()
      log(`blocksLength after deletion: ${blocksLength}`)
      blocksLength.should.equal(0)
    }).timeout(8000)
  })

  describe('dequeueBeforeInclusive()', async () => {
    it('should dequeue single block', async () => {
      const block = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
      const cumulativePrevElements = 0
      const blockIndex = 0
      const localBlock = await enqueueAndGenerateBlock(
        block,
        blockIndex,
        cumulativePrevElements
      )
      let blocksLength = await rollupList.getBlocksLength()
      log(`blocksLength before deletion: ${blocksLength}`)
      let front = await rollupList.front()
      log(`front before deletion: ${front}`)
      let firstBlockHash = await rollupList.blocks(0)
      log(`firstBlockHash before deletion: ${firstBlockHash}`)

      // delete the single appended block
      await rollupList.dequeueBeforeInclusive(blockIndex)

      blocksLength = await rollupList.getBlocksLength()
      log(`blocksLength after deletion: ${blocksLength}`)
      blocksLength.should.equal(1)
      firstBlockHash = await rollupList.blocks(0)
      log(`firstBlockHash after deletion: ${firstBlockHash}`)
      firstBlockHash.should.equal(
        '0x0000000000000000000000000000000000000000000000000000000000000000'
      )
      front = await rollupList.front()
      log(`front after deletion: ${front}`)
      front.should.equal(1)
    }).timeout(8000)

    it('should dequeue many blocks', async () => {
      const block = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
      const localBlocks = []
      const numBlocks = 5
      for (let blockIndex = 0; blockIndex < numBlocks; blockIndex++) {
        const cumulativePrevElements = block.length * blockIndex
        const localBlock = await enqueueAndGenerateBlock(
          block,
          blockIndex,
          cumulativePrevElements
        )
        localBlocks.push(localBlock)
      }
      let blocksLength = await rollupList.getBlocksLength()
      log(`blocksLength before deletion: ${blocksLength}`)
      let front = await rollupList.front()
      log(`front before deletion: ${front}`)
      for (let i = 0; i < numBlocks; i++) {
        const ithBlockHash = await rollupList.blocks(i)
        log(`blockHash #${i} before deletion: ${ithBlockHash}`)
      }
      await rollupList.dequeueBeforeInclusive(numBlocks - 1)
      blocksLength = await rollupList.getBlocksLength()
      log(`blocksLength after deletion: ${blocksLength}`)
      blocksLength.should.equal(numBlocks)
      front = await rollupList.front()
      log(`front after deletion: ${front}`)
      front.should.equal(numBlocks)
      for (let i = 0; i < numBlocks; i++) {
        const ithBlockHash = await rollupList.blocks(i)
        log(`blockHash #${i} after deletion: ${ithBlockHash}`)
        ithBlockHash.should.equal(
          '0x0000000000000000000000000000000000000000000000000000000000000000'
        )
      }
    }).timeout(8000)
  })
})
