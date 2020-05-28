import '../setup'

/* External Imports */
import { getLogger, TestUtils } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { Contract } from 'ethers'

/* Internal Imports */
import { StateChainBatch } from './RLhelper'

/* Logging */
const log = getLogger('state-commitment-chain', true)

/* Contract Imports */
import * as StateCommitmentChain from '../../build/StateCommitmentChain.json'
import * as L1ToL2TransactionQueue from '../../build/L1ToL2TransactionQueue.json'
import * as SafetyTransactionQueue from '../../build/SafetyTransactionQueue.json'
import * as RollupMerkleUtils from '../../build/RollupMerkleUtils.json'

/* Begin tests */
describe('StateCommitmentChain', () => {
  const provider = createMockProvider()
  const [wallet, canonicalTransactionChain, randomWallet] = getWallets(provider)
  let stateChain
  let rollupMerkleUtils
  const DEFAULT_BATCH = ['0x1234', '0x5678']
  const DEFAULT_STATE_ROOT = '0x1234'

  const appendAndGenerateBatch = async (
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

  /* Link libraries before tests */
  before(async () => {
    rollupMerkleUtils = await deployContract(wallet, RollupMerkleUtils, [], {
      gasLimit: 6700000,
    })
  })

  /* Deploy a new RollupChain before each test */
  beforeEach(async () => {
    stateChain = await deployContract(
      wallet,
      StateCommitmentChain,
      [rollupMerkleUtils.address, canonicalTransactionChain.address],
      {
        gasLimit: 6700000,
      }
    )
  })

  describe('appendStateBatch()', async () => {
    it('should not throw when appending a batch from any wallet', async () => {
      await stateChain.connect(randomWallet).appendStateBatch(DEFAULT_BATCH)
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
      await stateChain.appendStateBatch(DEFAULT_BATCH)
      const batchesLength = await stateChain.getBatchesLength()
      batchesLength.toNumber().should.equal(1)
    })

    it('should update cumulativeNumElements correctly', async () => {
      await stateChain.appendStateBatch(DEFAULT_BATCH)
      const cumulativeNumElements = await stateChain.cumulativeNumElements.call()
      cumulativeNumElements.toNumber().should.equal(DEFAULT_BATCH.length)
    })

    it('should calculate batchHeaderHash correctly', async () => {
      const localBatch = await appendAndGenerateBatch(DEFAULT_BATCH)
      const expectedBatchHeaderHash = await localBatch.hashBatchHeader()
      const calculatedBatchHeaderHash = await stateChain.batches(0)
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
        const calculatedBatchHeaderHash = await stateChain.batches(batchIndex)
        calculatedBatchHeaderHash.should.equal(expectedBatchHeaderHash)
      }
      const cumulativeNumElements = await stateChain.cumulativeNumElements.call()
      cumulativeNumElements
        .toNumber()
        .should.equal(numBatchs * DEFAULT_BATCH.length)
      const batchesLength = await stateChain.getBatchesLength()
      batchesLength.toNumber().should.equal(numBatchs)
    })
  })
})
