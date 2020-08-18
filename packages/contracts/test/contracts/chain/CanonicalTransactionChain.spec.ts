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
describe.only('CanonicalTransactionChain', () => {
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

  const appendSequencerBatch = async (batch: string[]): Promise<number[]> => {
    const blocknumber = await provider.getBlockNumber()
    const timestamp = Math.floor(Date.now() / 1000)
    // Submit the rollup batch on-chain
    await canonicalTxChain
      .connect(sequencer)
      .appendSequencerBatch(batch, timestamp, blocknumber)
    return [timestamp, blocknumber]
  }

  const appendAndGenerateSequencerBatch = async (
    batch: string[],
    batchIndex: number = 0,
    cumulativePrevElements: number = 0
  ): Promise<TxChainBatch> => {
    const [timestamp, blocknumber] = await appendSequencerBatch(batch)
    return createTxChainBatch(
      batch,
      timestamp,
      blocknumber,
      false,
      batchIndex,
      cumulativePrevElements
    )
  }

  const createTxChainBatch = async (
    batch: string[],
    timestamp: number,
    blocknumber,
    isL1ToL2Tx: boolean,
    batchIndex: number = 0,
    cumulativePrevElements: number = 0
  ): Promise<TxChainBatch> => {
    const localBatch = new TxChainBatch(
      timestamp,
      blocknumber,
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
    const localBatch = new TxQueueBatch(rolledupData, timestamp, txReceipt.blockNumber)
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

    it('should revert if submitting a batch with timestamp older than the inclusion period', async () => {
      const timestamp = Math.floor(Date.now() / 1000)
      const blocknumber = Math.floor(timestamp/15)
      const oldTimestamp = timestamp - (FORCE_INCLUSION_PERIOD + 1000)
      await TestUtils.assertRevertsAsync(
        'Cannot submit a batch with a timestamp older than the sequencer inclusion period',
        async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(DEFAULT_BATCH, oldTimestamp, blocknumber)
        }
      )
    })

    it('should revert if submitting a batch with blocknumber older than the inclusion period', async () => {
      const timestamp = Math.floor(Date.now() / 1000)
      const FORCE_INCLUSION_PERIOD_BLOCKS = await canonicalTxChain.forceInclusionPeriodBlocks()
      for (let i = 0; i < FORCE_INCLUSION_PERIOD_BLOCKS + 1; i++) {
        await provider.send('evm_mine', [])
      }
      const currentBlockNumber = await canonicalTxChain.provider.getBlockNumber()
      await TestUtils.assertRevertsAsync(
        'Cannot submit a batch with a blocknumber older than the sequencer inclusion period',
        async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(DEFAULT_BATCH, timestamp, currentBlockNumber - FORCE_INCLUSION_PERIOD_BLOCKS)
        }
      )
    })

    it('should not revert if submitting an INCLUSION_PERIOD/2 old batch', async () => {
      const blocknumber = await provider.getBlockNumber()
      const timestamp = (await provider.getBlock(blocknumber)).timestamp
      const oldTimestamp = timestamp - FORCE_INCLUSION_PERIOD / 2
      await canonicalTxChain
        .connect(sequencer)
        .appendSequencerBatch(DEFAULT_BATCH, oldTimestamp, blocknumber)
    })

    it('should revert if submitting a batch with a future timestamp', async () => {
      const blocknumber = await provider.getBlockNumber()
      const timestamp = Math.floor(Date.now() / 1000)
      const futureTimestamp = timestamp + 30_000
      await TestUtils.assertRevertsAsync(
        'Cannot submit a batch with a timestamp in the future',
        async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(DEFAULT_BATCH, futureTimestamp, blocknumber)
        }
      )
    })

    it('should revert if submitting a batch with a future blocknumber', async () => {
      const timestamp = Math.floor(Date.now() / 1000)
      const blocknumber = Math.floor(timestamp/15)
      const futureBlocknumber = blocknumber + 100
      await TestUtils.assertRevertsAsync(
        'Cannot submit a batch with a blocknumber in the future',
        async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(DEFAULT_BATCH, timestamp, futureBlocknumber)
        }
      )
    })

    it('should revert if submitting a new batch with a timestamp older than last batch timestamp', async () => {
      const [timestamp, blocknumber] = await appendSequencerBatch(DEFAULT_BATCH)

      const oldTimestamp = timestamp - 1
      await TestUtils.assertRevertsAsync(
        'Timestamps must monotonically increase',
        async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(DEFAULT_BATCH, oldTimestamp, blocknumber)
        }
      )
    })

    it('should revert if submitting a new batch with a blocknumber older than last batch blocknumber', async () => {
      const [timestamp, blocknumber] = await appendSequencerBatch(DEFAULT_BATCH)

      const oldBlockNumber = blocknumber - 1
      await TestUtils.assertRevertsAsync(
        'Blocknumbers must monotonically increase',
        async () => {
          await canonicalTxChain
            .connect(sequencer)
            .appendSequencerBatch(DEFAULT_BATCH, timestamp, oldBlockNumber)
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
      const blocknumber = Math.floor(timestamp/15)

      await TestUtils.assertRevertsAsync(
        'Message sender does not have permission to append a batch',
        async () => {
          await canonicalTxChain.appendSequencerBatch(DEFAULT_BATCH, timestamp, blocknumber)
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
        localBatch = await enqueueAndGenerateL1ToL2Batch(
          DEFAULT_L1_L2_MESSAGE_PARAMS
        )
      })

      it('should successfully append a batch with an older timestamp and blocknumber', async () => {
        const oldTimestamp = localBatch.timestamp - 1
        const oldBlocknumber = localBatch.blocknumber - 1
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, oldTimestamp, oldBlocknumber)
      })

      it('should successfully append a batch with an equal timestamp', async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, localBatch.timestamp, localBatch.blocknumber)
      })

      it('should revert when there is an older batch in the L1ToL2Queue', async () => {
        const snapshotID = await provider.send('evm_snapshot', [])
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD])
        const newTimestamp = localBatch.timestamp + 60
        await TestUtils.assertRevertsAsync(
          'Must process older L1ToL2Queue batches first to enforce OVM timestamp monotonicity',
          async () => {
            await canonicalTxChain
              .connect(sequencer)
              .appendSequencerBatch(DEFAULT_BATCH, newTimestamp, localBatch.blocknumber)
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
          .appendSequencerBatch(DEFAULT_BATCH, oldTimestamp, localBatch.blocknumber)
      })

      it('should successfully append a batch with an equal timestamp', async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, localBatch.timestamp, localBatch.blocknumber)
      })

      it('should revert when there is an older-timestamp batch in the SafetyQueue', async () => {
        const snapshotID = await provider.send('evm_snapshot', [])
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD])
        const newTimestamp = localBatch.timestamp + 60
        await TestUtils.assertRevertsAsync(
          'Must process older SafetyQueue batches first to enforce OVM timestamp monotonicity',
          async () => {
            await canonicalTxChain
              .connect(sequencer)
              .appendSequencerBatch(DEFAULT_BATCH, newTimestamp, localBatch.blocknumber)
          }
        )
        await provider.send('evm_revert', [snapshotID])
      })

      it('should revert when there is an older-blocknumber batch in the SafetyQueue', async () => {
        await provider.send(`evm_mine`, [])
        await TestUtils.assertRevertsAsync(
          'Must process older SafetyQueue batches first to enforce OVM blocknumber monotonicity',
          async () => {
            await canonicalTxChain
              .connect(sequencer)
              .appendSequencerBatch(DEFAULT_BATCH, localBatch.timestamp, localBatch.blocknumber + 1)
          }
        )
      })
    })
    describe('when there is an old batch in the safetyQueue and a recent batch in the l1ToL2Queue', async () => {
      let safetyTimestamp
      let safetyBlocknumber
      let l1ToL2Timestamp
      let l1ToL2Blocknumber
      let snapshotID
      beforeEach(async () => {
        const localSafetyBatch = await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
        safetyTimestamp = localSafetyBatch.timestamp
        safetyBlocknumber = localSafetyBatch.blocknumber
        snapshotID = await provider.send('evm_snapshot', [])
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD / 2])
        const localL1ToL2Batch = await enqueueAndGenerateL1ToL2Batch(
          DEFAULT_L1_L2_MESSAGE_PARAMS
        )
        l1ToL2Timestamp = localL1ToL2Batch.timestamp
        l1ToL2Blocknumber = localL1ToL2Batch.blocknumber
      })
      afterEach(async () => {
        await provider.send('evm_revert', [snapshotID])
      })

      it('should successfully append a batch with an older timestamp than the oldest batch', async () => {
        const oldTimestamp = safetyTimestamp - 1
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, oldTimestamp, safetyBlocknumber)
      })

      it('should successfully append a batch with an older blocknumber than the oldest batch', async () => {
        const oldBlockNumber = safetyBlocknumber - 1
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, safetyTimestamp, oldBlockNumber)
      })

      it('should successfully append a batch with a timestamp and blocknumber equal to the oldest batch', async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, safetyTimestamp, safetyBlocknumber)
      })

      it('should revert when appending a batch with a timestamp in between the two batches', async () => {
        const middleTimestamp = safetyTimestamp + 1
        await TestUtils.assertRevertsAsync(
          'Must process older SafetyQueue batches first to enforce OVM timestamp monotonicity',
          async () => {
            await canonicalTxChain
              .connect(sequencer)
              .appendSequencerBatch(DEFAULT_BATCH, middleTimestamp, safetyBlocknumber)
          }
        )
      })

      it('should revert when appending a batch with a timestamp in between the two batches', async () => {
        const middleBlocknumber = safetyBlocknumber + 1
        await TestUtils.assertRevertsAsync(
          'Must process older SafetyQueue batches first to enforce OVM blocknumber monotonicity',
          async () => {
            await canonicalTxChain
              .connect(sequencer)
              .appendSequencerBatch(DEFAULT_BATCH, safetyTimestamp, middleBlocknumber)
          }
        )
      })

      it('should revert when appending a batch with a timestamp newer than both batches', async () => {
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD / 10]) // increase time by 60 seconds
        const newTimestamp = l1ToL2Timestamp + 1
        await TestUtils.assertRevertsAsync(
          'Must process older L1ToL2Queue batches first to enforce OVM timestamp monotonicity',
          async () => {
            await canonicalTxChain
              .connect(sequencer)
              .appendSequencerBatch(DEFAULT_BATCH, newTimestamp, safetyBlocknumber)
          }
        )
      })

      it('should revert when appending a batch with a blocknumber newer than both batches', async () => {
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD / 10]) // increase time by 60 seconds
        const newBlocknumber = l1ToL2Blocknumber + 1
        await TestUtils.assertRevertsAsync(
          'Must process older L1ToL2Queue batches first to enforce OVM blocknumber monotonicity',
          async () => {
            await canonicalTxChain
              .connect(sequencer)
              .appendSequencerBatch(DEFAULT_BATCH, safetyTimestamp, newBlocknumber)
          }
        )
      })
    })

    describe('when there is an old batch in the l1ToL2Queue and a recent batch in the safetyQueue', async () => {
      let l1ToL2Timestamp
      let l1ToL2Blocknumber
      let safetyTimestamp
      let safetyBlocknumber
      let snapshotID
      beforeEach(async () => {
        const localL1ToL2Batch = await enqueueAndGenerateL1ToL2Batch(
          DEFAULT_L1_L2_MESSAGE_PARAMS
        )
        l1ToL2Timestamp = localL1ToL2Batch.timestamp
        l1ToL2Blocknumber = localL1ToL2Batch.blocknumber
        snapshotID = await provider.send('evm_snapshot', [])
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD / 2])
        const localSafetyBatch = await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
        safetyTimestamp = localSafetyBatch.timestamp
        safetyBlocknumber = localSafetyBatch.blocknumber
      })
      afterEach(async () => {
        await provider.send('evm_revert', [snapshotID])
      })

      it('should successfully append a batch with an older timestamp than both batches', async () => {
        const oldTimestamp = l1ToL2Timestamp - 1
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, oldTimestamp, l1ToL2Blocknumber)
      })

      it('should successfully append a batch with an older blocknumber than both batches', async () => {
        const oldBlocknumber = l1ToL2Blocknumber - 1
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, l1ToL2Timestamp, oldBlocknumber)
      })

      it('should successfully append a batch with a timestamp and blocknumber equal to the older batch', async () => {
        await canonicalTxChain
          .connect(sequencer)
          .appendSequencerBatch(DEFAULT_BATCH, l1ToL2Timestamp, l1ToL2Blocknumber)
      })

      it('should revert when appending a batch with a timestamp in between the two batches', async () => {
        const middleTimestamp = l1ToL2Timestamp + 1
        await TestUtils.assertRevertsAsync(
          'Must process older L1ToL2Queue batches first to enforce OVM timestamp monotonicity',
          async () => {
            await canonicalTxChain
              .connect(sequencer)
              .appendSequencerBatch(DEFAULT_BATCH, middleTimestamp, safetyBlocknumber)
          }
        )
      })

      it('should revert when appending a batch with a blocknumber in between the two batches', async () => {
        const middleBlocknumber = l1ToL2Blocknumber + 1
        await TestUtils.assertRevertsAsync(
          'Must process older L1ToL2Queue batches first to enforce OVM timestamp monotonicity',
          async () => {
            await canonicalTxChain
              .connect(sequencer)
              .appendSequencerBatch(DEFAULT_BATCH, safetyTimestamp, middleBlocknumber)
          }
        )
      })

      it('should revert when appending a batch with a timestamp newer than both batches', async () => {
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD / 10]) // increase time by 60 seconds
        const newTimestamp = safetyTimestamp + 1
        await TestUtils.assertRevertsAsync(
          'Must process older L1ToL2Queue batches first to enforce OVM timestamp monotonicity',
          async () => {
            await canonicalTxChain
              .connect(sequencer)
              .appendSequencerBatch(DEFAULT_BATCH, newTimestamp, safetyBlocknumber)
          }
        )
      })

      it('should revert when appending a batch with a blocknumber newer than both batches', async () => {
        await provider.send('evm_increaseTime', [FORCE_INCLUSION_PERIOD / 10]) // increase time by 60 seconds
        const newBlocknumber = safetyBlocknumber + 1
        await TestUtils.assertRevertsAsync(
          'Must process older L1ToL2Queue batches first to enforce OVM blocknumber monotonicity',
          async () => {
            await canonicalTxChain
              .connect(sequencer)
              .appendSequencerBatch(DEFAULT_BATCH, l1ToL2Timestamp, newBlocknumber)
          }
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
        const { timestamp, txHash, blocknumber } = await l1ToL2Queue.batchHeaders(0)
        const localBatch = new TxChainBatch(
          timestamp,
          blocknumber,
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
        await enqueueAndGenerateL1ToL2Batch(DEFAULT_L1_L2_MESSAGE_PARAMS)
        await TestUtils.assertRevertsAsync(
          'Must process older SafetyQueue batches first to enforce OVM timestamp monotonicity',
          async () => {
            await canonicalTxChain.appendL1ToL2Batch()
          }
        )
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
        const { timestamp, txHash, blocknumber } = await safetyQueue.batchHeaders(0)
        const localBatch = new TxChainBatch(
          timestamp,
          blocknumber,
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
      await enqueueAndGenerateL1ToL2Batch(DEFAULT_L1_L2_MESSAGE_PARAMS)
      await provider.send('evm_increaseTime', [10])
      await enqueueAndGenerateSafetyBatch(DEFAULT_TX)
      await TestUtils.assertRevertsAsync(
        'Must process older L1ToL2Queue batches first to enforce OVM timestamp monotonicity',
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
      await enqueueAndGenerateL1ToL2Batch(DEFAULT_L1_L2_MESSAGE_PARAMS)
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
      const senderAddress = await l1ToL2TransactionPasser.getAddress()
      const l1ToL2Batch = await enqueueAndGenerateL1ToL2Batch(
        DEFAULT_L1_L2_MESSAGE_PARAMS
      )
      await canonicalTxChain.connect(sequencer).appendL1ToL2Batch()
      const localBatch = new TxChainBatch(
        l1ToL2Batch.timestamp, //timestamp
        l1ToL2Batch.blocknumber,
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
        safetyBatch.blocknumber,
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
