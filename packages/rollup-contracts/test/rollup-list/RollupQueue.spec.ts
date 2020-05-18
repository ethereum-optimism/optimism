import '../setup'

/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Internal Imports */
import { RollupQueueBatch } from './RLhelper'

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
  })

  const enqueueAndGenerateBlock = async (
    block: string[]
  ): Promise<RollupQueueBatch> => {
    // Submit the rollup block on-chain
    await rollupQueue.enqueueBlock(block)
    // Generate a local version of the rollup block
    const localBlock = new RollupQueueBatch(block)
    await localBlock.generateTree()
    return localBlock
  }
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

    it('should calculate blockHeaderHash correctly', async () => {
      const block = ['0x1234', '0x5678']
      const localBlock = await enqueueAndGenerateBlock(block)
      //Check blockHeaderHash
      const expectedBlockHeaderHash = await localBlock.hashBlockHeader()
      const calculatedBlockHeaderHash = await rollupQueue.blocks(0)
      calculatedBlockHeaderHash.should.equal(expectedBlockHeaderHash)
    })

    it('should add multiple blocks correctly', async () => {
      const block = ['0x1234', '0x5678']
      const numBlocks = 10
      for (let blockIndex = 0; blockIndex < numBlocks; blockIndex++) {
        const cumulativePrevElements = block.length * blockIndex
        const localBlock = await enqueueAndGenerateBlock(block)
        //Check blockHeaderHash
        const expectedBlockHeaderHash = await localBlock.hashBlockHeader()
        const calculatedBlockHeaderHash = await rollupQueue.blocks(blockIndex)
        calculatedBlockHeaderHash.should.equal(expectedBlockHeaderHash)
      }
      //check blocks length
      const blocksLength = await rollupQueue.getBlocksLength()
      blocksLength.toNumber().should.equal(numBlocks)
    })
  })

  describe('dequeueBeforeInclusive()', async () => {
    it('should dequeue single block', async () => {
      const block = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
      const cumulativePrevElements = 0
      const blockIndex = 0
      const localBlock = await enqueueAndGenerateBlock(block)
      let blocksLength = await rollupQueue.getBlocksLength()
      log.debug(`blocksLength before deletion: ${blocksLength}`)
      let front = await rollupQueue.front()
      log.debug(`front before deletion: ${front}`)
      let firstBlockHash = await rollupQueue.blocks(0)
      log.debug(`firstBlockHash before deletion: ${firstBlockHash}`)

      // delete the single appended block
      await rollupQueue.dequeueBeforeInclusive(blockIndex)

      blocksLength = await rollupQueue.getBlocksLength()
      log.debug(`blocksLength after deletion: ${blocksLength}`)
      blocksLength.should.equal(1)
      firstBlockHash = await rollupQueue.blocks(0)
      log.debug(`firstBlockHash after deletion: ${firstBlockHash}`)
      firstBlockHash.should.equal(
        '0x0000000000000000000000000000000000000000000000000000000000000000'
      )
      front = await rollupQueue.front()
      log.debug(`front after deletion: ${front}`)
      front.should.equal(1)
    })

    it('should dequeue many blocks', async () => {
      const block = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
      const localBlocks = []
      const numBlocks = 5
      for (let blockIndex = 0; blockIndex < numBlocks; blockIndex++) {
        const cumulativePrevElements = block.length * blockIndex
        const localBlock = await enqueueAndGenerateBlock(block)
        localBlocks.push(localBlock)
      }
      let blocksLength = await rollupQueue.getBlocksLength()
      log.debug(`blocksLength before deletion: ${blocksLength}`)
      let front = await rollupQueue.front()
      log.debug(`front before deletion: ${front}`)
      for (let i = 0; i < numBlocks; i++) {
        const ithBlockHash = await rollupQueue.blocks(i)
        log.debug(`blockHash #${i} before deletion: ${ithBlockHash}`)
      }
      await rollupQueue.dequeueBeforeInclusive(numBlocks - 1)
      blocksLength = await rollupQueue.getBlocksLength()
      log.debug(`blocksLength after deletion: ${blocksLength}`)
      blocksLength.should.equal(numBlocks)
      front = await rollupQueue.front()
      log.debug(`front after deletion: ${front}`)
      front.should.equal(numBlocks)
      for (let i = 0; i < numBlocks; i++) {
        const ithBlockHash = await rollupQueue.blocks(i)
        log.debug(`blockHash #${i} after deletion: ${ithBlockHash}`)
        ithBlockHash.should.equal(
          '0x0000000000000000000000000000000000000000000000000000000000000000'
        )
      }
    })
  })
})
