import { expect } from '../../setup'

/* External Imports */
import * as rlp from 'rlp'
import { ethers } from '@nomiclabs/buidler'
import { Contract, ContractFactory, Signer } from 'ethers'

/* Internal Imports */
import {
  DEFAULT_OPCODE_WHITELIST_MASK,
  GAS_LIMIT,
  TxChainBatch,
  StateChainBatch,
  toHexString,
} from '../../test-helpers'
import { TestUtils } from '@eth-optimism/core-utils'

interface OVMTransactionData {
  timestamp: number
  queueOrigin: number
  ovmEntrypoint: string
  callBytes: string
  fromAddress: string
  l1MsgSenderAddress: string
  allowRevert: boolean
}

const NULL_ADDRESS = '0x' + '00'.repeat(20)
const FORCE_INCLUSION_PERIOD = 600

const makeDummyTransaction = (calldata: string): OVMTransactionData => {
  return {
    timestamp: Math.floor(Date.now() / 1000),
    queueOrigin: 0,
    ovmEntrypoint: NULL_ADDRESS,
    callBytes: calldata,
    fromAddress: NULL_ADDRESS,
    l1MsgSenderAddress: NULL_ADDRESS,
    allowRevert: false,
  }
}

const encodeTransaction = (transaction: OVMTransactionData): string => {
  return toHexString(
    rlp.encode([
      transaction.timestamp,
      transaction.queueOrigin,
      transaction.ovmEntrypoint,
      transaction.callBytes,
      transaction.fromAddress,
      transaction.l1MsgSenderAddress,
      transaction.allowRevert ? 1 : 0,
    ])
  )
}

const appendTransactionBatch = async (
  canonicalTransactionChain: Contract,
  sequencer: Signer,
  batch: string[]
): Promise<number> => {
  const timestamp = Math.floor(Date.now() / 1000)

  await canonicalTransactionChain
    .connect(sequencer)
    .appendSequencerBatch(batch, timestamp)

  return timestamp
}

const appendAndGenerateTransactionBatch = async (
  canonicalTransactionChain: Contract,
  sequencer: Signer,
  batch: string[],
  batchIndex: number = 0,
  cumulativePrevElements: number = 0
): Promise<TxChainBatch> => {
  const timestamp = await appendTransactionBatch(
    canonicalTransactionChain,
    sequencer,
    batch
  )

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

const appendAndGenerateStateBatch = async (
  stateCommitmentChain: Contract,
  batch: string[],
  batchIndex: number = 0,
  cumulativePrevElements: number = 0
): Promise<StateChainBatch> => {
  await stateCommitmentChain.appendStateBatch(batch)

  const localBatch = new StateChainBatch(
    batchIndex,
    cumulativePrevElements,
    batch
  )

  await localBatch.generateTree()

  return localBatch
}

const DUMMY_STATE_BATCH = [
  '0x' + '01'.repeat(32),
  '0x' + '02'.repeat(32),
  '0x' + '03'.repeat(32),
  '0x' + '04'.repeat(32),
]

const DUMMY_TRANSACTION_BATCH = DUMMY_STATE_BATCH.map((element) => {
  return makeDummyTransaction(element)
})

const ENCODED_DUMMY_TRANSACTION_BATCH = DUMMY_TRANSACTION_BATCH.map(
  (transaction) => {
    return encodeTransaction(transaction)
  }
)

/* Tests */
describe('FraudVerifier', () => {
  let wallet: Signer
  let sequencer: Signer
  let l1ToL2TransactionPasser: Signer
  before(async () => {
    ;[wallet, sequencer, l1ToL2TransactionPasser] = await ethers.getSigners()
  })

  let ExecutionManager: ContractFactory
  let RollupMerkleUtils: ContractFactory
  let StateCommitmentChain: ContractFactory
  let CanonicalTransactonChain: ContractFactory
  let FraudVerifier: ContractFactory
  let StubStateTransitioner: ContractFactory
  let executionManager: Contract
  let rollupMerkleUtils: Contract
  before(async () => {
    ExecutionManager = await ethers.getContractFactory('ExecutionManager')
    RollupMerkleUtils = await ethers.getContractFactory('RollupMerkleUtils')
    StateCommitmentChain = await ethers.getContractFactory(
      'StateCommitmentChain'
    )
    CanonicalTransactonChain = await ethers.getContractFactory(
      'CanonicalTransactionChain'
    )
    FraudVerifier = await ethers.getContractFactory('FraudVerifier')
    StubStateTransitioner = await ethers.getContractFactory(
      'StubStateTransitioner'
    )

    executionManager = await ExecutionManager.deploy(
      DEFAULT_OPCODE_WHITELIST_MASK,
      NULL_ADDRESS,
      GAS_LIMIT,
      true
    )

    rollupMerkleUtils = await RollupMerkleUtils.deploy()
  })

  let canonicalTransactonChain: Contract
  let stateCommitmentChain: Contract
  let fraudVerifier: Contract
  beforeEach(async () => {
    canonicalTransactonChain = await CanonicalTransactonChain.deploy(
      rollupMerkleUtils.address,
      await sequencer.getAddress(),
      await l1ToL2TransactionPasser.getAddress(),
      FORCE_INCLUSION_PERIOD
    )

    stateCommitmentChain = await StateCommitmentChain.deploy(
      rollupMerkleUtils.address,
      canonicalTransactonChain.address
    )

    fraudVerifier = await FraudVerifier.deploy(
      executionManager.address,
      stateCommitmentChain.address,
      canonicalTransactonChain.address,
      true // Throw the verifier into testing mode.
    )

    await stateCommitmentChain.setFraudVerifier(fraudVerifier.address)
  })

  let transactionBatch: TxChainBatch
  let stateBatch: StateChainBatch
  beforeEach(async () => {
    transactionBatch = await appendAndGenerateTransactionBatch(
      canonicalTransactonChain,
      sequencer,
      ENCODED_DUMMY_TRANSACTION_BATCH
    )

    stateBatch = await appendAndGenerateStateBatch(
      stateCommitmentChain,
      DUMMY_STATE_BATCH
    )
  })

  describe('initializeFraudVerification', async () => {
    it('should correctly initialize with a valid state root and transaction', async () => {
      const preStateRoot = DUMMY_STATE_BATCH[0]
      const preStateRootProof = await stateBatch.getElementInclusionProof(0)

      const transaction = DUMMY_TRANSACTION_BATCH[0]
      const transactionIndex = transactionBatch.getPosition(0)
      const transactionProof = await transactionBatch.getElementInclusionProof(
        0
      )

      await fraudVerifier.initializeFraudVerification(
        transactionIndex,
        preStateRoot,
        preStateRootProof,
        transaction,
        transactionProof
      )

      expect(
        await fraudVerifier.hasStateTransitioner(transactionIndex, preStateRoot)
      ).to.equal(true)
    })

    it('should return if initializing twice', async () => {
      const preStateRoot = DUMMY_STATE_BATCH[0]
      const preStateRootProof = await stateBatch.getElementInclusionProof(0)

      const transaction = DUMMY_TRANSACTION_BATCH[0]
      const transactionIndex = transactionBatch.getPosition(0)
      const transactionProof = await transactionBatch.getElementInclusionProof(
        0
      )

      await fraudVerifier.initializeFraudVerification(
        transactionIndex,
        preStateRoot,
        preStateRootProof,
        transaction,
        transactionProof
      )

      expect(
        await fraudVerifier.hasStateTransitioner(transactionIndex, preStateRoot)
      ).to.equal(true)

      // Initializing again should execute correctly without actually creating
      // a new state transitioner.
      await fraudVerifier.initializeFraudVerification(
        transactionIndex,
        preStateRoot,
        preStateRootProof,
        transaction,
        transactionProof
      )

      expect(
        await fraudVerifier.hasStateTransitioner(transactionIndex, preStateRoot)
      ).to.equal(true)
    })

    it('should reject an invalid state root', async () => {
      // Using the wrong state root.
      const preStateRoot = DUMMY_STATE_BATCH[1]
      const preStateRootProof = await stateBatch.getElementInclusionProof(1)

      const transaction = DUMMY_TRANSACTION_BATCH[0]
      const transactionIndex = transactionBatch.getPosition(0)
      const transactionProof = await transactionBatch.getElementInclusionProof(
        0
      )

      await TestUtils.assertRevertsAsync(
        'Provided pre-state root is invalid.',
        async () => {
          await fraudVerifier.initializeFraudVerification(
            transactionIndex,
            preStateRoot,
            preStateRootProof,
            transaction,
            transactionProof
          )
        }
      )

      expect(
        await fraudVerifier.hasStateTransitioner(transactionIndex, preStateRoot)
      ).to.equal(false)
    })

    it('should reject an invalid transaction', async () => {
      const preStateRoot = DUMMY_STATE_BATCH[0]
      const preStateRootProof = await stateBatch.getElementInclusionProof(0)

      // Using the wrong transaction data.
      const transaction = DUMMY_TRANSACTION_BATCH[1]
      const transactionIndex = transactionBatch.getPosition(0)
      const transactionProof = await transactionBatch.getElementInclusionProof(
        0
      )

      await TestUtils.assertRevertsAsync(
        'Provided transaction data is invalid.',
        async () => {
          await fraudVerifier.initializeFraudVerification(
            transactionIndex,
            preStateRoot,
            preStateRootProof,
            transaction,
            transactionProof
          )
        }
      )

      expect(
        await fraudVerifier.hasStateTransitioner(transactionIndex, preStateRoot)
      ).to.equal(false)
    })
  })

  describe('finalizeFraudVerification', async () => {
    let stateTransitioner: Contract
    beforeEach(async () => {
      const preStateRoot = DUMMY_STATE_BATCH[0]
      const preStateRootProof = await stateBatch.getElementInclusionProof(0)

      const transaction = DUMMY_TRANSACTION_BATCH[0]
      const transactionIndex = transactionBatch.getPosition(0)
      const transactionProof = await transactionBatch.getElementInclusionProof(
        0
      )

      await fraudVerifier.initializeFraudVerification(
        transactionIndex,
        preStateRoot,
        preStateRootProof,
        transaction,
        transactionProof
      )

      const stateTransitionerAddress = await fraudVerifier.stateTransitioners(
        transactionIndex
      )
      stateTransitioner = StubStateTransitioner.attach(stateTransitionerAddress)
    })

    it('should correctly finalize when the computed state root differs', async () => {
      const preStateRoot = DUMMY_STATE_BATCH[0]
      const preStateRootProof = await stateBatch.getElementInclusionProof(0)

      const postStateRoot = DUMMY_STATE_BATCH[1]
      const postStateRootProof = await stateBatch.getElementInclusionProof(1)

      const transactionIndex = transactionBatch.getPosition(0)

      await stateTransitioner.setStateRoot('0x' + '00'.repeat(32))
      await stateTransitioner.completeTransition()

      expect(await stateCommitmentChain.getBatchesLength()).to.equal(1)

      await fraudVerifier.finalizeFraudVerification(
        transactionIndex,
        preStateRoot,
        preStateRootProof,
        postStateRoot,
        postStateRootProof
      )

      expect(await stateCommitmentChain.getBatchesLength()).to.equal(0)
    })

    it('should revert when the state transitioner has not been finalized', async () => {
      const preStateRoot = DUMMY_STATE_BATCH[0]
      const preStateRootProof = await stateBatch.getElementInclusionProof(0)

      const postStateRoot = DUMMY_STATE_BATCH[1]
      const postStateRootProof = await stateBatch.getElementInclusionProof(1)

      const transactionIndex = transactionBatch.getPosition(0)

      // Not finalizing the state transitioner.
      await stateTransitioner.setStateRoot('0x' + '00'.repeat(32))

      expect(await stateCommitmentChain.getBatchesLength()).to.equal(1)

      await TestUtils.assertRevertsAsync(
        'State transition process has not been completed.',
        async () => {
          await fraudVerifier.finalizeFraudVerification(
            transactionIndex,
            preStateRoot,
            preStateRootProof,
            postStateRoot,
            postStateRootProof
          )
        }
      )

      expect(await stateCommitmentChain.getBatchesLength()).to.equal(1)
    })

    it('should revert when the provided pre-state root does not match the state transitioner', async () => {
      // Using the wrong pre-state root.
      const preStateRoot = DUMMY_STATE_BATCH[1]
      const preStateRootProof = await stateBatch.getElementInclusionProof(1)

      const postStateRoot = DUMMY_STATE_BATCH[1]
      const postStateRootProof = await stateBatch.getElementInclusionProof(1)

      const transactionIndex = transactionBatch.getPosition(0)

      await stateTransitioner.setStateRoot('0x' + '00'.repeat(32))
      await stateTransitioner.completeTransition()

      expect(await stateCommitmentChain.getBatchesLength()).to.equal(1)

      await TestUtils.assertRevertsAsync(
        'Provided pre-state root does not match StateTransitioner.',
        async () => {
          await fraudVerifier.finalizeFraudVerification(
            transactionIndex,
            preStateRoot,
            preStateRootProof,
            postStateRoot,
            postStateRootProof
          )
        }
      )

      expect(await stateCommitmentChain.getBatchesLength()).to.equal(1)
    })

    it('should revert when the provided pre-state root is invalid', async () => {
      // Using the right root with an invalid proof.
      const preStateRoot = DUMMY_STATE_BATCH[0]
      const preStateRootProof = await stateBatch.getElementInclusionProof(1)

      const postStateRoot = DUMMY_STATE_BATCH[1]
      const postStateRootProof = await stateBatch.getElementInclusionProof(1)

      const transactionIndex = transactionBatch.getPosition(0)

      await stateTransitioner.setStateRoot('0x' + '00'.repeat(32))
      await stateTransitioner.completeTransition()

      expect(await stateCommitmentChain.getBatchesLength()).to.equal(1)

      await TestUtils.assertRevertsAsync(
        'Provided pre-state root is invalid.',
        async () => {
          await fraudVerifier.finalizeFraudVerification(
            transactionIndex,
            preStateRoot,
            preStateRootProof,
            postStateRoot,
            postStateRootProof
          )
        }
      )

      expect(await stateCommitmentChain.getBatchesLength()).to.equal(1)
    })

    it('should revert when the provided post-state root is invalid', async () => {
      const preStateRoot = DUMMY_STATE_BATCH[0]
      const preStateRootProof = await stateBatch.getElementInclusionProof(0)

      // Using the wrong pre-state root.
      const postStateRoot = DUMMY_STATE_BATCH[2]
      const postStateRootProof = await stateBatch.getElementInclusionProof(2)

      const transactionIndex = transactionBatch.getPosition(0)

      await stateTransitioner.setStateRoot('0x' + '00'.repeat(32))
      await stateTransitioner.completeTransition()

      expect(await stateCommitmentChain.getBatchesLength()).to.equal(1)

      await TestUtils.assertRevertsAsync(
        'Provided post-state root is invalid.',
        async () => {
          await fraudVerifier.finalizeFraudVerification(
            transactionIndex,
            preStateRoot,
            preStateRootProof,
            postStateRoot,
            postStateRootProof
          )
        }
      )

      expect(await stateCommitmentChain.getBatchesLength()).to.equal(1)
    })

    it('should revert when the provided post-state root matches the state transitioner', async () => {
      const preStateRoot = DUMMY_STATE_BATCH[0]
      const preStateRootProof = await stateBatch.getElementInclusionProof(0)

      const postStateRoot = DUMMY_STATE_BATCH[1]
      const postStateRootProof = await stateBatch.getElementInclusionProof(1)

      const transactionIndex = transactionBatch.getPosition(0)

      // Setting the root to match the given post-state root.
      await stateTransitioner.setStateRoot(postStateRoot)
      await stateTransitioner.completeTransition()

      expect(await stateCommitmentChain.getBatchesLength()).to.equal(1)

      await TestUtils.assertRevertsAsync(
        'State transition was not fraudulent.',
        async () => {
          await fraudVerifier.finalizeFraudVerification(
            transactionIndex,
            preStateRoot,
            preStateRootProof,
            postStateRoot,
            postStateRootProof
          )
        }
      )

      expect(await stateCommitmentChain.getBatchesLength()).to.equal(1)
    })
  })
})
