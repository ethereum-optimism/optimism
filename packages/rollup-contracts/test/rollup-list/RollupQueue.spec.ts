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
    tx: string
  ): Promise<RollupQueueBatch> => {
    // Submit the rollup batch on-chain
    const enqueueTx = await rollupQueue.enqueueTx(tx)
    const txReceipt = await provider.getTransactionReceipt(enqueueTx.hash)
    const timestamp = (await provider.getBlock(txReceipt.blockNumber)).timestamp
    // Generate a local version of the rollup batch
    const localBatch = new RollupQueueBatch(tx, timestamp)
    await localBatch.generateTree()
    return localBatch
  }
  /*
   * Test enqueueTx()
   */
  describe('enqueueTx() ', async () => {
    it('should not throw as long as it gets a bytes array (even if its invalid)', async () => {
      const tx = '0x1234'
      await rollupQueue.enqueueTx(tx) // Did not throw... success!
    })
    it('should add to batches array', async () => {
      const tx = '0x1234'
      const output = await rollupQueue.enqueueTx(tx)
      const batchesLength = await rollupQueue.getBatchesLength()
      batchesLength.toNumber().should.equal(1)
    })
    it('should calculate set the TimestampedHash correctly', async () => {
      const tx = '0x1234'
      const localBatch = await enqueueAndGenerateBatch(tx)
      const { txHash, timestamp } = await rollupQueue.batches(0)
      const expectedBatchHeaderHash = await localBatch.getMerkleRoot()
      txHash.should.equal(expectedBatchHeaderHash)
      timestamp.should.equal(localBatch.timestamp)
    })

    it('should add multiple batches correctly', async () => {
      const tx = '0x1234'
      const numBatches = 10
      for (let batchIndex = 0; batchIndex < numBatches; batchIndex++) {
        const localBatch = await enqueueAndGenerateBatch(tx)
        const { txHash, timestamp } = await rollupQueue.batches(batchIndex)
        const expectedBatchHeaderHash = await localBatch.getMerkleRoot()
        txHash.should.equal(expectedBatchHeaderHash)
        timestamp.should.equal(localBatch.timestamp)
      }
      //check batches length
      const batchesLength = await rollupQueue.getBatchesLength()
      batchesLength.toNumber().should.equal(numBatches)
    })
  })

  describe('dequeueBatch()', async () => {
    it('should dequeue single batch', async () => {
      const tx = '0x1234'
      const localBatch = await enqueueAndGenerateBatch(tx)
      // delete the single appended batch
      await rollupQueue.dequeueBatch()

      const batchesLength = await rollupQueue.getBatchesLength()
      batchesLength.should.equal(1)
      const { txHash, timestamp } = await rollupQueue.batches(0)
      txHash.should.equal(
        '0x0000000000000000000000000000000000000000000000000000000000000000'
      )
      timestamp.should.equal(0)
      const front = await rollupQueue.front()
      front.should.equal(1)
      const isEmpty = await rollupQueue.isEmpty()
      isEmpty.should.equal(true)
    })

    it('should dequeue many batches', async () => {
      const tx = '0x1234'
      const numBatches = 5
      for (let i = 0; i < numBatches; i++) {
        await enqueueAndGenerateBatch(tx)
      }
      for (let i = 0; i < numBatches; i++) {
        await rollupQueue.dequeueBatch()
        const front = await rollupQueue.front()
        front.should.equal(i + 1)
        const { txHash, timestamp } = await rollupQueue.batches(i)
        txHash.should.equal(
          '0x0000000000000000000000000000000000000000000000000000000000000000'
        )
        timestamp.should.equal(0)
      }
      const batchesLength = await rollupQueue.getBatchesLength()
      batchesLength.should.equal(numBatches)
      const isEmpty = await rollupQueue.isEmpty()
      isEmpty.should.equal(true)
    })

    it('should throw if dequeueing from empty queue', async () => {
      await rollupQueue
        .dequeueBatch()
        .should.be.revertedWith(
          'VM Exception while processing transaction: revert Cannot dequeue from an empty queue'
        )
    })

    it('should throw if dequeueing from a once populated, now empty queue', async () => {
      const tx = '0x1234'
      const numBatches = 3
      for (let i = 0; i < numBatches; i++) {
        await enqueueAndGenerateBatch(tx)
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
  describe('peek() and peekTimestamp()', async () => {
    it('should peek successfully with single element', async () => {
      const tx = '0x1234'
      const localBatch = await enqueueAndGenerateBatch(tx)
      const { txHash, timestamp } = await rollupQueue.peek()
      const peekTimestamp = await rollupQueue.peekTimestamp()
      const expectedBatchHeaderHash = await localBatch.getMerkleRoot()
      txHash.should.equal(expectedBatchHeaderHash)
      peekTimestamp.should.equal(timestamp)
      timestamp.should.equal(localBatch.timestamp)
    })
    it('should revert when peeking at empty queue', async () => {
      await rollupQueue
        .peek()
        .should.be.revertedWith(
          'VM Exception while processing transaction: revert Queue is empty, no element to peek at'
        )
      await rollupQueue
        .peekTimestamp()
        .should.be.revertedWith(
          'VM Exception while processing transaction: revert Queue is empty, no element to peek at'
        )
    })
  })
})
