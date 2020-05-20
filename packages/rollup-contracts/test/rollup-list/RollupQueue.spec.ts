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

  const enqueueAndGenerateBatch = async (
    batch: string[]
  ): Promise<RollupQueueBatch> => {
    // Submit the rollup batch on-chain
    await rollupQueue.enqueueBatch(batch)
    // Generate a local version of the rollup batch
    const localBatch = new RollupQueueBatch(batch)
    await localBatch.generateTree()
    return localBatch
  }
  /*
   * Test enqueueBatch()
   */
  describe('enqueueBatch() ', async () => {
    it('should not throw as long as it gets a bytes array (even if its invalid)', async () => {
      const batch = ['0x1234', '0x1234']
      await rollupQueue.enqueueBatch(batch) // Did not throw... success!
    })

    it('should throw if submitting an empty batch', async () => {
      const emptyBatch = []
      await rollupQueue
        .enqueueBatch(emptyBatch)
        .should.be.revertedWith(
          'VM Exception while processing transaction: revert Cannot submit an empty batch'
        )
    })

    it('should add to batches array', async () => {
      const batch = ['0x1234', '0x6578']
      const output = await rollupQueue.enqueueBatch(batch)
      const batchesLength = await rollupQueue.getBatchesLength()
      batchesLength.toNumber().should.equal(1)
    })

    it('should update cumulativeNumElements correctly', async () => {
      const batch = ['0x1234', '0x5678']
      await rollupQueue.enqueueBatch(batch)
      const cumulativeNumElements = await rollupQueue.cumulativeNumElements.call()
      cumulativeNumElements.toNumber().should.equal(2)
    })

    it('should calculate batchHeaderHash correctly', async () => {
      const batch = ['0x1234', '0x5678']
      const localBatch = await enqueueAndGenerateBatch(batch)
      //Check batchHeaderHash
      const expectedBatchHeaderHash = await localBatch.hashBatchHeader()
      const calculatedBatchHeaderHash = (await rollupQueue.batches(0))
        .batchHeaderHash
      calculatedBatchHeaderHash.should.equal(expectedBatchHeaderHash)
    })

    it('should add multiple batches correctly', async () => {
      const batch = ['0x1234', '0x5678']
      const numBatches = 10
      for (let batchIndex = 0; batchIndex < numBatches; batchIndex++) {
        const cumulativePrevElements = batch.length * batchIndex
        const localBatch = await enqueueAndGenerateBatch(batch)
        //Check batchHeaderHash
        const expectedBatchHeaderHash = await localBatch.hashBatchHeader()
        const calculatedBatchHeaderHash = (
          await rollupQueue.batches(batchIndex)
        ).batchHeaderHash
        calculatedBatchHeaderHash.should.equal(expectedBatchHeaderHash)
      }
      //check batches length
      const batchesLength = await rollupQueue.getBatchesLength()
      batchesLength.toNumber().should.equal(numBatches)
    })
  })

  describe('dequeueBatch()', async () => {
    it('should dequeue single batch', async () => {
      const batch = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
      const localBatch = await enqueueAndGenerateBatch(batch)
      // delete the single appended batch
      await rollupQueue.dequeueBatch()

      const batchesLength = await rollupQueue.getBatchesLength()
      batchesLength.should.equal(1)
      const firstBatchHash = (await rollupQueue.batches(0)).batchHeaderHash
      firstBatchHash.should.equal(
        '0x0000000000000000000000000000000000000000000000000000000000000000'
      )
      const front = await rollupQueue.front()
      front.should.equal(1)
    })

    it('should dequeue many batches', async () => {
      const batch = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
      const numBatches = 5
      for (let batchIndex = 0; batchIndex < numBatches; batchIndex++) {
        await enqueueAndGenerateBatch(batch)
      }
      for (let i = 0; i < numBatches; i++) {
        await rollupQueue.dequeueBatch()
        const front = await rollupQueue.front()
        front.should.equal(i + 1)
        const ithBatchHash = (await rollupQueue.batches(i)).batchHeaderHash
        ithBatchHash.should.equal(
          '0x0000000000000000000000000000000000000000000000000000000000000000'
        )
      }
      const batchesLength = await rollupQueue.getBatchesLength()
      batchesLength.should.equal(numBatches)
    })

    it('should throw if dequeueing from empty queue', async () => {
      await rollupQueue
        .dequeueBatch()
        .should.be.revertedWith(
          'VM Exception while processing transaction: revert Cannot dequeue from an empty queue'
        )
    })

    it('should throw if dequeueing from a once populated, now empty queue', async () => {
      const batch = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
      const numBatches = 3
      for (let batchIndex = 0; batchIndex < numBatches; batchIndex++) {
        await enqueueAndGenerateBatch(batch)
      }
      for (let i = 0; i < numBatches; i++) {
        await rollupQueue.dequeueBatch()
      }
      await rollupQueue
        .dequeueBatch()
        .should.be.revertedWith(
          'VM Exception while processing transaction: revert Cannot dequeue from an empty queue'
        )
    })
  })
})
