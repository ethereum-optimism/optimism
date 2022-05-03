/* External Imports */
import { ethers } from 'hardhat'
import { Signer, Contract } from 'ethers'
import { smock, FakeContract } from '@defi-wonderland/smock'
import {
  AppendSequencerBatchParams,
  BatchContext,
  encodeAppendSequencerBatch,
  expectApprox,
} from '@eth-optimism/core-utils'
import { TransactionResponse } from '@ethersproject/abstract-provider'

/* Internal Imports */
import {
  deploy,
  L2_GAS_DISCOUNT_DIVISOR,
  ENQUEUE_GAS_COST,
  getEthTime,
  getNextBlockNumber,
  NON_ZERO_ADDRESS,
} from '../../../helpers'

// Still have some duplication from CanonicalTransactionChain.spec.ts, but it's so minimal that
// this is probably cleaner for now. Particularly since we're planning to move all of this out into
// core-utils soon anyway.
const MAX_GAS_LIMIT = 8_000_000

const appendSequencerBatch = async (
  CanonicalTransactionChain: Contract,
  batch: AppendSequencerBatchParams
): Promise<TransactionResponse> => {
  return CanonicalTransactionChain.signer.sendTransaction({
    to: CanonicalTransactionChain.address,
    data:
      ethers.utils.id('appendSequencerBatch()').slice(0, 10) +
      encodeAppendSequencerBatch(batch),
  })
}

describe('[GAS BENCHMARK] CanonicalTransactionChain [ @skip-on-coverage ]', () => {
  let sequencer: Signer
  before(async () => {
    ;[sequencer] = await ethers.getSigners()
  })

  let AddressManager: Contract
  let Fake__StateCommitmentChain: FakeContract
  let CanonicalTransactionChain: Contract
  beforeEach(async () => {
    AddressManager = await deploy('Lib_AddressManager')

    await AddressManager.setAddress(
      'OVM_Sequencer',
      await sequencer.getAddress()
    )

    Fake__StateCommitmentChain = await smock.fake<Contract>(
      'StateCommitmentChain'
    )

    await AddressManager.setAddress(
      'StateCommitmentChain',
      Fake__StateCommitmentChain.address
    )

    CanonicalTransactionChain = await deploy('CanonicalTransactionChain', {
      signer: sequencer,
      args: [
        AddressManager.address,
        MAX_GAS_LIMIT,
        L2_GAS_DISCOUNT_DIVISOR,
        ENQUEUE_GAS_COST,
      ],
    })

    const batches = await deploy('ChainStorageContainer', {
      args: [AddressManager.address, 'CanonicalTransactionChain'],
    })

    await AddressManager.setAddress(
      'ChainStorageContainer-CTC-batches',
      batches.address
    )

    await AddressManager.setAddress(
      'CanonicalTransactionChain',
      CanonicalTransactionChain.address
    )
  })

  describe('appendSequencerBatch [ @skip-on-coverage ]', () => {
    it('200 transactions in a single context', async () => {
      const timestamp = (await getEthTime(ethers.provider)) - 100
      const blockNumber = await getNextBlockNumber(ethers.provider)

      const transactionTemplate = '0x' + '11'.repeat(400)
      const transactions = []
      const numTxs = 200
      for (let i = 0; i < numTxs; i++) {
        transactions.push(transactionTemplate)
      }

      const fixedCalldataCost =
        (transactionTemplate.slice(2).length / 2) * 16 * numTxs

      const res = await appendSequencerBatch(CanonicalTransactionChain, {
        shouldStartAtElement: 0,
        totalElementsToAppend: numTxs,
        contexts: [
          {
            numSequencedTransactions: numTxs,
            numSubsequentQueueTransactions: 0,
            timestamp,
            blockNumber,
          },
        ],
        transactions,
      })

      const receipt = await res.wait()
      const gasUsed = receipt.gasUsed.toNumber()

      console.log('Fixed calldata cost:', fixedCalldataCost)
      console.log(
        'Non-calldata overhead gas cost per transaction:',
        (gasUsed - fixedCalldataCost) / numTxs
      )

      expectApprox(gasUsed, 1_402_638, {
        absoluteUpperDeviation: 1000,
        // Assert a lower bound of 1% reduction on gas cost. If your tests are breaking because your
        // contracts are too efficient, consider updating the target value!
        percentLowerDeviation: 1,
      })
    }).timeout(10_000_000)

    it('200 transactions in 200 contexts', async () => {
      const timestamp = (await getEthTime(ethers.provider)) - 100
      const blockNumber = await getNextBlockNumber(ethers.provider)

      const transactionTemplate = '0x' + '11'.repeat(400)
      const transactions = []
      const numTxs = 200
      for (let i = 0; i < numTxs; i++) {
        transactions.push(transactionTemplate)
      }

      const fixedCalldataCost =
        (transactionTemplate.slice(2).length / 2) * 16 * numTxs

      const res = await appendSequencerBatch(CanonicalTransactionChain, {
        shouldStartAtElement: 0,
        totalElementsToAppend: numTxs,
        contexts: [...Array(numTxs)].map(() => {
          return {
            numSequencedTransactions: 1,
            numSubsequentQueueTransactions: 0,
            timestamp,
            blockNumber,
          }
        }),
        transactions,
      })

      const receipt = await res.wait()
      const gasUsed = receipt.gasUsed.toNumber()

      console.log('Fixed calldata cost:', fixedCalldataCost)
      console.log(
        'Non-calldata overhead gas cost per transaction:',
        (gasUsed - fixedCalldataCost) / numTxs
      )

      expectApprox(gasUsed, 1_619_781, {
        absoluteUpperDeviation: 1000,
        // Assert a lower bound of 1% reduction on gas cost. If your tests are breaking because your
        // contracts are too efficient, consider updating the target value!
        percentLowerDeviation: 1,
      })
    }).timeout(10_000_000)

    it('100 Sequencer transactions and 100 Queue transactions in 100 contexts', async () => {
      console.log(
        `Benchmark: 100 Sequencer transactions and 100 Queue transactions in 100 contexts`
      )
      const transactionTemplate = '0x' + '11'.repeat(400)
      const transactions = []
      const numTxs = 100
      for (let i = 0; i < numTxs; i++) {
        transactions.push(transactionTemplate)
      }

      // Enqueue the transactions and record their contexts
      const target = NON_ZERO_ADDRESS
      const gasLimit = 500_000
      const data = '0x' + '12'.repeat(1234)
      const numEnqueues = numTxs

      const queueContexts: BatchContext[] = []
      for (let i = 0; i < numEnqueues; i++) {
        await CanonicalTransactionChain.enqueue(target, gasLimit, data)

        queueContexts.push({
          blockNumber: (await getNextBlockNumber(ethers.provider)) - 1,
          timestamp: await getEthTime(ethers.provider),
          numSequencedTransactions: 1,
          numSubsequentQueueTransactions: 1,
        })
      }

      const fixedCalldataCost =
        (transactionTemplate.slice(2).length / 2) * 16 * numTxs

      const res = await appendSequencerBatch(CanonicalTransactionChain, {
        shouldStartAtElement: 0,
        totalElementsToAppend: numTxs + numEnqueues,
        contexts: queueContexts,
        transactions,
      })

      const receipt = await res.wait()
      const gasUsed = receipt.gasUsed.toNumber()

      console.log('Benchmark complete.')
      console.log('Fixed calldata cost:', fixedCalldataCost)
      console.log(
        'Non-calldata overhead gas cost per transaction:',
        (gasUsed - fixedCalldataCost) / numTxs
      )

      expectApprox(gasUsed, 891_158, {
        absoluteUpperDeviation: 1000,
        // Assert a lower bound of 1% reduction on gas cost. If your tests are breaking because your
        // contracts are too efficient, consider updating the target value!
        percentLowerDeviation: 1,
      })
    }).timeout(10_000_000)
  })

  describe('enqueue [ @skip-on-coverage ]', () => {
    const data = '0x' + '12'.repeat(1234)

    let enqueueL2GasPrepaid: number
    beforeEach(async () => {
      enqueueL2GasPrepaid =
        await CanonicalTransactionChain.enqueueL2GasPrepaid()
    })

    it('cost to enqueue a transaction above the prepaid threshold', async () => {
      const l2GasLimit = 2 * enqueueL2GasPrepaid

      const res = await CanonicalTransactionChain.enqueue(
        NON_ZERO_ADDRESS,
        l2GasLimit,
        data
      )

      const receipt = await res.wait()
      const gasUsed = receipt.gasUsed.toNumber()

      expectApprox(gasUsed, 196_687, {
        absoluteUpperDeviation: 500,
        // Assert a lower bound of 1% reduction on gas cost. If your tests are breaking because your
        // contracts are too efficient, consider updating the target value!
        percentLowerDeviation: 1,
      })
    })

    it('cost to enqueue a transaction below the prepaid threshold', async () => {
      const l2GasLimit = enqueueL2GasPrepaid - 1

      const res = await CanonicalTransactionChain.enqueue(
        NON_ZERO_ADDRESS,
        l2GasLimit,
        data
      )

      const receipt = await res.wait()
      const gasUsed = receipt.gasUsed.toNumber()

      expectApprox(gasUsed, 134_100, {
        absoluteUpperDeviation: 500,
        // Assert a lower bound of 1% reduction on gas cost. If your tests are breaking because your
        // contracts are too efficient, consider updating the target value!
        percentLowerDeviation: 1,
      })
    })
  })
})
