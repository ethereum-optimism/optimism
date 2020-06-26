import '../../setup'

/* External Imports */
import { getLogger, TestUtils } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { Contract } from 'ethers'

/* Internal Imports */
import { StateChainBatch, TxChainBatch } from '../../test-helpers/rl-helpers'

/* Contract Imports */
import * as StateCommitmentChain from '../../../build/contracts/StateCommitmentChain.json'
import * as CanonicalTransactionChain from '../../../build/contracts/CanonicalTransactionChain.json'
import * as RollupMerkleUtils from '../../../build/contracts/RollupMerkleUtils.json'
import * as SequencerBatchSubmitter from '../../../build/contracts/SequencerBatchSubmitter.json'

/* Logging */
const log = getLogger('batch-submitter', true)

/* Tests */
describe('SequencerBatchSubmitter', () => {
  const provider = createMockProvider()
  const [
    wallet,
    sequencer,
    l1ToL2TransactionPasser,
    fraudVerifier,
    randomWallet,
  ] = getWallets(provider)
  let stateChain
  let canonicalTxChain
  let rollupMerkleUtils
  let sequencerBatchSubmitter
  const DEFAULT_STATE_BATCH = ['0x1234', '0x5678']
  const DEFAULT_TX_BATCH = ['0xabcd', '0xef12']
  const FORCE_INCLUSION_PERIOD = 600

  const generateStateBatch = async (
    batch: string[],
    batchIndex: number = 0,
    cumulativePrevElements: number = 0
  ): Promise<StateChainBatch> => {
    const localBatch = new StateChainBatch(
      batchIndex,
      cumulativePrevElements,
      batch
    )
    await localBatch.generateTree()
    return localBatch
  }

  const generateTxBatch = async (
    batch: string[],
    timestamp: number,
    batchIndex: number = 0,
    cumulativePrevElements: number = 0
  ): Promise<TxChainBatch> => {
    const localBatch = new TxChainBatch(
      timestamp,
      false,
      batchIndex,
      cumulativePrevElements,
      batch
    )
    await localBatch.generateTree()
    return localBatch
  }

  before(async () => {
    rollupMerkleUtils = await deployContract(wallet, RollupMerkleUtils, [], {
      gasLimit: 6700000,
    })
  })

  beforeEach(async () => {
    sequencerBatchSubmitter = await deployContract(
      wallet,
      SequencerBatchSubmitter,
      [sequencer.address],
      {
        gasLimit: 6700000,
      }
    )

    canonicalTxChain = await deployContract(
      wallet,
      CanonicalTransactionChain,
      [
        rollupMerkleUtils.address,
        sequencerBatchSubmitter.address,
        l1ToL2TransactionPasser.address,
        FORCE_INCLUSION_PERIOD,
      ],
      {
        gasLimit: 6700000,
      }
    )

    stateChain = await deployContract(
      wallet,
      StateCommitmentChain,
      [
        rollupMerkleUtils.address,
        canonicalTxChain.address,
        fraudVerifier.address,
      ],
      {
        gasLimit: 6700000,
      }
    )

    await sequencerBatchSubmitter
      .connect(sequencer)
      .initialize(canonicalTxChain.address, stateChain.address)
  })

  describe('appendTransitionBatch()', async () => {
    it('should reject appending of transition batches from non-sequencer', async () => {
      const timestamp = Math.floor(Date.now() / 1000)
      await TestUtils.assertRevertsAsync(
        'Only the sequencer may perform this action',
        async () => {
          await sequencerBatchSubmitter
            .connect(randomWallet)
            .appendTransitionBatch(
              DEFAULT_TX_BATCH,
              timestamp,
              DEFAULT_STATE_BATCH
            )
        }
      )
    })

    it('should not allow appending differently sized tx and state batches', async () => {
      const alteredTxBatch = ['0xabcd', '0xef12', '0x3456']
      const timestamp = Math.floor(Date.now() / 1000)
      await TestUtils.assertRevertsAsync(
        'Must append the same number of state roots and transactions',
        async () => {
          await sequencerBatchSubmitter
            .connect(sequencer)
            .appendTransitionBatch(
              alteredTxBatch,
              timestamp,
              DEFAULT_STATE_BATCH
            )
        }
      )
    })

    it('should successfully append transition batch from the sequencer', async () => {
      const timestamp = Math.floor(Date.now() / 1000)
      const localTxBatch = await generateTxBatch(DEFAULT_TX_BATCH, timestamp)
      const localStateBatch = await generateStateBatch(DEFAULT_STATE_BATCH)
      await sequencerBatchSubmitter
        .connect(sequencer)
        .appendTransitionBatch(DEFAULT_TX_BATCH, timestamp, DEFAULT_STATE_BATCH)

      const expectedTxBatchHeaderHash = await localTxBatch.hashBatchHeader()
      const calculatedTxBatchHeaderHash = await canonicalTxChain.batches(0)
      calculatedTxBatchHeaderHash.should.equal(expectedTxBatchHeaderHash)

      const expectedStateBatchHeaderHash = await localStateBatch.hashBatchHeader()
      const calculatedStateBatchHeaderHash = await stateChain.batches(0)
      calculatedStateBatchHeaderHash.should.equal(expectedStateBatchHeaderHash)
    })
  })
})
