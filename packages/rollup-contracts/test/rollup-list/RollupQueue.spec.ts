import '../setup'

/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Internal Imports */
import { DefaultRollupBlock } from './RLhelper'

/* Logging */
const log = getLogger('rollup-queue', true)

/* Contract Imports */
import * as RollupQueue from '../../build/RollupQueue.json'
import * as RollupMerkleUtils from '../../build/RollupMerkleUtils.json'

/* Begin tests */
describe('RollupQueue', () => {
  const provider = createMockProvider()
  const [wallet1, wallet2] = getWallets(provider)
  let rollupQueue
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
    rollupQueue = await deployContract(
      wallet1,
      RollupQueue,
      [rollupMerkleUtils.address],
      {
        gasLimit: 6700000,
      }
    )
    rollupCtLogFilter = {
      address: rollupQueue.address,
      fromBlock: 0,
      toBlock: 'latest',
    }
  })

  // const enqueueAndGenerateBlock = async (
  //   block: string[],
  //   blockIndex: number,
  //   cumulativePrevElements: number
  // ): Promise<DefaultRollupBlock> => {
  //   // Submit the rollup block on-chain
  //   const enqueueTx = await rollupQueue.enqueueBlock(block)
  //   const txReceipt = await provider.getTransactionReceipt(enqueueTx.hash)
  //   // Generate a local version of the rollup block
  //   const ethBlockNumber = txReceipt.blockNumber
  //   const localBlock = new DefaultRollupBlock(
  //     ethBlockNumber,
  //     blockIndex,
  //     cumulativePrevElements,
  //     block
  //   )
  //   await localBlock.generateTree()
  //   return localBlock
  // }
  /*
   * Test enqueueBlock()
   */
  describe('enqueueBlock() ', async () => {
    it('should not throw as long as it gets a bytes array (even if its invalid)', async () => {
      const block = ['0x1234', '0x1234']
      await rollupQueue.enqueueBlock(block) // Did not throw... success!
    })

    it('should throw if submitting an empty block', async () => {
      const emptyBlock = []
      try {
        await rollupQueue.enqueueBlock(emptyBlock)
      } catch (err) {
        // Success we threw an error!
        return
      }
      throw new Error('Allowed an empty block to be appended')
    })

    it('should add to blocks array', async () => {
      const block = ['0x1234', '0x6578']
      const output = await rollupQueue.enqueueBlock(block)
      log.debug('enqueue block output', JSON.stringify(output))
      const blocksLength = await rollupQueue.getBlocksLength()
      blocksLength.toNumber().should.equal(1)
    })

    it('should update cumulativeNumElements correctly', async () => {
      const block = ['0x1234', '0x5678']
      await rollupQueue.enqueueBlock(block)
      const cumulativeNumElements = await rollupQueue.cumulativeNumElements.call()
      cumulativeNumElements.toNumber().should.equal(2)
    })

    // it('should calculate blockHeaderHash correctly', async () => {
    //   const block = ['0x1234', '0x5678']
    //   const blockIndex = 0
    //   const cumulativePrevElements = 0
    //   const localBlock = await enqueueAndGenerateBlock(
    //     block,
    //     blockIndex,
    //     cumulativePrevElements
    //   )
    //   //Check blockHeaderHash
    //   const expectedBlockHeaderHash = await localBlock.hashBlockHeader()
    //   const calculatedBlockHeaderHash = await rollupQueue.blocks(0)
    //   calculatedBlockHeaderHash.should.equal(expectedBlockHeaderHash)
    // })

    // it('should add multiple blocks correctly', async () => {
    //   const block = ['0x1234', '0x5678']
    //   const numBlocks = 10
    //   for (let blockIndex = 0; blockIndex < numBlocks; blockIndex++) {
    //     const cumulativePrevElements = block.length * blockIndex
    //     const localBlock = await enqueueAndGenerateBlock(
    //       block,
    //       blockIndex,
    //       cumulativePrevElements
    //     )
    //     //Check blockHeaderHash
    //     const expectedBlockHeaderHash = await localBlock.hashBlockHeader()
    //     const calculatedBlockHeaderHash = await rollupQueue.blocks(blockIndex)
    //     calculatedBlockHeaderHash.should.equal(expectedBlockHeaderHash)
    //   }
    //   //check cumulativeNumElements
    //   const cumulativeNumElements = await rollupQueue.cumulativeNumElements.call()
    //   cumulativeNumElements.toNumber().should.equal(numBlocks * block.length)
    //   //check blocks length
    //   const blocksLength = await rollupQueue.getBlocksLength()
    //   blocksLength.toNumber().should.equal(numBlocks)
    // })
    //TODO test with actual transitions and actual state roots
    //TODO test above with multiple blocks with different # elements and different size elements
  })

  // /*
  //  * Test verifyElement()
  //  */
  // describe('verifyElement() ', async () => {
  //   it('should return true for valid elements for different blockIndexs', async () => {
  //     const maxBlockNumber = 5
  //     const minBlockNumber = 0
  //     const block = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
  //     // Create trees of multiple sizes tree
  //     for (
  //       let blockIndex = minBlockNumber;
  //       blockIndex < maxBlockNumber + 1;
  //       blockIndex++
  //     ) {
  //       log.debug(`testing valid proof for block #: ${blockIndex}`)
  //       const cumulativePrevElements = block.length * blockIndex
  //       const localBlock = await enqueueAndGenerateBlock(
  //         block,
  //         blockIndex,
  //         cumulativePrevElements
  //       )
  //       // Create inclusion proof for the element at elementIndex
  //       const elementIndex = 3
  //       const element = block[elementIndex]
  //       const position = localBlock.getPosition(elementIndex)
  //       const elementInclusionProof = await localBlock.getElementInclusionProof(
  //         elementIndex
  //       )
  //       log.debug(
  //         `trying to correctly verify this inclusion proof: ${JSON.stringify(
  //           elementInclusionProof
  //         )}`
  //       )
  //       //run verifyElement()
  //       //
  //       const isIncluded = await rollupQueue.verifyElement(
  //         element,
  //         position,
  //         elementInclusionProof
  //       )
  //       log.debug('isIncluded: ', JSON.stringify(isIncluded))
  //       isIncluded.should.equal(true)
  //     }
  //   })

  //   it('should return false for wrong position with wrong indexInBlock', async () => {
  //     const block = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
  //     const cumulativePrevElements = 0
  //     const blockIndex = 0
  //     const localBlock = await enqueueAndGenerateBlock(
  //       block,
  //       blockIndex,
  //       cumulativePrevElements
  //     )
  //     const elementIndex = 1
  //     const element = block[elementIndex]
  //     const position = localBlock.getPosition(elementIndex)
  //     const elementInclusionProof = await localBlock.getElementInclusionProof(
  //       elementIndex
  //     )
  //     log.debug(
  //       `trying to falsely verify this inclusion proof: ${JSON.stringify(
  //         elementInclusionProof
  //       )}`
  //     )
  //     //Give wrong position so inclusion proof is wrong
  //     const wrongPosition = position + 1
  //     //run verifyElement()
  //     //
  //     const isIncluded = await rollupQueue.verifyElement(
  //       element,
  //       wrongPosition,
  //       elementInclusionProof
  //     )
  //     log.debug('isIncluded: ', JSON.stringify(isIncluded))
  //     isIncluded.should.equal(false)
  //   })

  //   it('should return false for wrong position and matching indexInBlock', async () => {
  //     const block = ['0x1234', '0x4567', '0x890a', '0xabcd']
  //     const cumulativePrevElements = 0
  //     const blockIndex = 0
  //     const localBlock = await enqueueAndGenerateBlock(
  //       block,
  //       blockIndex,
  //       cumulativePrevElements
  //     )
  //     //generate inclusion proof
  //     const elementIndex = 1
  //     const element = block[elementIndex]
  //     const position = localBlock.getPosition(elementIndex)
  //     const elementInclusionProof = await localBlock.getElementInclusionProof(
  //       elementIndex
  //     )
  //     //Give wrong position so inclusion proof is wrong
  //     const wrongPosition = position + 1
  //     //Change index to also be false (so position = index + cumulative)
  //     elementInclusionProof.indexInBlock++
  //     log.debug(
  //       `trying to falsely verify this inclusion proof: ${JSON.stringify(
  //         elementInclusionProof
  //       )}`
  //     )
  //     //run verifyElement()
  //     //
  //     const isIncluded = await rollupQueue.verifyElement(
  //       element,
  //       wrongPosition,
  //       elementInclusionProof
  //     )
  //     log.debug('isIncluded: ', JSON.stringify(isIncluded))
  //     isIncluded.should.equal(false)
  //   })
  // })

  // /*
  //  * Test deleteAfterInclusive()
  //  */
  // describe('deleteAfterInclusive() ', async () => {
  //   it('should delete single block', async () => {
  //     const block = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
  //     const cumulativePrevElements = 0
  //     const blockIndex = 0
  //     const localBlock = await enqueueAndGenerateBlock(
  //       block,
  //       blockIndex,
  //       cumulativePrevElements
  //     )
  //     const blockHeader = {
  //       ethBlockNumber: localBlock.ethBlockNumber,
  //       elementsMerkleRoot: await localBlock.elementsMerkleTree.getRootHash(),
  //       numElementsInBlock: block.length,
  //       cumulativePrevElements,
  //     }
  //     // Submit the rollup block on-chain
  //     let blocksLength = await rollupQueue.getBlocksLength()
  //     log.debug(`blocksLength before deletion: ${blocksLength}`)
  //     await rollupQueue.deleteAfterInclusive(
  //       blockIndex, // delete the single appended block
  //       blockHeader
  //     )
  //     blocksLength = await rollupQueue.getBlocksLength()
  //     log.debug(`blocksLength after deletion: ${blocksLength}`)
  //     blocksLength.should.equal(0)
  //   })

  //   it('should delete many blocks', async () => {
  //     const block = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
  //     const localBlocks = []
  //     for (let blockIndex = 0; blockIndex < 5; blockIndex++) {
  //       const cumulativePrevElements = blockIndex * block.length
  //       const localBlock = await enqueueAndGenerateBlock(
  //         block,
  //         blockIndex,
  //         cumulativePrevElements
  //       )
  //       localBlocks.push(localBlock)
  //     }
  //     const deleteBlockNumber = 0
  //     const deleteBlock = localBlocks[deleteBlockNumber]
  //     const blockHeader = {
  //       ethBlockNumber: deleteBlock.ethBlockNumber,
  //       elementsMerkleRoot: deleteBlock.elementsMerkleTree.getRootHash(),
  //       numElementsInBlock: block.length,
  //       cumulativePrevElements: deleteBlock.cumulativePrevElements,
  //     }
  //     let blocksLength = await rollupQueue.getBlocksLength()
  //     log.debug(`blocksLength before deletion: ${blocksLength}`)
  //     await rollupQueue.deleteAfterInclusive(
  //       deleteBlockNumber, // delete all blocks (including and after block 0)
  //       blockHeader
  //     )
  //     blocksLength = await rollupQueue.getBlocksLength()
  //     log.debug(`blocksLength after deletion: ${blocksLength}`)
  //     blocksLength.should.equal(0)
  //   })
  // })

  // describe('dequeueBeforeInclusive()', async () => {
  //   it('should dequeue single block', async () => {
  //     const block = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
  //     const cumulativePrevElements = 0
  //     const blockIndex = 0
  //     const localBlock = await enqueueAndGenerateBlock(
  //       block,
  //       blockIndex,
  //       cumulativePrevElements
  //     )
  //     let blocksLength = await rollupQueue.getBlocksLength()
  //     log.debug(`blocksLength before deletion: ${blocksLength}`)
  //     let front = await rollupQueue.front()
  //     log.debug(`front before deletion: ${front}`)
  //     let firstBlockHash = await rollupQueue.blocks(0)
  //     log.debug(`firstBlockHash before deletion: ${firstBlockHash}`)

  //     // delete the single appended block
  //     await rollupQueue.dequeueBeforeInclusive(blockIndex)

  //     blocksLength = await rollupQueue.getBlocksLength()
  //     log.debug(`blocksLength after deletion: ${blocksLength}`)
  //     blocksLength.should.equal(1)
  //     firstBlockHash = await rollupQueue.blocks(0)
  //     log.debug(`firstBlockHash after deletion: ${firstBlockHash}`)
  //     firstBlockHash.should.equal(
  //       '0x0000000000000000000000000000000000000000000000000000000000000000'
  //     )
  //     front = await rollupQueue.front()
  //     log.debug(`front after deletion: ${front}`)
  //     front.should.equal(1)
  //   })

  //   it('should dequeue many blocks', async () => {
  //     const block = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
  //     const localBlocks = []
  //     const numBlocks = 5
  //     for (let blockIndex = 0; blockIndex < numBlocks; blockIndex++) {
  //       const cumulativePrevElements = block.length * blockIndex
  //       const localBlock = await enqueueAndGenerateBlock(
  //         block,
  //         blockIndex,
  //         cumulativePrevElements
  //       )
  //       localBlocks.push(localBlock)
  //     }
  //     let blocksLength = await rollupQueue.getBlocksLength()
  //     log.debug(`blocksLength before deletion: ${blocksLength}`)
  //     let front = await rollupQueue.front()
  //     log.debug(`front before deletion: ${front}`)
  //     for (let i = 0; i < numBlocks; i++) {
  //       const ithBlockHash = await rollupQueue.blocks(i)
  //       log.debug(`blockHash #${i} before deletion: ${ithBlockHash}`)
  //     }
  //     await rollupQueue.dequeueBeforeInclusive(numBlocks - 1)
  //     blocksLength = await rollupQueue.getBlocksLength()
  //     log.debug(`blocksLength after deletion: ${blocksLength}`)
  //     blocksLength.should.equal(numBlocks)
  //     front = await rollupQueue.front()
  //     log.debug(`front after deletion: ${front}`)
  //     front.should.equal(numBlocks)
  //     for (let i = 0; i < numBlocks; i++) {
  //       const ithBlockHash = await rollupQueue.blocks(i)
  //       log.debug(`blockHash #${i} after deletion: ${ithBlockHash}`)
  //       ithBlockHash.should.equal(
  //         '0x0000000000000000000000000000000000000000000000000000000000000000'
  //       )
  //     }
  //   })
  // })
})
