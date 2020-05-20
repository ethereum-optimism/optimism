import '../setup'

/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { Contract } from 'ethers'

/* Internal Imports */
import { DefaultRollupBatch, RollupQueueBatch } from './RLhelper'

/* Logging */
const log = getLogger('rollup-tx-queue', true)

/* Contract Imports */
import * as CanonicalTransactionChain from '../../build/CanonicalTransactionChain.json'
import * as L1ToL2TransactionQueue from '../../build/L1ToL2TransactionQueue.json'
import * as RollupMerkleUtils from '../../build/RollupMerkleUtils.json'

/* Begin tests */
describe('CanonicalTransactionChain', () => {
  const provider = createMockProvider()
  const [
    wallet,
    sequencer,
    canonicalTransactionChain,
    l1ToL2TransactionPasser,
  ] = getWallets(provider)
  let canonicalTxChain
  let rollupMerkleUtils

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
      ],
      {
        gasLimit: 6700000,
      }
    )
  })

  const enqueueAndGenerateBatch = async (
    batch: string[],
    timestamp: number,
    batchIndex: number,
    cumulativePrevElements: number
  ): Promise<DefaultRollupBatch> => {
    // Submit the rollup batch on-chain
    await canonicalTxChain
      .connect(sequencer)
      .appendTransactionBatch(batch, timestamp)
    // Generate a local version of the rollup batch
    const localBatch = new DefaultRollupBatch(
      timestamp,
      false,
      batchIndex,
      cumulativePrevElements,
      batch
    )
    await localBatch.generateTree()
    return localBatch
  }

  /*
   * Test appendTransactionBatch()
   */
  describe('appendTransactionBatch()', async () => {
    it('should not throw as long as it gets a bytes array (even if its invalid)', async () => {
      const batch = ['0x1234', '0x1234']
      const timestamp = 0
      await canonicalTxChain
        .connect(sequencer)
        .appendTransactionBatch(batch, timestamp) // Did not throw... success!
    })

    it('should throw if submitting an empty batch', async () => {
      const emptyBatch = []
      const timestamp = 0
      await canonicalTxChain
        .connect(sequencer)
        .appendTransactionBatch(emptyBatch, timestamp)
        .should.be.revertedWith(
          'VM Exception while processing transaction: revert Cannot submit an empty batch'
        )
    })

    it('should add to batches array', async () => {
      const batch = ['0x1234', '0x6578']
      const timestamp = 0
      const output = await canonicalTxChain
        .connect(sequencer)
        .appendTransactionBatch(batch, timestamp)
      const batchesLength = await canonicalTxChain.getBatchsLength()
      batchesLength.toNumber().should.equal(1)
    })

    it('should update cumulativeNumElements correctly', async () => {
      const batch = ['0x1234', '0x5678']
      const timestamp = 0
      await canonicalTxChain
        .connect(sequencer)
        .appendTransactionBatch(batch, timestamp)
      const cumulativeNumElements = await canonicalTxChain.cumulativeNumElements.call()
      cumulativeNumElements.toNumber().should.equal(2)
    })
    it('should allow appendTransactionBatch from sequencer', async () => {
      const batch = ['0x1234', '0x6578']
      const timestamp = 0
      await canonicalTxChain
        .connect(sequencer)
        .appendTransactionBatch(batch, timestamp) // Did not throw... success!
    })
    it('should not allow appendTransactionBatch from other address', async () => {
      const batch = ['0x1234', '0x6578']
      const timestamp = 0
      await canonicalTxChain
        .appendTransactionBatch(batch, timestamp)
        .should.be.revertedWith(
          'VM Exception while processing transaction: revert Message sender does not have permission to append a batch'
        )
    })
    it('should calculate batchHeaderHash correctly', async () => {
      const batch = ['0x1234', '0x5678']
      const batchIndex = 0
      const cumulativePrevElements = 0
      const timestamp = 0
      const localBatch = await enqueueAndGenerateBatch(
        batch,
        timestamp,
        batchIndex,
        cumulativePrevElements
      )
      //Check batchHeaderHash
      const expectedBatchHeaderHash = await localBatch.hashBatchHeader()
      const calculatedBatchHeaderHash = await canonicalTxChain.batches(0)
      calculatedBatchHeaderHash.should.equal(expectedBatchHeaderHash)
    })
    it('should add multiple batches correctly', async () => {
      const batch = ['0x1234', '0x5678']
      const numBatchs = 10
      for (let batchIndex = 0; batchIndex < numBatchs; batchIndex++) {
        const timestamp = batchIndex
        const cumulativePrevElements = batch.length * batchIndex
        const localBatch = await enqueueAndGenerateBatch(
          batch,
          timestamp,
          batchIndex,
          cumulativePrevElements
        )
        //Check batchHeaderHash
        const expectedBatchHeaderHash = await localBatch.hashBatchHeader()
        const calculatedBatchHeaderHash = await canonicalTxChain.batches(
          batchIndex
        )
        calculatedBatchHeaderHash.should.equal(expectedBatchHeaderHash)
      }
      //check cumulativeNumElements
      const cumulativeNumElements = await canonicalTxChain.cumulativeNumElements.call()
      cumulativeNumElements.toNumber().should.equal(numBatchs * batch.length)
      //check batches length
      const batchesLength = await canonicalTxChain.getBatchsLength()
      batchesLength.toNumber().should.equal(numBatchs)
    })
  })

  describe('appendL1ToL2Batch()', async () => {
    let l1ToL2Queue
    const localL1ToL2Queue = []
    const enqueueAndGenerateQueueBatch = async (
      batch: string[]
    ): Promise<RollupQueueBatch> => {
      // Submit the rollup batch on-chain
      await l1ToL2Queue.connect(l1ToL2TransactionPasser).enqueueBatch(batch)
      // Generate a local version of the rollup batch
      const localBatch = new RollupQueueBatch(batch)
      await localBatch.generateTree()
      return localBatch
    }
    beforeEach(async () => {
      const batch = ['0x1234', '0x1234']
      const l1ToL2QueueAddress = await canonicalTxChain.l1ToL2Queue()
      l1ToL2Queue = new Contract(
        l1ToL2QueueAddress,
        L1ToL2TransactionQueue.abi,
        provider
      )
      const localBatch = await enqueueAndGenerateQueueBatch(batch)
      localL1ToL2Queue.push(localBatch)
    })
    it.only('should revert when passed an incorrect batch header', async () => {
      const localBatchHeader = await localL1ToL2Queue[0].getBatchHeader()
      localBatchHeader.numElementsInBatch++
      await canonicalTxChain
        .connect(sequencer)
        .appendL1ToL2Batch(localBatchHeader)
        .should.be.revertedWith(
          'VM Exception while processing transaction: revert This batch header is different than the batch header at the front of the L1ToL2TransactionQueue'
        )
    })
    it('should successfully dequeue a L1ToL2Batch', async () => {
      const localBatchHeader = await localL1ToL2Queue[0].getBatchHeader()
      console.log('local', localBatchHeader)
      await canonicalTxChain
        .connect(sequencer)
        .appendL1ToL2Batch(localBatchHeader)
      const front = await l1ToL2Queue.front()
      front.should.equal(1)
      const { timestamp, batchHeaderHash } = await l1ToL2Queue.batches(0)
      timestamp.should.equal(0)
      batchHeaderHash.should.equal(
        '0x0000000000000000000000000000000000000000000000000000000000000000'
      )
    })
  })

  describe('verifyElement() ', async () => {
    it('should return true for valid elements for different batchIndexes', async () => {
      const maxBatchNumber = 5
      const minBatchNumber = 0
      const batch = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
      for (
        let batchIndex = minBatchNumber;
        batchIndex < maxBatchNumber + 1;
        batchIndex++
      ) {
        const timestamp = batchIndex
        const cumulativePrevElements = batch.length * batchIndex
        const localBatch = await enqueueAndGenerateBatch(
          batch,
          timestamp,
          batchIndex,
          cumulativePrevElements
        )
        // Create inclusion proof for the element at elementIndex
        const elementIndex = 3
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
    })

    it('should return false for wrong position with wrong indexInBatch', async () => {
      const batch = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
      const cumulativePrevElements = 0
      const batchIndex = 0
      const timestamp = 0
      const localBatch = await enqueueAndGenerateBatch(
        batch,
        timestamp,
        batchIndex,
        cumulativePrevElements
      )
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
      const batch = ['0x1234', '0x4567', '0x890a', '0xabcd']
      const cumulativePrevElements = 0
      const batchIndex = 0
      const timestamp = 0
      const localBatch = await enqueueAndGenerateBatch(
        batch,
        timestamp,
        batchIndex,
        cumulativePrevElements
      )
      //generate inclusion proof
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
