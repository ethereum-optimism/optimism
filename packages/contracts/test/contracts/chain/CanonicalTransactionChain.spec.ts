import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, sleep, TestUtils } from '@eth-optimism/core-utils'
import { Contract, Signer, ContractFactory } from 'ethers'

/* Internal Imports */
import {
  makeRandomBatchOfSize,
  TxQueueBatch,
  TxChainBatch,
  makeAddressResolver,
  deployAndRegister,
  AddressResolverMapping,
} from '../../test-helpers'

/* Logging */
const log = getLogger('canonical-tx-chain', true)

/* Tests */
describe('CanonicalTransactionChain', () => {
  const provider = ethers.provider
  const FORCE_INCLUSION_PERIOD = 600 //600 seconds = 10 minutes
  const DEFAULT_BATCH = ['0x1234', '0x5678']
  const DEFAULT_TX = '0x1234'

  let wallet: Signer
  let sequencer: Signer
  let l1ToL2TransactionPasser: Signer
  let randomWallet: Signer
  before(async () => {
    ;[
      wallet,
      sequencer,
      l1ToL2TransactionPasser,
      randomWallet,
    ] = await ethers.getSigners()
  })

  let canonicalTxChain: Contract
  let l1ToL2Queue: Contract
  let safetyQueue: Contract

  const appendSequencerBatch = async (batch: string[]): Promise<number> => {
    const timestamp = Math.floor(Date.now() / 1000)
    // Submit the rollup batch on-chain
    await canonicalTxChain
      .connect(sequencer)
      .appendSequencerBatch(batch, timestamp)
    return timestamp
  }

  const appendAndGenerateSequencerBatch = async (
    batch: string[],
    batchIndex: number = 0,
    cumulativePrevElements: number = 0
  ): Promise<TxChainBatch> => {
    const timestamp = await appendSequencerBatch(batch)
    return createTxChainBatch(
      batch,
      timestamp,
      false,
      batchIndex,
      cumulativePrevElements
    )
  }

  const createTxChainBatch = async (
    batch: string[],
    timestamp: number,
    isL1ToL2Tx: boolean,
    batchIndex: number = 0,
    cumulativePrevElements: number = 0
  ): Promise<TxChainBatch> => {
    const localBatch = new TxChainBatch(
      timestamp,
      isL1ToL2Tx,
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

  let resolver: AddressResolverMapping
  before(async () => {
    resolver = await makeAddressResolver(wallet)
  })

  let CanonicalTransactionChain: ContractFactory
  let L1ToL2TransactionQueue: ContractFactory
  let SafetyTransactionQueue: ContractFactory
  before(async () => {
    CanonicalTransactionChain = await ethers.getContractFactory(
      'CanonicalTransactionChain'
    )
    L1ToL2TransactionQueue = await ethers.getContractFactory(
      'L1ToL2TransactionQueue'
    )
    SafetyTransactionQueue = await ethers.getContractFactory(
      'SafetyTransactionQueue'
    )
  })

  beforeEach(async () => {
    canonicalTxChain = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'CanonicalTransactionChain',
      {
        factory: CanonicalTransactionChain,
        params: [
          resolver.addressResolver.address,
          await sequencer.getAddress(),
          FORCE_INCLUSION_PERIOD,
        ],
      }
    )

    l1ToL2Queue = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'L1ToL2TransactionQueue',
      {
        factory: L1ToL2TransactionQueue,
        params: [resolver.addressResolver.address],
      }
    )

    safetyQueue = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'SafetyTransactionQueue',
      {
        factory: SafetyTransactionQueue,
        params: [resolver.addressResolver.address],
      }
    )
  })

  describe('appendSequencerBatch()', async () => {
    it('should not throw when appending a batch from the sequencer', async () => {
      await appendSequencerBatch(DEFAULT_BATCH)
    })

    it('should throw if submitting an empty batch', async () => {
      const emptyBatch = []
      await TestUtils.assertRevertsAsync(
        'Cannot submit an empty batch',
        async () => {
          await appendSequencerBatch(emptyBatch)
        }
      )
    })

    it('should revert if submitting a batch older than the inclusion period', async () => {
      const timestamp = Math.floor(Date.now() / 1000)
      const oldTimestamp = timestamp - (FORCE_INCLUSION_PERIOD + 1)
      await TestUtils.assertRevertsAsync(
        'Cannot submit a batch with a timestamp older than the sequencer inclusion period',
        async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(DEFAULT_BATCH, oldTimestamp)
        }
      )
    })

    it('should not revert if submitting a 5 minute old batch', async () => {
      const timestamp = Math.floor(Date.now() / 1000)
      const oldTimestamp = timestamp - FORCE_INCLUSION_PERIOD / 2
      await canonicalTxChain
        .connect(sequencer)
        .appendSequencerBatch(DEFAULT_BATCH, oldTimestamp)
    })

    it('should revert if submitting a batch with a future timestamp', async () => {
      const timestamp = Math.floor(Date.now() / 1000)
      const futureTimestamp = timestamp + 100
      await TestUtils.assertRevertsAsync(
        'Cannot submit a batch with a timestamp in the future',
        async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(DEFAULT_BATCH, futureTimestamp)
        }
      )
    })

    it('should revert if submitting a new batch with a timestamp older than last batch timestamp', async () => {
      const timestamp = await appendSequencerBatch(DEFAULT_BATCH)
      const oldTimestamp = timestamp - 1
      await TestUtils.assertRevertsAsync(
        'Timestamps must monotonically increase',
        async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(DEFAULT_BATCH, oldTimestamp)
        }
      )
    })

    it('should add to batches array', async () => {
      await appendSequencerBatch(DEFAULT_BATCH)
      const batchesLength = await canonicalTxChain.getBatchesLength()
      batchesLength.toNumber().should.equal(1)
    })

    it('should update cumulativeNumElements correctly', async () => {
      await appendSequencerBatch(DEFAULT_BATCH)
      const cumulativeNumElements = await canonicalTxChain.cumulativeNumElements.call()
      cumulativeNumElements.toNumber().should.equal(DEFAULT_BATCH.length)
    })

    it('should not allow appendSequencerBatch from non-sequencer', async () => {
      const timestamp = Math.floor(Date.now() / 1000)
      await TestUtils.assertRevertsAsync(
        'Message sender does not have permission to append a batch',
        async () => {
          await canonicalTxChain.appendSequencerBatch(DEFAULT_BATCH, timestamp)
        }
      )
    })

    it('should calculate batchHeaderHash correctly', async () => {
      const localBatch = await appendAndGenerateSequencerBatch(DEFAULT_BATCH)
      const expectedBatchHeaderHash = await localBatch.hashBatchHeader()
      const calculatedBatchHeaderHash = await canonicalTxChain.batches(0)
      calculatedBatchHeaderHash.should.equal(expectedBatchHeaderHash)
    })

    it('should add multiple batches correctly', async () => {
      const numBatches = 5
      let expectedNumElements = 0
      for (let batchIndex = 0; batchIndex < numBatches; batchIndex++) {
        const batch = makeRandomBatchOfSize(batchIndex + 1)
        const cumulativePrevElements = expectedNumElements
        const localBatch = await appendAndGenerateSequencerBatch(
          batch,
          batchIndex,
          cumulativePrevElements
        )
        const expectedBatchHeaderHash = await localBatch.hashBatchHeader()
        const calculatedBatchHeaderHash = await canonicalTxChain.batches(
          batchIndex
        )
        calculatedBatchHeaderHash.should.equal(expectedBatchHeaderHash)
        expectedNumElements += batch.length
      }
      const cumulativeNumElements = await canonicalTxChain.cumulativeNumElements.call()
      cumulativeNumElements.toNumber().should.equal(expectedNumElements)
      const batchesLength = await canonicalTxChain.getBatchesLength()
      batchesLength.toNumber().should.equal(numBatches)
    })

    describe('when there is a batch in the L1toL2Queue', async () => {
      let localBatch
      beforeEach(async () => {
        localBatch = await enqueueAndGenerateL1ToL2Batch(DEFAULT_TX)
      })

      it('should successfully append a batch with an older timestamp', async () => {
        const oldTimestamp = localBatch.timestamp - 1
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, oldTimestamp)
      })

      it('should successfully append a batch with an equal timestamp', async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, localBatch.timestamp)
      })

      it('should revert when there is an older batch in the L1ToL2Queue', async () => {
        const snapshotID = await provider.send('evm_snapshot', [])
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD])
        const newTimestamp = localBatch.timestamp + 60
        await TestUtils.assertRevertsAsync(
          'Must process older L1ToL2Queue batches first to enforce timestamp monotonicity',
          async () => {
            await canonicalTxChain
              .connect(sequencer)
              .appendSequencerBatch(DEFAULT_BATCH, newTimestamp)
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

      it('should successfully append a batch with an older timestamp', async () => {
        const oldTimestamp = localBatch.timestamp - 1
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, oldTimestamp)
      })

      it('should successfully append a batch with an equal timestamp', async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, localBatch.timestamp)
      })

      it('should revert when there is an older batch in the SafetyQueue', async () => {
        const snapshotID = await provider.send('evm_snapshot', [])
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD])
        const newTimestamp = localBatch.timestamp + 60
        await TestUtils.assertRevertsAsync(
          'Must process older SafetyQueue batches first to enforce timestamp monotonicity',
          async () => {
            await canonicalTxChain
              .connect(sequencer)
              .appendSequencerBatch(DEFAULT_BATCH, newTimestamp)
          }
        )
        await provider.send('evm_revert', [snapshotID])
      })
    })
    describe('when there is an old batch in the safetyQueue and a recent batch in the l1ToL2Queue', async () => {
      let safetyTimestamp
      let l1ToL2Timestamp
      let snapshotID
      beforeEach(async () => {
        const localSafetyBatch = await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
        safetyTimestamp = localSafetyBatch.timestamp
        snapshotID = await provider.send('evm_snapshot', [])
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD / 2])
        const localL1ToL2Batch = await enqueueAndGenerateL1ToL2Batch(DEFAULT_TX)
        l1ToL2Timestamp = localL1ToL2Batch.timestamp
      })
      afterEach(async () => {
        await provider.send('evm_revert', [snapshotID])
      })

      it('should successfully append a batch with an older timestamp than the oldest batch', async () => {
        const oldTimestamp = safetyTimestamp - 1
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, oldTimestamp)
      })

      it('should successfully append a batch with a timestamp equal to the oldest batch', async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, safetyTimestamp)
      })

      it('should revert when appending a batch with a timestamp in between the two batches', async () => {
        const middleTimestamp = safetyTimestamp + 1
        await TestUtils.assertRevertsAsync(
          'Must process older SafetyQueue batches first to enforce timestamp monotonicity',
          async () => {
            await canonicalTxChain
              .connect(sequencer)
              .appendSequencerBatch(DEFAULT_BATCH, middleTimestamp)
          }
        )
      })

      it('should revert when appending a batch with a timestamp newer than both batches', async () => {
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD / 10]) // increase time by 60 seconds
        const oldTimestamp = l1ToL2Timestamp + 1
        await TestUtils.assertRevertsAsync(
          'Must process older L1ToL2Queue batches first to enforce timestamp monotonicity',
          async () => {
            await canonicalTxChain
              .connect(sequencer)
              .appendSequencerBatch(DEFAULT_BATCH, oldTimestamp)
          }
        )
      })
    })

    describe('when there is an old batch in the l1ToL2Queue and a recent batch in the safetyQueue', async () => {
      let l1ToL2Timestamp
      let safetyTimestamp
      let snapshotID
      beforeEach(async () => {
        const localL1ToL2Batch = await enqueueAndGenerateL1ToL2Batch(DEFAULT_TX)
        l1ToL2Timestamp = localL1ToL2Batch.timestamp
        snapshotID = await provider.send('evm_snapshot', [])
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD / 2])
        const localSafetyBatch = await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
        safetyTimestamp = localSafetyBatch.timestamp
      })
      afterEach(async () => {
        await provider.send('evm_revert', [snapshotID])
      })

      it('should successfully append a batch with an older timestamp than both batches', async () => {
        const oldTimestamp = l1ToL2Timestamp - 1
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, oldTimestamp)
      })

      it('should successfully append a batch with a timestamp equal to the older batch', async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, l1ToL2Timestamp)
      })

      it('should revert when appending a batch with a timestamp in between the two batches', async () => {
        const middleTimestamp = l1ToL2Timestamp + 1
        await TestUtils.assertRevertsAsync(
          'Must process older L1ToL2Queue batches first to enforce timestamp monotonicity',
          async () => {
            await canonicalTxChain
              .connect(sequencer)
              .appendSequencerBatch(DEFAULT_BATCH, middleTimestamp)
          }
        )
      })

      it('should revert when appending a batch with a timestamp newer than both batches', async () => {
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD / 10]) // increase time by 60 seconds
        const newTimestamp = safetyTimestamp + 1
        await TestUtils.assertRevertsAsync(
          'Must process older L1ToL2Queue batches first to enforce timestamp monotonicity',
          async () => {
            await canonicalTxChain
              .connect(sequencer)
              .appendSequencerBatch(DEFAULT_BATCH, newTimestamp)
          }
        )
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
        const localBatch = new TxChainBatch(
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
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD])
        await canonicalTxChain.appendL1ToL2Batch()
        await provider.send('evm_revert', [snapshotID])
      })
    })

    describe('when there is a batch in both the SafetyQueue and L1toL2Queue', async () => {
      it('should revert when the SafetyQueue batch is older', async () => {
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

      it('should succeed when the L1ToL2Queue batch is older', async () => {
        const snapshotID = await provider.send('evm_snapshot', [])
        await enqueueAndGenerateL1ToL2Batch(DEFAULT_TX)
        await provider.send('evm_increaseTime', [10])
        await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
        await canonicalTxChain.connect(sequencer).appendL1ToL2Batch()
        await provider.send('evm_revert', [snapshotID])
      })
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
        const localBatch = new TxChainBatch(
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

      it('should not allow non-sequencer to appendSafetyBatch if less than force inclusion period', async () => {
        await TestUtils.assertRevertsAsync(
          'Message sender does not have permission to append this batch',
          async () => {
            await canonicalTxChain.appendSafetyBatch()
          }
        )
      })

      it('should allow non-sequencer to appendSafetyBatch after force inclusion period has elapsed', async () => {
        const snapshotID = await provider.send('evm_snapshot', [])
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD])
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
      const numBatches = 4
      let cumulativePrevElements = 0
      for (let batchIndex = 0; batchIndex < numBatches; batchIndex++) {
        const batchSize = batchIndex * batchIndex + 1 // 1, 2, 5, 10
        const batch = makeRandomBatchOfSize(batchSize)
        const localBatch = await appendAndGenerateSequencerBatch(
          batch,
          batchIndex,
          cumulativePrevElements
        )
        cumulativePrevElements += batchSize
        for (
          let elementIndex = 0;
          elementIndex < batch.length;
          elementIndex++
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
      const localBatch = new TxChainBatch(
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
      const localBatch = new TxChainBatch(
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
      const localBatch = await appendAndGenerateSequencerBatch(batch)
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
      const localBatch = await appendAndGenerateSequencerBatch(batch)
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

  describe('Event Emitting', () => {
    it('should emit SequencerBatchAppended event when appending sequencer batch', async () => {
      let receivedBatchHeaderHash: string
      canonicalTxChain.on(
        canonicalTxChain.filters['SequencerBatchAppended'](),
        (...data) => {
          receivedBatchHeaderHash = data[0]
        }
      )
      const localBatch: TxChainBatch = await appendAndGenerateSequencerBatch(
        DEFAULT_BATCH
      )

      await sleep(5_000)

      const received = !!receivedBatchHeaderHash
      received.should.equal(true, `Did not receive expected event!`)

      receivedBatchHeaderHash.should.equal(
        await localBatch.hashBatchHeader(),
        'Header hash mismatch!'
      )
    })

    it('should emit CalldataTxEnqueued event when enqueuing safety batch', async () => {
      let txEnqueued: boolean = false
      safetyQueue.on(safetyQueue.filters['CalldataTxEnqueued'](), () => {
        txEnqueued = true
      })

      await enqueueAndGenerateSafetyBatch(DEFAULT_TX)

      await sleep(5_000)

      txEnqueued.should.equal(true, `Did not receive expected event!`)
    })

    it('should emit L1ToL2TxEnqueued event when enqueuing L1 To L2 batch', async () => {
      let enqueuedTx: string
      l1ToL2Queue.on(l1ToL2Queue.filters['L1ToL2TxEnqueued'](), (...data) => {
        enqueuedTx = data[0]
      })

      const localBatch: TxQueueBatch = await enqueueAndGenerateL1ToL2Batch(
        DEFAULT_TX
      )

      await sleep(5_000)

      const receivedTx: boolean = !!enqueuedTx
      receivedTx.should.equal(true, `Did not receive expected event!`)

      enqueuedTx.should.equal(
        localBatch.elements[0],
        `Emitted tx did not match submitted tx!`
      )
    })

    it('should emit L1ToL2BatchAppended event when appending L1 to L2 batch', async () => {
      let receivedBatchHeaderHash: string
      canonicalTxChain.on(
        canonicalTxChain.filters['L1ToL2BatchAppended'](),
        (...data) => {
          receivedBatchHeaderHash = data[0]
        }
      )

      const localBatch: TxQueueBatch = await enqueueAndGenerateL1ToL2Batch(
        DEFAULT_TX
      )
      await canonicalTxChain.connect(sequencer).appendL1ToL2Batch()
      const front = await l1ToL2Queue.front()
      front.should.equal(1)
      const { timestamp, txHash } = await l1ToL2Queue.batchHeaders(0)
      timestamp.should.equal(0)
      txHash.should.equal(
        '0x0000000000000000000000000000000000000000000000000000000000000000'
      )

      await sleep(5_000)

      const received = !!receivedBatchHeaderHash
      received.should.equal(true, `Did not receive expected event!`)

      receivedBatchHeaderHash.should.equal(
        await localBatch.hashBatchHeader(true),
        `Incorrect batch header hash!`
      )
    })

    it('should emit SafetyQueueBatchAppended event when appending Safety Queue batch', async () => {
      let receivedBatchHeaderHash: string
      canonicalTxChain.on(
        canonicalTxChain.filters['SafetyQueueBatchAppended'](),
        (...data) => {
          receivedBatchHeaderHash = data[0]
        }
      )

      const localBatch = await enqueueAndGenerateSafetyBatch(DEFAULT_TX)

      await canonicalTxChain.connect(sequencer).appendSafetyBatch()
      const front = await safetyQueue.front()
      front.should.equal(1)
      const { timestamp, txHash } = await safetyQueue.batchHeaders(0)
      timestamp.should.equal(0)
      txHash.should.equal(
        '0x0000000000000000000000000000000000000000000000000000000000000000'
      )

      await sleep(5_000)

      const received = !!receivedBatchHeaderHash
      received.should.equal(true, `Did not receive expected event!`)

      receivedBatchHeaderHash.should.equal(
        await localBatch.hashBatchHeader(false),
        `Incorrect batch header hash!`
      )
    })
  })
})
