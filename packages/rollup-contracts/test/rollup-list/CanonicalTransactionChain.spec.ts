import '../setup'

/* External Imports */
import { getLogger, TestUtils } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { Contract } from 'ethers'

/* Internal Imports */
import { CanonicalTxBatch, TxQueueBatch } from './RLhelper'

/* Logging */
const log = getLogger('canonical-tx-chain', true)

/* Contract Imports */
import * as CanonicalTransactionChain from '../../build/CanonicalTransactionChain.json'
import * as L1ToL2TransactionQueue from '../../build/L1ToL2TransactionQueue.json'
import * as SafetyTransactionQueue from '../../build/SafetyTransactionQueue.json'
import * as RollupMerkleUtils from '../../build/RollupMerkleUtils.json'

/* Begin tests */
describe.only('CanonicalTransactionChain', () => {
  const provider = createMockProvider()
  const [wallet, sequencer, l1ToL2TransactionPasser, randomWallet] = getWallets(
    provider
  )
  let canonicalTxChain
  let rollupMerkleUtils
  let l1ToL2Queue
  let safetyQueue
  const LIVENESS_ASSUMPTION = 600 //600 seconds = 10 minutes
  const DEFAULT_BATCH = ['0x1234', '0x5678']
  const DEFAULT_TX = '0x1234'

  const appendBatch = async (batch: string[]): Promise<number> => {
    const timestamp = Math.floor(Date.now() / 1000)
    // Submit the rollup batch on-chain
    await canonicalTxChain
      .connect(sequencer)
      .appendTransactionBatch(batch, timestamp)
    return timestamp
  }

  const appendAndGenerateBatch = async (
    batch: string[],
    batchIndex: number = 0,
    cumulativePrevElements: number = 0
  ): Promise<CanonicalTxBatch> => {
    const timestamp = await appendBatch(batch)
    // Generate a local version of the rollup batch
    const localBatch = new CanonicalTxBatch(
      timestamp,
      false,
      batchIndex,
      cumulativePrevElements,
      batch
    )
    await localBatch.generateTree()
    return localBatch
  }

  const enqueueAndGenerateL1ToL2Batch = async (
    _tx: string
  ): Promise<TxQueueBatch> => {
    // Submit the rollup batch on-chain
    const enqueueTx = await l1ToL2Queue
      .connect(l1ToL2TransactionPasser)
      .enqueueTx(_tx)
    const localBatch = await generateQueueBatch(_tx, enqueueTx.hash)
    return localBatch
  }
  const enqueueAndGenerateSafetyBatch = async (
    _tx: string
  ): Promise<TxQueueBatch> => {
    const enqueueTx = await safetyQueue.connect(randomWallet).enqueueTx(_tx)
    const localBatch = await generateQueueBatch(_tx, enqueueTx.hash)
    return localBatch
  }

  const generateQueueBatch = async (
    _tx: string,
    _txHash: string
  ): Promise<TxQueueBatch> => {
    const txReceipt = await provider.getTransactionReceipt(_txHash)
    const timestamp = (await provider.getBlock(txReceipt.blockNumber)).timestamp
    // Generate a local version of the rollup batch
    const localBatch = new TxQueueBatch(_tx, timestamp)
    await localBatch.generateTree()
    return localBatch
  }

  /* Link libraries before tests */
  before(async () => {
    rollupMerkleUtils = await deployContract(wallet, RollupMerkleUtils, [], {
      gasLimit: 6700000,
    })
  })

  /* Deploy a new RollupChain before each test */
  beforeEach(async () => {
    canonicalTxChain = await deployContract(
      wallet,
      CanonicalTransactionChain,
      [
        rollupMerkleUtils.address,
        sequencer.address,
        l1ToL2TransactionPasser.address,
        LIVENESS_ASSUMPTION,
      ],
      {
        gasLimit: 6700000,
      }
    )

    const l1ToL2QueueAddress = await canonicalTxChain.l1ToL2Queue()
    l1ToL2Queue = new Contract(
      l1ToL2QueueAddress,
      L1ToL2TransactionQueue.abi,
      provider
    )

    const safetyQueueAddress = await canonicalTxChain.safetyQueue()
    safetyQueue = new Contract(
      safetyQueueAddress,
      SafetyTransactionQueue.abi,
      provider
    )
  })

  describe('appendTransactionBatch()', async () => {
    it('should not throw when appending a batch from the sequencer', async () => {
      await appendBatch(DEFAULT_BATCH)
    })

    it('should throw if submitting an empty batch', async () => {
      const emptyBatch = []
      await TestUtils.assertRevertsAsync(
        'Cannot submit an empty batch',
        async () => {
          await appendBatch(emptyBatch)
        }
      )
    })

    it('should revert if submitting a batch older than the inclusion period', async () => {
      const timestamp = Math.floor(Date.now() / 1000)
      const oldTimestamp = timestamp - (LIVENESS_ASSUMPTION + 1)
      await TestUtils.assertRevertsAsync(
        'Cannot submit a batch with a timestamp older than the sequencer inclusion period',
        async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendTransactionBatch(DEFAULT_BATCH, oldTimestamp)
        }
      )
    })

    it('should not revert if submitting a 5 minute old batch', async () => {
      const timestamp = Math.floor(Date.now() / 1000)
      const oldTimestamp = timestamp - LIVENESS_ASSUMPTION / 2
      await canonicalTxChain
        .connect(sequencer)
        .appendTransactionBatch(DEFAULT_BATCH, oldTimestamp)
    })

    it('should revert if submitting a batch with a future timestamp', async () => {
      const timestamp = Math.floor(Date.now() / 1000)
      const futureTimestamp = timestamp + 100
      await TestUtils.assertRevertsAsync(
        'Cannot submit a batch with a timestamp in the future',
        async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendTransactionBatch(DEFAULT_BATCH, futureTimestamp)
        }
      )
    })

    it('should revert if submitting a new batch with a timestamp less than latest batch timestamp', async () => {
      const timestamp = await appendBatch(DEFAULT_BATCH)
      const oldTimestamp = timestamp - 1
      await TestUtils.assertRevertsAsync(
        'Timestamps must monotonically increase',
        async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendTransactionBatch(DEFAULT_BATCH, oldTimestamp)
        }
      )
    })

    it('should add to batches array', async () => {
      await appendBatch(DEFAULT_BATCH)
      const batchesLength = await canonicalTxChain.getBatchesLength()
      batchesLength.toNumber().should.equal(1)
    })

    it('should update cumulativeNumElements correctly', async () => {
      await appendBatch(DEFAULT_BATCH)
      const cumulativeNumElements = await canonicalTxChain.cumulativeNumElements.call()
      cumulativeNumElements.toNumber().should.equal(DEFAULT_BATCH.length)
    })

    it('should not allow appendTransactionBatch from non-sequencer', async () => {
      const timestamp = Math.floor(Date.now() / 1000)
      await TestUtils.assertRevertsAsync(
        'Message sender does not have permission to append a batch',
        async () => {
          await canonicalTxChain.appendTransactionBatch(
            DEFAULT_BATCH,
            timestamp
          )
        }
      )
    })

    it('should calculate batchHeaderHash correctly', async () => {
      const localBatch = await appendAndGenerateBatch(DEFAULT_BATCH)
      const expectedBatchHeaderHash = await localBatch.hashBatchHeader()
      const calculatedBatchHeaderHash = await canonicalTxChain.batches(0)
      calculatedBatchHeaderHash.should.equal(expectedBatchHeaderHash)
    })

    it('should add multiple batches correctly', async () => {
      const numBatchs = 10
      for (let batchIndex = 0; batchIndex < numBatchs; batchIndex++) {
        const cumulativePrevElements = DEFAULT_BATCH.length * batchIndex
        const localBatch = await appendAndGenerateBatch(
          DEFAULT_BATCH,
          batchIndex,
          cumulativePrevElements
        )
        const expectedBatchHeaderHash = await localBatch.hashBatchHeader()
        const calculatedBatchHeaderHash = await canonicalTxChain.batches(
          batchIndex
        )
        calculatedBatchHeaderHash.should.equal(expectedBatchHeaderHash)
      }
      const cumulativeNumElements = await canonicalTxChain.cumulativeNumElements.call()
      cumulativeNumElements
        .toNumber()
        .should.equal(numBatchs * DEFAULT_BATCH.length)
      const batchesLength = await canonicalTxChain.getBatchesLength()
      batchesLength.toNumber().should.equal(numBatchs)
    })

    describe('when there is a batch in the L1toL2Queue', async () => {
      let localBatch
      beforeEach(async () => {
        localBatch = await enqueueAndGenerateL1ToL2Batch(DEFAULT_TX)
      })

      it('should succesfully append a batch with an older timestamp', async () => {
        const oldTimestamp = localBatch.timestamp - 1
        await canonicalTxChain
          .connect(sequencer)
          .appendTransactionBatch(DEFAULT_BATCH, oldTimestamp)
      })

      it('should succesfully append a batch with an equal timestamp', async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendTransactionBatch(DEFAULT_BATCH, localBatch.timestamp)
      })

      it('should revert when there is an older batch in the L1ToL2Queue', async () => {
        const snapshotID = await provider.send('evm_snapshot', [])
        await provider.send('evm_increaseTime', [LIVENESS_ASSUMPTION])
        const newTimestamp = localBatch.timestamp + 60
        await TestUtils.assertRevertsAsync(
          'Must process older L1ToL2Queue batches first to enforce timestamp monotonicity',
          async () => {
            await canonicalTxChain
              .connect(sequencer)
              .appendTransactionBatch(DEFAULT_BATCH, newTimestamp)
          }
        )
        await provider.send('evm_revert', [snapshotID])
      })
    })

    describe('when there is a batch in the SafetyQueue', async () => {
      let localBatch
      beforeEach(async () => {
        localBatch = await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
      })

      it('should succesfully append a batch with an older timestamp', async () => {
        const oldTimestamp = localBatch.timestamp - 1
        await canonicalTxChain
          .connect(sequencer)
          .appendTransactionBatch(DEFAULT_BATCH, oldTimestamp)
      })

      it('should succesfully append a batch with an equal timestamp', async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendTransactionBatch(DEFAULT_BATCH, localBatch.timestamp)
      })

      it('should revert when there is an older batch in the SafetyQueue', async () => {
        const snapshotID = await provider.send('evm_snapshot', [])
        await provider.send('evm_increaseTime', [LIVENESS_ASSUMPTION])
        const newTimestamp = localBatch.timestamp + 60
        await TestUtils.assertRevertsAsync(
          'Must process older SafetyQueue batches first to enforce timestamp monotonicity',
          async () => {
            await canonicalTxChain
              .connect(sequencer)
              .appendTransactionBatch(DEFAULT_BATCH, newTimestamp)
          }
        )
        await provider.send('evm_revert', [snapshotID])
      })
    })
    describe('when both the safetyQueue and l1ToL2Queue are not empty', async () => {
      let timestamp
      beforeEach(async () => {
        const localBatch = await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
        timestamp = localBatch.timestamp
        await enqueueAndGenerateL1ToL2Batch(DEFAULT_TX)
      })

      it('should succesfully append a batch with an older timestamp than the oldest batch', async () => {
        const oldTimestamp = timestamp - 1
        await canonicalTxChain
          .connect(sequencer)
          .appendTransactionBatch(DEFAULT_BATCH, oldTimestamp)
      })

      it('should succesfully append a batch with a timestamp equal to the oldest batch', async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendTransactionBatch(DEFAULT_BATCH, timestamp)
      })
    })
  })

  describe('appendL1ToL2Batch()', async () => {
    describe('when there is a batch in the L1toL2Queue', async () => {
      beforeEach(async () => {
        await enqueueAndGenerateL1ToL2Batch(DEFAULT_TX)
      })

      it('should successfully dequeue a L1ToL2Batch', async () => {
        await canonicalTxChain.connect(sequencer).appendL1ToL2Batch()
        const front = await l1ToL2Queue.front()
        front.should.equal(1)
        const { timestamp, txHash } = await l1ToL2Queue.batchHeaders(0)
        timestamp.should.equal(0)
        txHash.should.equal(
          '0x0000000000000000000000000000000000000000000000000000000000000000'
        )
      })

      it('should successfully append a L1ToL2Batch', async () => {
        const { timestamp, txHash } = await l1ToL2Queue.batchHeaders(0)
        const localBatch = new CanonicalTxBatch(
          timestamp,
          true, // isL1ToL2Tx
          0, //batchIndex
          0, // cumulativePrevElements
          [DEFAULT_TX] // elements
        )
        await localBatch.generateTree()
        const localBatchHeaderHash = await localBatch.hashBatchHeader()
        await canonicalTxChain.connect(sequencer).appendL1ToL2Batch()
        const batchHeaderHash = await canonicalTxChain.batches(0)
        batchHeaderHash.should.equal(localBatchHeaderHash)
      })

      it('should not allow non-sequencer to appendL1ToL2Batch if less than the inclusion period', async () => {
        await TestUtils.assertRevertsAsync(
          'Message sender does not have permission to append this batch',
          async () => {
            await canonicalTxChain.appendL1ToL2Batch()
          }
        )
      })

      it('should allow non-sequencer to appendL1ToL2Batch after inclusion period has elapsed', async () => {
        const snapshotID = await provider.send('evm_snapshot', [])
        await provider.send('evm_increaseTime', [LIVENESS_ASSUMPTION])
        await canonicalTxChain.appendL1ToL2Batch()
        await provider.send('evm_revert', [snapshotID])
      })
    })

    it('should revert when there is an older batch in the SafetyQueue ', async () => {
      const snapshotID = await provider.send('evm_snapshot', [])
      await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
      await provider.send('evm_increaseTime', [10])
      await enqueueAndGenerateL1ToL2Batch(DEFAULT_TX)
      await TestUtils.assertRevertsAsync(
        'Must process older SafetyQueue batches first to enforce timestamp monotonicity',
        async () => {
          await canonicalTxChain.appendL1ToL2Batch()
        }
      )
      await provider.send('evm_revert', [snapshotID])
    })

    it('should succeed when there are only newer batches in the SafetyQueue ', async () => {
      const snapshotID = await provider.send('evm_snapshot', [])
      await enqueueAndGenerateL1ToL2Batch(DEFAULT_TX)
      await provider.send('evm_increaseTime', [10])
      await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
      await canonicalTxChain.connect(sequencer).appendL1ToL2Batch()
      await provider.send('evm_revert', [snapshotID])
    })

    it('should revert when L1ToL2TxQueue is empty', async () => {
      await TestUtils.assertRevertsAsync(
        'Queue is empty, no element to peek at',
        async () => {
          await canonicalTxChain.appendL1ToL2Batch()
        }
      )
    })
  })

  describe('appendSafetyBatch()', async () => {
    describe('when there is a batch in the SafetyQueue', async () => {
      beforeEach(async () => {
        await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
      })

      it('should successfully dequeue a SafetyBatch', async () => {
        await canonicalTxChain.connect(sequencer).appendSafetyBatch()
        const front = await safetyQueue.front()
        front.should.equal(1)
        const { timestamp, txHash } = await safetyQueue.batchHeaders(0)
        timestamp.should.equal(0)
        txHash.should.equal(
          '0x0000000000000000000000000000000000000000000000000000000000000000'
        )
      })

      it('should successfully append a SafetyBatch', async () => {
        const { timestamp, txHash } = await safetyQueue.batchHeaders(0)
        const localBatch = new CanonicalTxBatch(
          timestamp,
          false, // isL1ToL2Tx
          0, //batchIndex
          0, // cumulativePrevElements
          [DEFAULT_TX] // elements
        )
        await localBatch.generateTree()
        const localBatchHeaderHash = await localBatch.hashBatchHeader()
        await canonicalTxChain.connect(sequencer).appendSafetyBatch()
        const batchHeaderHash = await canonicalTxChain.batches(0)
        batchHeaderHash.should.equal(localBatchHeaderHash)
      })

      it('should not allow non-sequencer to appendSafetyBatch if less than 10 minutes old', async () => {
        await TestUtils.assertRevertsAsync(
          'Message sender does not have permission to append this batch',
          async () => {
            await canonicalTxChain.appendSafetyBatch()
          }
        )
      })
      it('should allow non-sequencer to appendSafetyBatch after 10 minutes have elapsed', async () => {
        const snapshotID = await provider.send('evm_snapshot', [])
        await provider.send('evm_increaseTime', [LIVENESS_ASSUMPTION])
        await canonicalTxChain.appendSafetyBatch()
        await provider.send('evm_revert', [snapshotID])
      })
    })

    it('should revert when trying to appendSafetyBatch when there is an older batch in the L1ToL2Queue ', async () => {
      const snapshotID = await provider.send('evm_snapshot', [])
      await enqueueAndGenerateL1ToL2Batch(DEFAULT_TX)
      await provider.send('evm_increaseTime', [10])
      await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
      await TestUtils.assertRevertsAsync(
        'Must process older L1ToL2Queue batches first to enforce timestamp monotonicity',
        async () => {
          await canonicalTxChain.appendSafetyBatch()
        }
      )
      await provider.send('evm_revert', [snapshotID])
    })

    it('should succeed when there are only newer batches in the L1ToL2Queue ', async () => {
      const snapshotID = await provider.send('evm_snapshot', [])
      await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
      await provider.send('evm_increaseTime', [10])
      await enqueueAndGenerateL1ToL2Batch(DEFAULT_TX)
      await canonicalTxChain.connect(sequencer).appendSafetyBatch()
      await provider.send('evm_revert', [snapshotID])
    })

    it('should revert when SafetyTxQueue is empty', async () => {
      await TestUtils.assertRevertsAsync(
        'Queue is empty, no element to peek at',
        async () => {
          await canonicalTxChain.appendSafetyBatch()
        }
      )
    })
  })

  describe('verifyElement() ', async () => {
    it('should return true for valid elements for different batches and elements', async () => {
      const numBatches = 3
      const batch = [
        '0x1234',
        '0x4567',
        '0x890a',
        '0x4567',
        '0x890a',
        '0xabcd',
        '0x1234',
      ]
      for (let batchIndex = 0; batchIndex < numBatches; batchIndex++) {
        const cumulativePrevElements = batch.length * batchIndex
        const localBatch = await appendAndGenerateBatch(
          batch,
          batchIndex,
          cumulativePrevElements
        )
        for (
          let elementIndex = 0;
          elementIndex < batch.length;
          elementIndex += 3
        ) {
          const element = batch[elementIndex]
          const position = localBatch.getPosition(elementIndex)
          const elementInclusionProof = await localBatch.getElementInclusionProof(
            elementIndex
          )
          const isIncluded = await canonicalTxChain.verifyElement(
            element,
            position,
            elementInclusionProof
          )
          isIncluded.should.equal(true)
        }
      }
    })

    it('should return true for valid element from a l1ToL2Batch', async () => {
      const l1ToL2Batch = await enqueueAndGenerateL1ToL2Batch(DEFAULT_TX)
      await canonicalTxChain.connect(sequencer).appendL1ToL2Batch()
      const localBatch = new CanonicalTxBatch(
        l1ToL2Batch.timestamp, //timestamp
        true, //isL1ToL2Tx
        0, //batchIndex
        0, //cumulativePrevElements
        [DEFAULT_TX] //batch
      )
      await localBatch.generateTree()
      const elementIndex = 0
      const position = localBatch.getPosition(elementIndex)
      const elementInclusionProof = await localBatch.getElementInclusionProof(
        elementIndex
      )
      const isIncluded = await canonicalTxChain.verifyElement(
        DEFAULT_TX, // element
        position,
        elementInclusionProof
      )
      isIncluded.should.equal(true)
    })

    it('should return true for valid element from a SafetyBatch', async () => {
      const safetyBatch = await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
      await canonicalTxChain.connect(sequencer).appendSafetyBatch()
      const localBatch = new CanonicalTxBatch(
        safetyBatch.timestamp, //timestamp
        false, //isL1ToL2Tx
        0, //batchIndex
        0, //cumulativePrevElements
        [DEFAULT_TX] //batch
      )
      await localBatch.generateTree()
      const elementIndex = 0
      const position = localBatch.getPosition(elementIndex)
      const elementInclusionProof = await localBatch.getElementInclusionProof(
        elementIndex
      )
      const isIncluded = await canonicalTxChain.verifyElement(
        DEFAULT_TX, // element
        position,
        elementInclusionProof
      )
      isIncluded.should.equal(true)
    })

    it('should return false for wrong position with wrong indexInBatch', async () => {
      const batch = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
      const localBatch = await appendAndGenerateBatch(batch)
      const elementIndex = 1
      const element = batch[elementIndex]
      const position = localBatch.getPosition(elementIndex)
      const elementInclusionProof = await localBatch.getElementInclusionProof(
        elementIndex
      )
      //Give wrong position so inclusion proof is wrong
      const wrongPosition = position + 1
      const isIncluded = await canonicalTxChain.verifyElement(
        element,
        wrongPosition,
        elementInclusionProof
      )
      isIncluded.should.equal(false)
    })

    it('should return false for wrong position and matching indexInBatch', async () => {
      const batch = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
      const localBatch = await appendAndGenerateBatch(batch)
      const elementIndex = 1
      const element = batch[elementIndex]
      const position = localBatch.getPosition(elementIndex)
      const elementInclusionProof = await localBatch.getElementInclusionProof(
        elementIndex
      )
      //Give wrong position so inclusion proof is wrong
      const wrongPosition = position + 1
      //Change index to also be false (so position = index + cumulative)
      elementInclusionProof.indexInBatch++
      const isIncluded = await canonicalTxChain.verifyElement(
        element,
        wrongPosition,
        elementInclusionProof
      )
      isIncluded.should.equal(false)
    })
  })
})
