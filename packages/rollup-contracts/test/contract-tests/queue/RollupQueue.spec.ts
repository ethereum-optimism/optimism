import '../../setup'

/* External Imports */
import { getLogger, TestUtils } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Internal Imports */
import { TxQueueBatch } from '../../test-helpers/RLhelper'

/* Contract Imports */
import * as RollupQueue from '../../../build/contracts/RollupQueue.json'
import * as RollupMerkleUtils from '../../../build/contracts/RollupMerkleUtils.json'

/* Logging */
const log = getLogger('rollup-queue', true)

/* Helpers */
const DEFAULT_TX = '0x1234'

/* Tests */
describe('RollupQueue', () => {
  const provider = createMockProvider()
  const [wallet] = getWallets(provider)
  let rollupQueue

  beforeEach(async () => {
    rollupQueue = await deployContract(wallet, RollupQueue, [], {
      gasLimit: 6700000,
    })
  })

  const enqueueAndGenerateBatch = async (tx: string): Promise<TxQueueBatch> => {
    // Submit the rollup batch on-chain
    const enqueueTx = await rollupQueue.enqueueTx(tx)
    const txReceipt = await provider.getTransactionReceipt(enqueueTx.hash)
    const timestamp = (await provider.getBlock(txReceipt.blockNumber)).timestamp
    // Generate a local version of the rollup batch
    const localBatch = new TxQueueBatch(tx, timestamp)
    await localBatch.generateTree()
    return localBatch
  }

  describe('enqueueTx() ', async () => {
    it('should add to batchHeaders array', async () => {
      await rollupQueue.enqueueTx(DEFAULT_TX)
      const batchesLength = await rollupQueue.getBatchHeadersLength()
      batchesLength.toNumber().should.equal(1)
    })

    it('should set the TimestampedHash correctly', async () => {
      const localBatch = await enqueueAndGenerateBatch(DEFAULT_TX)
      const { txHash, timestamp } = await rollupQueue.batchHeaders(0)
      const expectedBatchHeaderHash = await localBatch.getMerkleRoot()
      txHash.should.equal(expectedBatchHeaderHash)
      timestamp.should.equal(localBatch.timestamp)
    })

    it('should add multiple batches correctly', async () => {
      const numBatches = 5
      for (let i = 0; i < numBatches; i++) {
        const localBatch = await enqueueAndGenerateBatch(DEFAULT_TX)
        const { txHash, timestamp } = await rollupQueue.batchHeaders(i)
        const expectedTxHash = await localBatch.getMerkleRoot()
        txHash.should.equal(expectedTxHash)
        timestamp.should.equal(localBatch.timestamp)
      }
      //check batches length
      const batchesLength = await rollupQueue.getBatchHeadersLength()
      batchesLength.toNumber().should.equal(numBatches)
    })
  })

  describe('dequeue()', async () => {
    it('should dequeue single batch', async () => {
      const localBatch = await enqueueAndGenerateBatch(DEFAULT_TX)
      await rollupQueue.dequeue()

      const batchesLength = await rollupQueue.getBatchHeadersLength()
      batchesLength.should.equal(1)
      const { txHash, timestamp } = await rollupQueue.batchHeaders(0)
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
      const numBatches = 5
      const localBatches = []
      for (let i = 0; i < numBatches; i++) {
        const localBatch = await enqueueAndGenerateBatch(DEFAULT_TX)
        localBatches.push(localBatch)
      }
      for (let i = 0; i < numBatches; i++) {
        const frontBatch = await rollupQueue.peek()
        const localFrontBatch = localBatches[i]
        const expectedTxHash = await localFrontBatch.getMerkleRoot()
        frontBatch.txHash.should.equal(expectedTxHash)
        frontBatch.timestamp.should.equal(localFrontBatch.timestamp)

        await rollupQueue.dequeue()

        const front = await rollupQueue.front()
        front.should.equal(i + 1)

        const dequeuedBatch = await rollupQueue.batchHeaders(i)
        dequeuedBatch.txHash.should.equal(
          '0x0000000000000000000000000000000000000000000000000000000000000000'
        )
        dequeuedBatch.timestamp.should.equal(0)
      }
      const batchesLength = await rollupQueue.getBatchHeadersLength()
      batchesLength.should.equal(numBatches)
      const isEmpty = await rollupQueue.isEmpty()
      isEmpty.should.equal(true)
    })

    it('should revert if dequeueing from empty queue', async () => {
      await TestUtils.assertRevertsAsync(
        'Cannot dequeue from an empty queue',
        async () => {
          await rollupQueue.dequeue()
        }
      )
    })

    it('should revert if dequeueing from a once populated, now empty queue', async () => {
      const numBatches = 3
      for (let i = 0; i < numBatches; i++) {
        await enqueueAndGenerateBatch(DEFAULT_TX)
        await rollupQueue.dequeue()
      }
      await TestUtils.assertRevertsAsync(
        'Cannot dequeue from an empty queue',
        async () => {
          await rollupQueue.dequeue()
        }
      )
    })
  })

  describe('peek() and peekTimestamp()', async () => {
    it('should peek successfully with single element', async () => {
      const localBatch = await enqueueAndGenerateBatch(DEFAULT_TX)
      const { txHash, timestamp } = await rollupQueue.peek()
      const expectedBatchHeaderHash = await localBatch.getMerkleRoot()
      txHash.should.equal(expectedBatchHeaderHash)
      timestamp.should.equal(localBatch.timestamp)

      const peekTimestamp = await rollupQueue.peekTimestamp()
      peekTimestamp.should.equal(timestamp)
    })

    it('should revert when peeking at an empty queue', async () => {
      await TestUtils.assertRevertsAsync(
        'Queue is empty, no element to peek at',
        async () => {
          await rollupQueue.peek()
        }
      )
      await TestUtils.assertRevertsAsync(
        'Queue is empty, no element to peek at',
        async () => {
          await rollupQueue.peekTimestamp()
        }
      )
    })
  })
})
