import '../setup'

/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Internal Imports */
import { DefaultRollupBatch } from './RLhelper'

/* Logging */
const log = getLogger('rollup-tx-queue', true)

/* Contract Imports */
import * as CanonicalTransactionChain from '../../build/CanonicalTransactionChain.json'
import * as RollupMerkleUtils from '../../build/RollupMerkleUtils.json'

/* Begin tests */
describe('CanonicalTransactionChain', () => {
  const provider = createMockProvider()
  const [wallet, sequencer, canonicalTransactionChain] = getWallets(provider)
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
      [rollupMerkleUtils.address, sequencer.address],
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
   * Test enqueueBatch()
   */
  describe('appendTransactionBatch() ', async () => {
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
      log.debug('enqueue batch output', JSON.stringify(output))
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
          'VM Exception while processing transaction: revert Message sender does not have permission to enqueue'
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
    //TODO test with actual transitions and actual state roots
    //TODO test above with multiple batches with different # elements and different size elements
  })

  /*
   * Test verifyElement()
   */
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
        log.debug(`testing valid proof for batch #: ${batchIndex}`)
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
        log.debug(
          `trying to correctly verify this inclusion proof: ${JSON.stringify(
            elementInclusionProof
          )}`
        )
        //run verifyElement()
        //
        const isIncluded = await canonicalTxChain.verifyElement(
          element,
          position,
          elementInclusionProof
        )
        log.debug('isIncluded: ', JSON.stringify(isIncluded))
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
      log.debug(
        `trying to falsely verify this inclusion proof: ${JSON.stringify(
          elementInclusionProof
        )}`
      )
      //Give wrong position so inclusion proof is wrong
      const wrongPosition = position + 1
      //run verifyElement()
      //
      const isIncluded = await canonicalTxChain.verifyElement(
        element,
        wrongPosition,
        elementInclusionProof
      )
      log.debug('isIncluded: ', JSON.stringify(isIncluded))
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
      log.debug(
        `trying to falsely verify this inclusion proof: ${JSON.stringify(
          elementInclusionProof
        )}`
      )
      //run verifyElement()
      //
      const isIncluded = await canonicalTxChain.verifyElement(
        element,
        wrongPosition,
        elementInclusionProof
      )
      log.debug('isIncluded: ', JSON.stringify(isIncluded))
      isIncluded.should.equal(false)
    })
  })
})
