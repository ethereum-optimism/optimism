import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import {
  getLogger,
  sleep,
  TestUtils,
  ZERO_ADDRESS,
} from '@eth-optimism/core-utils'
import { Contract, Signer, ContractFactory } from 'ethers'

/* Internal Imports */
import {
  makeRandomBatchOfSize,
  TxQueueBatch,
  TxChainBatch,
  makeAddressResolver,
  deployAndRegister,
  AddressResolverMapping,
  GET_DUMMY_TX_WITH_OVM_GAS_LIMIT,
  getL1ToL2MessageTxData,
} from '../../test-helpers'

/* Logging */
const log = getLogger('canonical-tx-chain', true)

const abi = new ethers.utils.AbiCoder()

/* Tests */
describe('CanonicalTransactionChain', () => {
  const provider = ethers.provider
  const FORCE_INCLUSION_PERIOD = 4000
  const DEFAULT_BATCH = [
    GET_DUMMY_TX_WITH_OVM_GAS_LIMIT(30_000),
    GET_DUMMY_TX_WITH_OVM_GAS_LIMIT(35_000),
  ]
  const DEFAULT_TX = GET_DUMMY_TX_WITH_OVM_GAS_LIMIT(30_000)
  const DEFAULT_L1_L2_MESSAGE_PARAMS = [ZERO_ADDRESS, 30_000, '0x12341234']

  let wallet: Signer
  let sequencer: Signer
  let l1ToL2TransactionPasser: Signer
  let randomWallet: Signer

  let canonicalTxChain: Contract
  let l1ToL2Queue: Contract
  let safetyQueue: Contract
  let resolver: AddressResolverMapping
  let CanonicalTransactionChain: ContractFactory
  let L1ToL2TransactionQueue: ContractFactory
  let SafetyTransactionQueue: ContractFactory

  before(async () => {
    ;[
      wallet,
      sequencer,
      l1ToL2TransactionPasser,
      randomWallet,
    ] = await ethers.getSigners()

    resolver = await makeAddressResolver(wallet)

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

  /* HELPERS */

  const appendSequencerBatch = async (
    batch: string[],
    txStartIndex: number
  ): Promise<number[]> => {
    const blockNumber = await provider.getBlockNumber()
    const timestamp = Math.floor(Date.now() / 1000)
    // Submit the rollup batch on-chain
    await canonicalTxChain
      .connect(sequencer)
      .appendSequencerBatch(batch, timestamp, blockNumber, txStartIndex)
    return [timestamp, blockNumber]
  }

  const appendAndGenerateSequencerBatch = async (
    batch: string[],
    batchIndex: number = 0,
    cumulativePrevElements: number = 0
  ): Promise<TxChainBatch> => {
    const [timestamp, blockNumber] = await appendSequencerBatch(
      batch,
      cumulativePrevElements
    )
    return createTxChainBatch(
      batch,
      timestamp,
      blockNumber,
      false,
      batchIndex,
      cumulativePrevElements
    )
  }

  const createTxChainBatch = async (
    batch: string[],
    timestamp: number,
    blockNumber,
    isL1ToL2Tx: boolean,
    batchIndex: number = 0,
    cumulativePrevElements: number = 0
  ): Promise<TxChainBatch> => {
    const localBatch = new TxChainBatch(
      timestamp,
      blockNumber,
      isL1ToL2Tx,
      batchIndex,
      cumulativePrevElements,
      batch
    )
    await localBatch.generateTree()
    return localBatch
  }

  const enqueueAndGenerateL1ToL2Batch = async (
    l1ToL2Params: any[]
  ): Promise<TxQueueBatch> => {
    // Submit the rollup batch on-chain
    const enqueueTx = await l1ToL2Queue
      .connect(l1ToL2TransactionPasser)
      .enqueueL1ToL2Message(...l1ToL2Params)
    const localBatch = await generateL1ToL2Batch(l1ToL2Params, enqueueTx.hash)
    return localBatch
  }

  const generateL1ToL2Batch = async (
    _l1ToL2Params: any[],
    _enqueueTxHash: string
  ): Promise<TxQueueBatch> => {
    const txReceipt = await provider.getTransactionReceipt(_enqueueTxHash)
    const sender = txReceipt.from
    const rolledupData = abi.encode(
      ['address', 'address', 'uint32', 'bytes'],
      [sender, ..._l1ToL2Params]
    )
    // Generate a local version of the rollup batch
    const timestamp = (await provider.getBlock(txReceipt.blockNumber)).timestamp
    const localBatch = new TxQueueBatch(
      rolledupData,
      timestamp,
      txReceipt.blockNumber
    )
    await localBatch.generateTree()
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
    const localBatch = new TxQueueBatch(_tx, timestamp, txReceipt.blockNumber)
    await localBatch.generateTree()
    return localBatch
  }

  /* TESTS */

  describe('appendSequencerBatch()', async () => {
    it('should not throw when appending a batch from the sequencer', async () => {
      await appendSequencerBatch(DEFAULT_BATCH, 0)
    })

    it('should throw if submitting an empty batch', async () => {
      const emptyBatch = []
      await TestUtils.assertRevertsAsync(async () => {
        await appendSequencerBatch(emptyBatch, 0)
      }, 'Cannot submit an empty batch')
    })

    it('should revert if submitting a batch with timestamp older than the inclusion period', async () => {
      const timestamp = Math.floor(Date.now() / 1000)
      const blockNumber = Math.floor(timestamp / 15)
      const oldTimestamp = timestamp - (FORCE_INCLUSION_PERIOD + 1000)
      await TestUtils.assertRevertsAsync(async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, oldTimestamp, blockNumber, 0)
      }, 'Cannot submit a batch with a timestamp older than the sequencer inclusion period')
    })

    it('should revert if submitting a batch with blockNumber older than the inclusion period', async () => {
      const timestamp = Math.floor(Date.now() / 1000)
      const FORCE_INCLUSION_PERIOD_BLOCKS = await canonicalTxChain.forceInclusionPeriodBlocks()
      for (let i = 0; i < FORCE_INCLUSION_PERIOD_BLOCKS + 1; i++) {
        await provider.send('evm_mine', [])
      }
      const currentBlockNumber = await canonicalTxChain.provider.getBlockNumber()
      await TestUtils.assertRevertsAsync(async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(
            DEFAULT_BATCH,
            timestamp,
            currentBlockNumber - FORCE_INCLUSION_PERIOD_BLOCKS,
            0
          )
      }, 'Cannot submit a batch with a blockNumber older than the sequencer inclusion period')
    })

    it('should not revert if submitting an INCLUSION_PERIOD/2 old batch', async () => {
      const blockNumber = await provider.getBlockNumber()
      const timestamp = (await provider.getBlock(blockNumber)).timestamp
      const oldTimestamp = timestamp - FORCE_INCLUSION_PERIOD / 2
      await canonicalTxChain
        .connect(sequencer)
        .appendSequencerBatch(DEFAULT_BATCH, oldTimestamp, blockNumber, 0)
    })

    it('should revert if submitting a batch with a future timestamp', async () => {
      const blockNumber = await provider.getBlockNumber()
      const timestamp = Math.floor(Date.now() / 1000)
      const futureTimestamp = timestamp + 30_000
      await TestUtils.assertRevertsAsync(async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, futureTimestamp, blockNumber, 0)
      }, 'Cannot submit a batch with a timestamp in the future')
    })

    it('should revert if submitting a batch with a future blockNumber', async () => {
      const timestamp = Math.floor(Date.now() / 1000)
      const blockNumber = Math.floor(timestamp / 15)
      const futureBlockNumber = blockNumber + 100
      await TestUtils.assertRevertsAsync(async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, timestamp, futureBlockNumber, 0)
      }, 'Cannot submit a batch with a blockNumber in the future')
    })

    it('should revert if submitting a new batch with a timestamp older than last batch timestamp', async () => {
      const [timestamp, blockNumber] = await appendSequencerBatch(
        DEFAULT_BATCH,
        0
      )

      const oldTimestamp = timestamp - 1
      await TestUtils.assertRevertsAsync(async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(
            DEFAULT_BATCH,
            oldTimestamp,
            blockNumber,
            DEFAULT_BATCH.length
          )
      }, 'Timestamps must monotonically increase')
    })

    it('should revert if submitting a new batch with a blockNumber older than last batch blockNumber', async () => {
      const [timestamp, blockNumber] = await appendSequencerBatch(
        DEFAULT_BATCH,
        0
      )

      const oldBlockNumber = blockNumber - 1
      await TestUtils.assertRevertsAsync(async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(
            DEFAULT_BATCH,
            timestamp,
            oldBlockNumber,
            DEFAULT_BATCH.length
          )
      }, 'BlockNumbers must monotonically increase')
    })
    it('should add to batches array', async () => {
      await appendSequencerBatch(DEFAULT_BATCH, 0)
      const batchesLength = await canonicalTxChain.getBatchesLength()
      batchesLength.toNumber().should.equal(1)
    })

    it('should update cumulativeNumElements correctly', async () => {
      await appendSequencerBatch(DEFAULT_BATCH, 0)
      const cumulativeNumElements = await canonicalTxChain.cumulativeNumElements.call()
      cumulativeNumElements.toNumber().should.equal(DEFAULT_BATCH.length)
    })

    it('should not allow appendSequencerBatch from non-sequencer', async () => {
      const timestamp = Math.floor(Date.now() / 1000)
      const blockNumber = Math.floor(timestamp / 15)

      await TestUtils.assertRevertsAsync(async () => {
        await canonicalTxChain.appendSequencerBatch(
          DEFAULT_BATCH,
          timestamp,
          blockNumber,
          0
        )
      }, 'Message sender does not have permission to append a batch')
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
        localBatch = await enqueueAndGenerateL1ToL2Batch(
          DEFAULT_L1_L2_MESSAGE_PARAMS
        )
      })

      it('should successfully append a batch with an older timestamp and blockNumber', async () => {
        const oldTimestamp = localBatch.timestamp - 1
        const oldBlockNumber = localBatch.blockNumber - 1
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, oldTimestamp, oldBlockNumber, 0)
      })

      it('should successfully append a batch with an equal timestamp', async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(
            DEFAULT_BATCH,
            localBatch.timestamp,
            localBatch.blockNumber,
            0
          )
      })

      it('should revert when there is an older batch in the L1ToL2Queue', async () => {
        const snapshotID = await provider.send('evm_snapshot', [])
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD])
        const newTimestamp = localBatch.timestamp + 60
        await TestUtils.assertRevertsAsync(async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(
              DEFAULT_BATCH,
              newTimestamp,
              localBatch.blockNumber,
              0
            )
        }, 'Must process older L1ToL2Queue batches first to enforce OVM timestamp monotonicity')
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
          .appendSequencerBatch(
            DEFAULT_BATCH,
            oldTimestamp,
            localBatch.blockNumber,
            0
          )
      })

      it('should successfully append a batch with an equal timestamp', async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(
            DEFAULT_BATCH,
            localBatch.timestamp,
            localBatch.blockNumber,
            0
          )
      })

      it('should revert when there is an older-timestamp batch in the SafetyQueue', async () => {
        const snapshotID = await provider.send('evm_snapshot', [])
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD])
        const newTimestamp = localBatch.timestamp + 60
        await TestUtils.assertRevertsAsync(async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(
              DEFAULT_BATCH,
              newTimestamp,
              localBatch.blockNumber,
              0
            )
        }, 'Must process older SafetyQueue batches first to enforce OVM timestamp monotonicity')
        await provider.send('evm_revert', [snapshotID])
      })

      it('should revert when there is an older-blockNumber batch in the SafetyQueue', async () => {
        await provider.send(`evm_mine`, [])
        await TestUtils.assertRevertsAsync(async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(
              DEFAULT_BATCH,
              localBatch.timestamp,
              localBatch.blockNumber + 1,
              0
            )
        }, 'Must process older SafetyQueue batches first to enforce OVM blockNumber monotonicity')
      })
    })

    describe('when there is an old batch in the safetyQueue and a recent batch in the l1ToL2Queue', async () => {
      let safetyTimestamp
      let safetyBlockNumber
      let l1ToL2Timestamp
      let l1ToL2BlockNumber
      let snapshotID
      beforeEach(async () => {
        const localSafetyBatch = await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
        safetyTimestamp = localSafetyBatch.timestamp
        safetyBlockNumber = localSafetyBatch.blockNumber
        snapshotID = await provider.send('evm_snapshot', [])
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD / 2])
        const localL1ToL2Batch = await enqueueAndGenerateL1ToL2Batch(
          DEFAULT_L1_L2_MESSAGE_PARAMS
        )
        l1ToL2Timestamp = localL1ToL2Batch.timestamp
        l1ToL2BlockNumber = localL1ToL2Batch.blockNumber
      })
      afterEach(async () => {
        await provider.send('evm_revert', [snapshotID])
      })

      it('should successfully append a batch with an older timestamp than the oldest batch', async () => {
        const oldTimestamp = safetyTimestamp - 1
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(
            DEFAULT_BATCH,
            oldTimestamp,
            safetyBlockNumber,
            0
          )
      })

      it('should successfully append a batch with an older blockNumber than the oldest batch', async () => {
        const oldBlockNumber = safetyBlockNumber - 1
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(
            DEFAULT_BATCH,
            safetyTimestamp,
            oldBlockNumber,
            0
          )
      })

      it('should successfully append a batch with a timestamp and blockNumber equal to the oldest batch', async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(
            DEFAULT_BATCH,
            safetyTimestamp,
            safetyBlockNumber,
            0
          )
      })

      it('should revert when appending a batch with a timestamp in between the two batches', async () => {
        const middleTimestamp = safetyTimestamp + 1
        await TestUtils.assertRevertsAsync(async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(
              DEFAULT_BATCH,
              middleTimestamp,
              safetyBlockNumber,
              0
            )
        }, 'Must process older SafetyQueue batches first to enforce OVM timestamp monotonicity')
      })

      it('should revert when appending a batch with a timestamp in between the two batches', async () => {
        const middleBlockNumber = safetyBlockNumber + 1
        await TestUtils.assertRevertsAsync(async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(
              DEFAULT_BATCH,
              safetyTimestamp,
              middleBlockNumber,
              0
            )
        }, 'Must process older SafetyQueue batches first to enforce OVM blockNumber monotonicity')
      })

      it('should revert when appending a batch with a timestamp newer than both batches', async () => {
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD / 10]) // increase time by 60 seconds
        const newTimestamp = l1ToL2Timestamp + 1
        await TestUtils.assertRevertsAsync(async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(
              DEFAULT_BATCH,
              newTimestamp,
              safetyBlockNumber,
              0
            )
        }, 'Must process older L1ToL2Queue batches first to enforce OVM timestamp monotonicity')
      })

      it('should revert when appending a batch with a blockNumber newer than both batches', async () => {
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD / 10]) // increase time by 60 seconds
        const newBlockNumber = l1ToL2BlockNumber + 1
        await TestUtils.assertRevertsAsync(async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(
              DEFAULT_BATCH,
              safetyTimestamp,
              newBlockNumber,
              0
            )
        }, 'Must process older L1ToL2Queue batches first to enforce OVM blockNumber monotonicity')
      })
    })

    describe('when there is an old batch in the l1ToL2Queue and a recent batch in the safetyQueue', async () => {
      let l1ToL2Timestamp
      let l1ToL2BlockNumber
      let safetyTimestamp
      let safetyBlockNumber
      let snapshotID
      beforeEach(async () => {
        const localL1ToL2Batch = await enqueueAndGenerateL1ToL2Batch(
          DEFAULT_L1_L2_MESSAGE_PARAMS
        )
        l1ToL2Timestamp = localL1ToL2Batch.timestamp
        l1ToL2BlockNumber = localL1ToL2Batch.blockNumber
        snapshotID = await provider.send('evm_snapshot', [])
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD / 2])
        const localSafetyBatch = await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
        safetyTimestamp = localSafetyBatch.timestamp
        safetyBlockNumber = localSafetyBatch.blockNumber
      })
      afterEach(async () => {
        await provider.send('evm_revert', [snapshotID])
      })

      it('should successfully append a batch with an older timestamp than both batches', async () => {
        const oldTimestamp = l1ToL2Timestamp - 1
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(
            DEFAULT_BATCH,
            oldTimestamp,
            l1ToL2BlockNumber,
            0
          )
      })

      it('should successfully append a batch with an older blockNumber than both batches', async () => {
        const oldBlockNumber = l1ToL2BlockNumber - 1
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(
            DEFAULT_BATCH,
            l1ToL2Timestamp,
            oldBlockNumber,
            0
          )
      })

      it('should successfully append a batch with a timestamp and blockNumber equal to the older batch', async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(
            DEFAULT_BATCH,
            l1ToL2Timestamp,
            l1ToL2BlockNumber,
            0
          )
      })

      it('should revert when appending a batch with a timestamp in between the two batches', async () => {
        const middleTimestamp = l1ToL2Timestamp + 1
        await TestUtils.assertRevertsAsync(async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(
              DEFAULT_BATCH,
              middleTimestamp,
              safetyBlockNumber,
              0
            )
        }, 'Must process older L1ToL2Queue batches first to enforce OVM timestamp monotonicity')
      })

      it('should revert when appending a batch with a blockNumber in between the two batches', async () => {
        const middleBlockNumber = l1ToL2BlockNumber + 1
        await TestUtils.assertRevertsAsync(async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(
              DEFAULT_BATCH,
              safetyTimestamp,
              middleBlockNumber,
              0
            )
        }, 'Must process older L1ToL2Queue batches first to enforce OVM timestamp monotonicity')
      })

      it('should revert when appending a batch with a timestamp newer than both batches', async () => {
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD / 10]) // increase time by 60 seconds
        const newTimestamp = safetyTimestamp + 1
        await TestUtils.assertRevertsAsync(async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(
              DEFAULT_BATCH,
              newTimestamp,
              safetyBlockNumber,
              0
            )
        }, 'Must process older L1ToL2Queue batches first to enforce OVM timestamp monotonicity')
      })

      it('should revert when appending a batch with a blockNumber newer than both batches', async () => {
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD / 10]) // increase time by 60 seconds
        const newBlockNumber = safetyBlockNumber + 1
        await TestUtils.assertRevertsAsync(async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(
              DEFAULT_BATCH,
              l1ToL2Timestamp,
              newBlockNumber,
              0
            )
        }, 'Must process older L1ToL2Queue batches first to enforce OVM blockNumber monotonicity')
      })
    })

    describe('when the txs in the batch are not the next index', () => {
      it('reverts if starts at index is less than canonical chain length', async () => {
        const blockNumber = 1
        const timestamp = Math.round(new Date().getTime() / 1000) - 5
        const startsAtIndex = 0
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(
            DEFAULT_BATCH,
            timestamp,
            blockNumber,
            startsAtIndex
          )

        // Should fail because the index needs to be 1
        await TestUtils.assertRevertsAsync(async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(
              DEFAULT_BATCH,
              timestamp,
              blockNumber,
              startsAtIndex
            )
        }, 'Cannot submit a batch with a startsAtTxIndex less than cumulativeNumElements')
      })

      it('reverts if starts at index is greater than canonical chain length and there are no queued batches', async () => {
        const blockNumber = 1
        const timestamp = Math.round(new Date().getTime() / 1000) - 5
        const startsAtIndex = 1
        // Should fail because the index needs to be 1
        await TestUtils.assertRevertsAsync(async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(
              DEFAULT_BATCH,
              timestamp,
              blockNumber,
              startsAtIndex
            )
        }, 'Cannot append from queues up to index because queues are empty')
      })

      it('successfully handles a future start index when there is a batch in the Safety Queue', async () => {
        const queuedBatch = await enqueueAndGenerateSafetyBatch(DEFAULT_TX)

        const blockNumber = queuedBatch.blockNumber + 1
        const timestamp = queuedBatch.timestamp + 1
        const startsAtIndex = 1
        // Should fail because the index needs to be 1
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(
            DEFAULT_BATCH,
            timestamp,
            blockNumber,
            startsAtIndex
          )

        const numElementsAfter = await canonicalTxChain.cumulativeNumElements()
        numElementsAfter
          .toNumber()
          .should.equal(
            DEFAULT_BATCH.length + queuedBatch.elements.length,
            `Incorrect number of txs added to the canonical chain!`
          )

        const safetyQueueEmpty = await safetyQueue.isEmpty()
        safetyQueueEmpty.should.equal(
          true,
          `Safety queue should be empty because its tx was appended!`
        )

        const firstBatch = await canonicalTxChain.batches(0)
        firstBatch.should.equal(
          await queuedBatch.hashBatchHeader(false),
          `Incorrect batch ordering!`
        )

        const defaultBatch = new TxChainBatch(
          timestamp,
          blockNumber,
          false,
          1,
          1,
          DEFAULT_BATCH
        )
        await defaultBatch.generateTree()
        const secondBatch = await canonicalTxChain.batches(1)
        secondBatch.should.equal(
          await defaultBatch.hashBatchHeader(),
          `Incorrect batch ordering!`
        )
      })

      it('successfully handles a future start index when there is a batch in the L1 To L2 Queue', async () => {
        const queuedBatch = await enqueueAndGenerateL1ToL2Batch(
          DEFAULT_L1_L2_MESSAGE_PARAMS
        )

        const blockNumber = queuedBatch.blockNumber + 1
        const timestamp = queuedBatch.timestamp + 1
        const startsAtIndex = 1
        // Should fail because the index needs to be 1
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(
            DEFAULT_BATCH,
            timestamp,
            blockNumber,
            startsAtIndex
          )

        const numElementsAfter = await canonicalTxChain.cumulativeNumElements()
        numElementsAfter
          .toNumber()
          .should.equal(
            DEFAULT_BATCH.length + queuedBatch.elements.length,
            `Incorrect number of txs added to the canonical chain!`
          )

        const l1ToL2QueueIsEmpty = await safetyQueue.isEmpty()
        l1ToL2QueueIsEmpty.should.equal(
          true,
          `L1ToL2 queue should be empty because its tx was appended!`
        )

        const firstBatch = await canonicalTxChain.batches(0)
        firstBatch.should.equal(
          await queuedBatch.hashBatchHeader(true),
          `Incorrect batch ordering!`
        )

        const defaultBatch = new TxChainBatch(
          timestamp,
          blockNumber,
          false,
          1,
          1,
          DEFAULT_BATCH
        )
        await defaultBatch.generateTree()
        const secondBatch = await canonicalTxChain.batches(1)
        secondBatch.should.equal(
          await defaultBatch.hashBatchHeader(),
          `Incorrect batch ordering!`
        )
      })

      it('successfully handles pulling in a batch from L1 To L2 and Safety Queue', async () => {
        const safetyQueueBatch = await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
        const l1ToL2QueueBatch = await enqueueAndGenerateL1ToL2Batch(
          DEFAULT_L1_L2_MESSAGE_PARAMS
        )

        const blockNumber = l1ToL2QueueBatch.blockNumber + 1
        const timestamp = l1ToL2QueueBatch.timestamp + 1
        const startsAtIndex = 2
        // Should fail because the index needs to be 1
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(
            DEFAULT_BATCH,
            timestamp,
            blockNumber,
            startsAtIndex
          )

        const numElementsAfter = await canonicalTxChain.cumulativeNumElements()
        const expectedElements =
          DEFAULT_BATCH.length +
          safetyQueueBatch.elements.length +
          l1ToL2QueueBatch.elements.length
        numElementsAfter
          .toNumber()
          .should.equal(
            expectedElements,
            `Incorrect number of txs added to the canonical chain!`
          )

        const safetyQueueEmpty = await safetyQueue.isEmpty()
        safetyQueueEmpty.should.equal(
          true,
          `Safety queue should be empty because its tx was appended!`
        )

        const l1ToL2QueueIsEmpty = await l1ToL2Queue.isEmpty()
        l1ToL2QueueIsEmpty.should.equal(
          true,
          `L1ToL2 queue should be empty because its tx was appended!`
        )

        const safetyQueueBatchHeader = await safetyQueueBatch.hashBatchHeader(
          false
        )
        const l1ToL2QueueBatchHeader = await l1ToL2QueueBatch.hashBatchHeader(
          true,
          1
        )
        const defaultBatch = new TxChainBatch(
          timestamp,
          blockNumber,
          false,
          2,
          2,
          DEFAULT_BATCH
        )
        await defaultBatch.generateTree()
        const defaultBatchHeader = await defaultBatch.hashBatchHeader()

        const firstBatch = await canonicalTxChain.batches(0)
        firstBatch.should.equal(
          safetyQueueBatchHeader,
          `Incorrect batch ordering on batch 0!`
        )
        const secondBatch = await canonicalTxChain.batches(1)
        secondBatch.should.equal(
          l1ToL2QueueBatchHeader,
          `Incorrect batch ordering on batch 1!`
        )

        const thirdBatch = await canonicalTxChain.batches(2)
        thirdBatch.should.equal(
          defaultBatchHeader,
          `Incorrect batch ordering on batch 2!`
        )
      })

      it('successfully handles pulling in a single batch from L1 To L2, when the Safety Queue also has a batch', async () => {
        const l1ToL2QueueBatch = await enqueueAndGenerateL1ToL2Batch(
          DEFAULT_L1_L2_MESSAGE_PARAMS
        )
        await enqueueAndGenerateSafetyBatch(DEFAULT_TX)

        const blockNumber = l1ToL2QueueBatch.blockNumber + 1
        const timestamp = l1ToL2QueueBatch.timestamp + 1
        const startsAtIndex = 1
        // Should fail because the index needs to be 1
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(
            DEFAULT_BATCH,
            timestamp,
            blockNumber,
            startsAtIndex
          )

        const numElementsAfter = await canonicalTxChain.cumulativeNumElements()
        const expectedElements =
          DEFAULT_BATCH.length + l1ToL2QueueBatch.elements.length
        numElementsAfter
          .toNumber()
          .should.equal(
            expectedElements,
            `Incorrect number of txs added to the canonical chain!`
          )

        const safetyQueueEmpty = await safetyQueue.isEmpty()
        safetyQueueEmpty.should.equal(
          false,
          `Safety queue should not be empty!`
        )

        const l1ToL2QueueIsEmpty = await l1ToL2Queue.isEmpty()
        l1ToL2QueueIsEmpty.should.equal(
          true,
          `L1ToL2 queue should be empty because its tx was appended!`
        )

        const l1ToL2QueueBatchHeader = await l1ToL2QueueBatch.hashBatchHeader(
          true,
          0
        )
        const defaultBatch = new TxChainBatch(
          timestamp,
          blockNumber,
          false,
          1,
          1,
          DEFAULT_BATCH
        )
        await defaultBatch.generateTree()
        const defaultBatchHeader = await defaultBatch.hashBatchHeader()

        const firstBatch = await canonicalTxChain.batches(0)
        firstBatch.should.equal(
          l1ToL2QueueBatchHeader,
          `Incorrect batch ordering on batch 0!`
        )

        const secondBatch = await canonicalTxChain.batches(1)
        secondBatch.should.equal(
          defaultBatchHeader,
          `Incorrect batch ordering on batch 2!`
        )
      })

      it('successfully handles pulling in a single batch from Safety Queue, when the L1ToL2 Queue also has a batch', async () => {
        const safetyQueueBatch = await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
        await enqueueAndGenerateL1ToL2Batch(DEFAULT_L1_L2_MESSAGE_PARAMS)

        const blockNumber = safetyQueueBatch.blockNumber + 1
        const timestamp = safetyQueueBatch.timestamp + 1
        const startsAtIndex = 1
        // Should fail because the index needs to be 1
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(
            DEFAULT_BATCH,
            timestamp,
            blockNumber,
            startsAtIndex
          )

        const numElementsAfter = await canonicalTxChain.cumulativeNumElements()
        const expectedElements =
          DEFAULT_BATCH.length + safetyQueueBatch.elements.length
        numElementsAfter
          .toNumber()
          .should.equal(
            expectedElements,
            `Incorrect number of txs added to the canonical chain!`
          )

        const safetyQueueEmpty = await safetyQueue.isEmpty()
        safetyQueueEmpty.should.equal(
          true,
          `Safety queue should be empty because its tx was appended!`
        )

        const l1ToL2QueueIsEmpty = await l1ToL2Queue.isEmpty()
        l1ToL2QueueIsEmpty.should.equal(
          false,
          `L1ToL2 queue should not be empty!`
        )

        const safetyQueueBatchHeader = await safetyQueueBatch.hashBatchHeader(
          false
        )
        const defaultBatch = new TxChainBatch(
          timestamp,
          blockNumber,
          false,
          1,
          1,
          DEFAULT_BATCH
        )
        await defaultBatch.generateTree()
        const defaultBatchHeader = await defaultBatch.hashBatchHeader()

        const firstBatch = await canonicalTxChain.batches(0)
        firstBatch.should.equal(
          safetyQueueBatchHeader,
          `Incorrect batch ordering on batch 0!`
        )

        const secondBatch = await canonicalTxChain.batches(1)
        secondBatch.should.equal(
          defaultBatchHeader,
          `Incorrect batch ordering on batch 2!`
        )
      })

      it('successfully appends a sequencer batch with batches in the Safety Queue, L1ToL2 Queue', async () => {
        const safetyQueueBatch = await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
        await enqueueAndGenerateL1ToL2Batch(DEFAULT_L1_L2_MESSAGE_PARAMS)

        const blockNumber = safetyQueueBatch.blockNumber - 1
        const timestamp = safetyQueueBatch.timestamp - 1
        const startsAtIndex = 0
        // Should fail because the index needs to be 1
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(
            DEFAULT_BATCH,
            timestamp,
            blockNumber,
            startsAtIndex
          )

        const numElementsAfter = await canonicalTxChain.cumulativeNumElements()
        const expectedElements = DEFAULT_BATCH.length
        numElementsAfter
          .toNumber()
          .should.equal(
            expectedElements,
            `Incorrect number of txs added to the canonical chain!`
          )

        const safetyQueueEmpty = await safetyQueue.isEmpty()
        safetyQueueEmpty.should.equal(
          false,
          `Safety queue should not be empty!`
        )

        const l1ToL2QueueIsEmpty = await l1ToL2Queue.isEmpty()
        l1ToL2QueueIsEmpty.should.equal(
          false,
          `L1ToL2 queue should not be empty!`
        )

        const defaultBatch = new TxChainBatch(
          timestamp,
          blockNumber,
          false,
          0,
          0,
          DEFAULT_BATCH
        )
        await defaultBatch.generateTree()
        const defaultBatchHeader = await defaultBatch.hashBatchHeader()

        const firstBatchHeader = await canonicalTxChain.batches(0)
        firstBatchHeader.should.equal(
          defaultBatchHeader,
          `Incorrect batch ordering on batch 2!`
        )
      })
    })
  })

  describe('appendL1ToL2Batch()', async () => {
    describe('when there is a batch in the L1toL2Queue', async () => {
      beforeEach(async () => {
        await enqueueAndGenerateL1ToL2Batch(DEFAULT_L1_L2_MESSAGE_PARAMS)
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
        const {
          timestamp,
          txHash,
          blockNumber,
        } = await l1ToL2Queue.batchHeaders(0)
        const localBatch = new TxChainBatch(
          timestamp,
          blockNumber,
          true, // isL1ToL2Tx
          0, //batchIndex
          0, // cumulativePrevElements
          [DEFAULT_L1_L2_MESSAGE_PARAMS], // elements
          await l1ToL2TransactionPasser.getAddress()
        )
        await localBatch.generateTree()
        const localBatchHeaderHash = await localBatch.hashBatchHeader()
        await canonicalTxChain.connect(sequencer).appendL1ToL2Batch()
        const batchHeaderHash = await canonicalTxChain.batches(0)
        batchHeaderHash.should.equal(localBatchHeaderHash)
      })

      it('should not allow non-sequencer to appendL1ToL2Batch if less than the inclusion period', async () => {
        await TestUtils.assertRevertsAsync(async () => {
          await canonicalTxChain.appendL1ToL2Batch()
        }, 'Message sender does not have permission to append this batch')
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
        await enqueueAndGenerateL1ToL2Batch(DEFAULT_L1_L2_MESSAGE_PARAMS)
        await TestUtils.assertRevertsAsync(async () => {
          await canonicalTxChain.appendL1ToL2Batch()
        }, 'Must process older SafetyQueue batches first to enforce OVM timestamp monotonicity')
        await provider.send('evm_revert', [snapshotID])
      })

      it('should succeed when the L1ToL2Queue batch is older', async () => {
        const snapshotID = await provider.send('evm_snapshot', [])
        await enqueueAndGenerateL1ToL2Batch(DEFAULT_L1_L2_MESSAGE_PARAMS)
        await provider.send('evm_increaseTime', [10])
        await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
        await canonicalTxChain.connect(sequencer).appendL1ToL2Batch()
        await provider.send('evm_revert', [snapshotID])
      })
    })

    it('should revert when L1ToL2TxQueue is empty', async () => {
      await TestUtils.assertRevertsAsync(async () => {
        await canonicalTxChain.appendL1ToL2Batch()
      }, 'Queue is empty, no element to peek at')
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
        const {
          timestamp,
          txHash,
          blockNumber,
        } = await safetyQueue.batchHeaders(0)
        const localBatch = new TxChainBatch(
          timestamp,
          blockNumber,
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
        await TestUtils.assertRevertsAsync(async () => {
          await canonicalTxChain.appendSafetyBatch()
        }, 'Message sender does not have permission to append this batch')
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
      await enqueueAndGenerateL1ToL2Batch(DEFAULT_L1_L2_MESSAGE_PARAMS)
      await provider.send('evm_increaseTime', [10])
      await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
      await TestUtils.assertRevertsAsync(async () => {
        await canonicalTxChain.appendSafetyBatch()
      }, 'Must process older L1ToL2Queue batches first to enforce OVM timestamp monotonicity')
      await provider.send('evm_revert', [snapshotID])
    })

    it('should succeed when there are only newer batches in the L1ToL2Queue ', async () => {
      const snapshotID = await provider.send('evm_snapshot', [])
      await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
      await provider.send('evm_increaseTime', [10])
      await enqueueAndGenerateL1ToL2Batch(DEFAULT_L1_L2_MESSAGE_PARAMS)
      await canonicalTxChain.connect(sequencer).appendSafetyBatch()
      await provider.send('evm_revert', [snapshotID])
    })

    it('should revert when SafetyTxQueue is empty', async () => {
      await TestUtils.assertRevertsAsync(async () => {
        await canonicalTxChain.appendSafetyBatch()
      }, 'Queue is empty, no element to peek at')
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
      const senderAddress = await l1ToL2TransactionPasser.getAddress()
      const l1ToL2Batch = await enqueueAndGenerateL1ToL2Batch(
        DEFAULT_L1_L2_MESSAGE_PARAMS
      )
      await canonicalTxChain.connect(sequencer).appendL1ToL2Batch()
      const localBatch = new TxChainBatch(
        l1ToL2Batch.timestamp, //timestamp
        l1ToL2Batch.blockNumber,
        true, //isL1ToL2Tx
        0, //batchIndex
        0, //cumulativePrevElements
        [DEFAULT_L1_L2_MESSAGE_PARAMS], //batch
        senderAddress
      )
      await localBatch.generateTree()
      const elementIndex = 0
      const position = localBatch.getPosition(elementIndex)
      const elementInclusionProof = await localBatch.getElementInclusionProof(
        elementIndex
      )
      const isIncluded = await canonicalTxChain.verifyElement(
        getL1ToL2MessageTxData(
          senderAddress,
          DEFAULT_L1_L2_MESSAGE_PARAMS[0],
          DEFAULT_L1_L2_MESSAGE_PARAMS[1],
          DEFAULT_L1_L2_MESSAGE_PARAMS[2]
        ), // element
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
        safetyBatch.blockNumber,
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
      let enqueuedTx: any[]
      l1ToL2Queue.on(
        l1ToL2Queue.filters['L1ToL2TxEnqueued(address,address,uint32,bytes)'](),
        (...data) => {
          enqueuedTx = [data[0], data[1], data[2], data[3]]
        }
      )

      const localBatch: TxQueueBatch = await enqueueAndGenerateL1ToL2Batch(
        DEFAULT_L1_L2_MESSAGE_PARAMS
      )

      await sleep(5_000)

      const receivedTx: boolean = !!enqueuedTx
      receivedTx.should.equal(true, `Did not receive expected event!`)

      const encodedEnqueuedTx = abi.encode(
        ['address', 'address', 'uint32', 'bytes'],
        enqueuedTx
      )

      encodedEnqueuedTx.should.equal(
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
        DEFAULT_L1_L2_MESSAGE_PARAMS
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
