import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, sleep, TestUtils } from '@eth-optimism/core-utils'
import { Contract, Signer, ContractFactory } from 'ethers'

/* Internal Imports */
import { makeRandomBatchOfSize, StateChainBatch } from '../../test-helpers'

/* Logging */
const log = getLogger('state-commitment-chain', true)

/* Tests */
describe('StateCommitmentChain', () => {
  const DEFAULT_STATE_BATCH = ['0x1234', '0x5678']
  const DEFAULT_TX_BATCH = [
    '0x1234',
    '0x5678',
    '0x1234',
    '0x5678',
    '0x1234',
    '0x5678',
    '0x1234',
    '0x5678',
    '0x1234',
    '0x5678',
  ]
  const FORCE_INCLUSION_PERIOD = 600

  let wallet: Signer
  let sequencer: Signer
  let l1ToL2TransactionPasser: Signer
  let fraudVerifier: Signer
  let randomWallet: Signer
  before(async () => {
    ;[
      wallet,
      sequencer,
      l1ToL2TransactionPasser,
      fraudVerifier,
      randomWallet,
    ] = await ethers.getSigners()
  })

  let stateChain: Contract
  let canonicalTxChain: Contract
  let rollupMerkleUtils: Contract

  const appendAndGenerateStateBatch = async (
    batch: string[],
    batchIndex: number = 0,
    cumulativePrevElements: number = 0
  ): Promise<StateChainBatch> => {
    await stateChain.appendStateBatch(batch)
    // Generate a local version of the rollup batch
    const localBatch = new StateChainBatch(
      batchIndex,
      cumulativePrevElements,
      batch
    )
    await localBatch.generateTree()
    return localBatch
  }

  const appendTxBatch = async (batch: string[]): Promise<void> => {
    const timestamp = Math.floor(Date.now() / 1000)
    // Submit the rollup batch on-chain
    await canonicalTxChain
      .connect(sequencer)
      .appendSequencerBatch(batch, timestamp)
  }

  let RollupMerkleUtils: ContractFactory
  let CanonicalTransactionChain: ContractFactory
  let StateCommitmentChain: ContractFactory
  before(async () => {
    RollupMerkleUtils = await ethers.getContractFactory('RollupMerkleUtils')
    CanonicalTransactionChain = await ethers.getContractFactory(
      'CanonicalTransactionChain'
    )
    StateCommitmentChain = await ethers.getContractFactory(
      'StateCommitmentChain'
    )

    rollupMerkleUtils = await RollupMerkleUtils.deploy()

    canonicalTxChain = await CanonicalTransactionChain.deploy(
      rollupMerkleUtils.address,
      await sequencer.getAddress(),
      await l1ToL2TransactionPasser.getAddress(),
      FORCE_INCLUSION_PERIOD
    )

    await appendTxBatch(DEFAULT_TX_BATCH)
  })

  beforeEach(async () => {
    stateChain = await StateCommitmentChain.deploy(
      rollupMerkleUtils.address,
      canonicalTxChain.address
    )

    await stateChain.setFraudVerifier(await fraudVerifier.getAddress())
  })

  describe('appendStateBatch()', async () => {
    it('should allow appending of state batches from any wallet', async () => {
      await stateChain
        .connect(randomWallet)
        .appendStateBatch(DEFAULT_STATE_BATCH)
    })

    it('should throw if submitting an empty batch', async () => {
      const emptyBatch = []
      await TestUtils.assertRevertsAsync(
        'Cannot submit an empty state commitment batch',
        async () => {
          await stateChain.appendStateBatch(emptyBatch)
        }
      )
    })

    it('should add to batches array', async () => {
      await stateChain.appendStateBatch(DEFAULT_STATE_BATCH)
      const batchesLength = await stateChain.getBatchesLength()
      batchesLength.toNumber().should.equal(1)
    })

    it('should update cumulativeNumElements correctly', async () => {
      await stateChain.appendStateBatch(DEFAULT_STATE_BATCH)
      const cumulativeNumElements = await stateChain.cumulativeNumElements.call()
      cumulativeNumElements.toNumber().should.equal(DEFAULT_STATE_BATCH.length)
    })

    it('should calculate batchHeaderHash correctly', async () => {
      const localBatch = await appendAndGenerateStateBatch(DEFAULT_STATE_BATCH)
      const expectedBatchHeaderHash = await localBatch.hashBatchHeader()
      const calculatedBatchHeaderHash = await stateChain.batches(0)
      calculatedBatchHeaderHash.should.equal(expectedBatchHeaderHash)
    })

    it('should add multiple batches correctly', async () => {
      const numBatches = 3
      let expectedNumElements = 0
      for (let batchIndex = 0; batchIndex < numBatches; batchIndex++) {
        const batch = makeRandomBatchOfSize(batchIndex + 1)
        const cumulativePrevElements = expectedNumElements
        const localBatch = await appendAndGenerateStateBatch(
          batch,
          batchIndex,
          cumulativePrevElements
        )
        const expectedBatchHeaderHash = await localBatch.hashBatchHeader()
        const calculatedBatchHeaderHash = await stateChain.batches(batchIndex)
        calculatedBatchHeaderHash.should.equal(expectedBatchHeaderHash)
        expectedNumElements += batch.length
      }
      const cumulativeNumElements = await stateChain.cumulativeNumElements.call()
      cumulativeNumElements.toNumber().should.equal(expectedNumElements)
      const batchesLength = await stateChain.getBatchesLength()
      batchesLength.toNumber().should.equal(numBatches)
    })

    it('should throw if submitting more state commitments than number of txs in canonical tx chain', async () => {
      const numBatches = 5
      for (let i = 0; i < numBatches; i++) {
        await stateChain.appendStateBatch(DEFAULT_STATE_BATCH)
      }
      await TestUtils.assertRevertsAsync(
        'Cannot append more state commitments than total number of transactions in CanonicalTransactionChain',
        async () => {
          await stateChain.appendStateBatch(DEFAULT_STATE_BATCH)
        }
      )
    })
  })

  describe('verifyElement() ', async () => {
    it('should return true for valid elements for different batches and elements', async () => {
      // add enough transaction batches so # txs > # state roots
      await appendTxBatch(DEFAULT_TX_BATCH)

      const numBatches = 4
      let cumulativePrevElements = 0
      for (let batchIndex = 0; batchIndex < numBatches; batchIndex++) {
        const batchSize = batchIndex * batchIndex + 1 // 1, 2, 5, 10
        const batch = makeRandomBatchOfSize(batchSize)
        const localBatch = await appendAndGenerateStateBatch(
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
          const isIncluded = await stateChain.verifyElement(
            element,
            position,
            elementInclusionProof
          )
          isIncluded.should.equal(true)
        }
      }
    })

    it('should return false for wrong position with wrong indexInBatch', async () => {
      const batch = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
      const localBatch = await appendAndGenerateStateBatch(batch)
      const elementIndex = 1
      const element = batch[elementIndex]
      const position = localBatch.getPosition(elementIndex)
      const elementInclusionProof = await localBatch.getElementInclusionProof(
        elementIndex
      )
      //Give wrong position so inclusion proof is wrong
      const wrongPosition = position + 1
      const isIncluded = await stateChain.verifyElement(
        element,
        wrongPosition,
        elementInclusionProof
      )
      isIncluded.should.equal(false)
    })

    it('should return false for wrong position and matching indexInBatch', async () => {
      const batch = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
      const localBatch = await appendAndGenerateStateBatch(batch)
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
      const isIncluded = await stateChain.verifyElement(
        element,
        wrongPosition,
        elementInclusionProof
      )
      isIncluded.should.equal(false)
    })
  })

  describe('deleteAfterInclusive() ', async () => {
    it('should not allow deletion from address other than fraud verifier', async () => {
      const cumulativePrevElements = 0
      const batchIndex = 0
      const localBatch = await appendAndGenerateStateBatch(DEFAULT_STATE_BATCH)
      const batchHeader = {
        elementsMerkleRoot: await localBatch.elementsMerkleTree.getRootHash(),
        numElementsInBatch: DEFAULT_STATE_BATCH.length,
        cumulativePrevElements,
      }
      await TestUtils.assertRevertsAsync(
        'Only FraudVerifier has permission to delete state batches',
        async () => {
          await stateChain.connect(randomWallet).deleteAfterInclusive(
            batchIndex, // delete the single appended batch
            batchHeader
          )
        }
      )
    })
    describe('when a single batch is deleted', async () => {
      beforeEach(async () => {
        const cumulativePrevElements = 0
        const batchIndex = 0
        const localBatch = await appendAndGenerateStateBatch(
          DEFAULT_STATE_BATCH
        )
        const batchHeader = {
          elementsMerkleRoot: await localBatch.elementsMerkleTree.getRootHash(),
          numElementsInBatch: DEFAULT_STATE_BATCH.length,
          cumulativePrevElements,
        }
        await stateChain.connect(fraudVerifier).deleteAfterInclusive(
          batchIndex, // delete the single appended batch
          batchHeader
        )
      })

      it('should successfully update the batches array', async () => {
        const batchesLength = await stateChain.getBatchesLength()
        batchesLength.should.equal(0)
      })

      it('should successfully append a batch after deletion', async () => {
        const localBatch = await appendAndGenerateStateBatch(
          DEFAULT_STATE_BATCH
        )
        const expectedBatchHeaderHash = await localBatch.hashBatchHeader()
        const calculatedBatchHeaderHash = await stateChain.batches(0)
        calculatedBatchHeaderHash.should.equal(expectedBatchHeaderHash)
      })
    })

    it('should delete many batches', async () => {
      const deleteBatchIndex = 0
      const localBatches = []
      for (let batchIndex = 0; batchIndex < 5; batchIndex++) {
        const cumulativePrevElements = batchIndex * DEFAULT_STATE_BATCH.length
        const localBatch = await appendAndGenerateStateBatch(
          DEFAULT_STATE_BATCH,
          batchIndex,
          cumulativePrevElements
        )
        localBatches.push(localBatch)
      }
      const deleteBatch = localBatches[deleteBatchIndex]
      const batchHeader = {
        elementsMerkleRoot: deleteBatch.elementsMerkleTree.getRootHash(),
        numElementsInBatch: DEFAULT_STATE_BATCH.length,
        cumulativePrevElements: deleteBatch.cumulativePrevElements,
      }
      await stateChain.connect(fraudVerifier).deleteAfterInclusive(
        deleteBatchIndex, // delete all batches (including and after batch 0)
        batchHeader
      )
      const batchesLength = await stateChain.getBatchesLength()
      batchesLength.should.equal(0)
    })

    it('should revert if batchHeader is incorrect', async () => {
      const cumulativePrevElements = 0
      const batchIndex = 0
      const localBatch = await appendAndGenerateStateBatch(DEFAULT_STATE_BATCH)
      const batchHeader = {
        elementsMerkleRoot: await localBatch.elementsMerkleTree.getRootHash(),
        numElementsInBatch: DEFAULT_STATE_BATCH.length + 1, // increment to make header incorrect
        cumulativePrevElements,
      }
      await TestUtils.assertRevertsAsync(
        'Calculated batch header is different than expected batch header',
        async () => {
          await stateChain.connect(fraudVerifier).deleteAfterInclusive(
            batchIndex, // delete the single appended batch
            batchHeader
          )
        }
      )
    })

    it('should revert if trying to delete a batch outside of valid range', async () => {
      const cumulativePrevElements = 0
      const batchIndex = 1 // outside of range
      const localBatch = await appendAndGenerateStateBatch(DEFAULT_STATE_BATCH)
      const batchHeader = {
        elementsMerkleRoot: await localBatch.elementsMerkleTree.getRootHash(),
        numElementsInBatch: DEFAULT_STATE_BATCH.length + 1, // increment to make header incorrect
        cumulativePrevElements,
      }
      await TestUtils.assertRevertsAsync(
        'Cannot delete batches outside of valid range',
        async () => {
          await stateChain
            .connect(fraudVerifier)
            .deleteAfterInclusive(batchIndex, batchHeader)
        }
      )
    })
  })

  describe('Event Emitting', () => {
    it('should emit StateBatchAppended when state batch is appended', async () => {
      let receivedBatchHeaderHash: string
      stateChain.on(stateChain.filters['StateBatchAppended'](), (...data) => {
        receivedBatchHeaderHash = data[0]
      })
      const localBatch = await appendAndGenerateStateBatch(DEFAULT_STATE_BATCH)

      await sleep(5_000)

      const batchReceived: boolean = !!receivedBatchHeaderHash
      batchReceived.should.equal(
        true,
        `State Batch Appended event not received!`
      )
      receivedBatchHeaderHash.should.equal(
        await localBatch.hashBatchHeader(),
        `State Batch Appended event has incorrect batch header hash!`
      )
    })
  })
})
