import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, TestUtils } from '@eth-optimism/core-utils'
import { Contract, Signer, ContractFactory } from 'ethers'

/* Internal Imports */
import {
  DEFAULT_FORCE_INCLUSION_PERIOD,
  makeAddressResolver,
  AddressResolverMapping,
  deployAndRegister,
  generateTxBatch,
  generateStateBatch,
} from '../../test-helpers'

/* Logging */
const log = getLogger('batch-submitter', true)

/* Tests */
describe('SequencerBatchSubmitter', () => {
  const DEFAULT_STATE_BATCH = ['0x1234', '0x5678']
  const DEFAULT_TX_BATCH = ['0xabcd', '0xef12']

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

  let resolver: AddressResolverMapping
  before(async () => {
    resolver = await makeAddressResolver(wallet)

    await resolver.addressResolver.setAddress(
      'FraudVerifier',
      await fraudVerifier.getAddress()
    )
  })

  let SequencerBatchSubmitter: ContractFactory
  let CanonicalTransactionChain: ContractFactory
  let StateCommitmentChain: ContractFactory
  before(async () => {
    SequencerBatchSubmitter = await ethers.getContractFactory(
      'SequencerBatchSubmitter'
    )
    CanonicalTransactionChain = await ethers.getContractFactory(
      'CanonicalTransactionChain'
    )
    StateCommitmentChain = await ethers.getContractFactory(
      'StateCommitmentChain'
    )
  })

  let stateChain: Contract
  let canonicalTxChain: Contract
  let sequencerBatchSubmitter: Contract
  beforeEach(async () => {
    sequencerBatchSubmitter = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'SequencerBatchSubmitter',
      {
        factory: SequencerBatchSubmitter,
        params: [
          resolver.addressResolver.address,
          await sequencer.getAddress(),
        ],
      }
    )

    canonicalTxChain = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'CanonicalTransactionChain',
      {
        factory: CanonicalTransactionChain,
        params: [
          resolver.addressResolver.address,
          sequencerBatchSubmitter.address,
          await l1ToL2TransactionPasser.getAddress(),
          DEFAULT_FORCE_INCLUSION_PERIOD,
        ],
      }
    )

    stateChain = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'StateCommitmentChain',
      {
        factory: StateCommitmentChain,
        params: [resolver.addressResolver.address],
      }
    )
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
