import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, TestUtils } from '@eth-optimism/core-utils'
import { Contract, Signer, ContractFactory } from 'ethers'

/* Internal Imports */
import { StateChainBatch, TxChainBatch } from '../../test-helpers'

/* Logging */
const log = getLogger('batch-submitter', true)

/* Tests */
describe('SequencerBatchSubmitter', () => {
  const DEFAULT_STATE_BATCH = ['0x1234', '0x5678']
  const DEFAULT_TX_BATCH = ['0xabcd', '0xef12']
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
  let sequencerBatchSubmitter: Contract

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

  let RollupMerkleUtils: ContractFactory
  let SequencerBatchSubmitter: ContractFactory
  let CanonicalTransactionChain: ContractFactory
  let StateCommitmentChain: ContractFactory
  before(async () => {
    RollupMerkleUtils = await ethers.getContractFactory('RollupMerkleUtils')
    SequencerBatchSubmitter = await ethers.getContractFactory(
      'SequencerBatchSubmitter'
    )
    CanonicalTransactionChain = await ethers.getContractFactory(
      'CanonicalTransactionChain'
    )
    StateCommitmentChain = await ethers.getContractFactory(
      'StateCommitmentChain'
    )

    rollupMerkleUtils = await RollupMerkleUtils.deploy()
  })

  beforeEach(async () => {
    sequencerBatchSubmitter = await SequencerBatchSubmitter.deploy(
      await sequencer.getAddress()
    )

    canonicalTxChain = await CanonicalTransactionChain.deploy(
      rollupMerkleUtils.address,
      sequencerBatchSubmitter.address,
      await l1ToL2TransactionPasser.getAddress(),
      FORCE_INCLUSION_PERIOD
    )

    stateChain = await StateCommitmentChain.deploy(
      rollupMerkleUtils.address,
      canonicalTxChain.address,
      await fraudVerifier.getAddress()
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
