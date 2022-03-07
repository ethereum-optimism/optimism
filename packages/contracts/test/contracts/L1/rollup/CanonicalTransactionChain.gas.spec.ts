/* External Imports */
import { ethers } from 'hardhat'
import { Signer, ContractFactory, Contract } from 'ethers'
import { smock, FakeContract } from '@defi-wonderland/smock'
import {
  AppendSequencerBatchParams,
  BatchContext,
  encodeAppendSequencerBatch,
  expectApprox,
} from '@eth-optimism/core-utils'
import { TransactionResponse } from '@ethersproject/abstract-provider'
import { keccak256 } from 'ethers/lib/utils'

/* Internal Imports */
import {
  makeAddressManager,
  setProxyTarget,
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
  const methodId = keccak256(Buffer.from('appendSequencerBatch()')).slice(2, 10)
  const calldata = encodeAppendSequencerBatch(batch)
  return CanonicalTransactionChain.signer.sendTransaction({
    to: CanonicalTransactionChain.address,
    data: '0x' + methodId + calldata,
  })
}

describe('[GAS BENCHMARK] CanonicalTransactionChain [ @skip-on-coverage ]', () => {
  let sequencer: Signer
  before(async () => {
    ;[sequencer] = await ethers.getSigners()
  })

  let AddressManager: Contract
  let Fake__StateCommitmentChain: FakeContract
  before(async () => {
    AddressManager = await makeAddressManager()
    await AddressManager.setAddress(
      'OVM_Sequencer',
      await sequencer.getAddress()
    )

    Fake__StateCommitmentChain = await smock.fake<Contract>(
      await ethers.getContractFactory('StateCommitmentChain')
    )

    await setProxyTarget(
      AddressManager,
      'StateCommitmentChain',
      Fake__StateCommitmentChain
    )
  })

  let Factory__CanonicalTransactionChain: ContractFactory
  let Factory__ChainStorageContainer: ContractFactory
  before(async () => {
    Factory__CanonicalTransactionChain = await ethers.getContractFactory(
      'CanonicalTransactionChain'
    )

    Factory__ChainStorageContainer = await ethers.getContractFactory(
      'ChainStorageContainer'
    )
  })

  let CanonicalTransactionChain: Contract
  beforeEach(async () => {
    CanonicalTransactionChain = await Factory__CanonicalTransactionChain.deploy(
      AddressManager.address,
      MAX_GAS_LIMIT,
      L2_GAS_DISCOUNT_DIVISOR,
      ENQUEUE_GAS_COST
    )

    const batches = await Factory__ChainStorageContainer.deploy(
      AddressManager.address,
      'CanonicalTransactionChain'
    )
    await Factory__ChainStorageContainer.deploy(
      AddressManager.address,
      'CanonicalTransactionChain'
    )

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
    beforeEach(() => {
      CanonicalTransactionChain = CanonicalTransactionChain.connect(sequencer)
    })

    it('200 transactions in a single context', async () => {
      console.log(`Benchmark: 200 transactions in a single context.`)
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

      console.log('Benchmark complete.')

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
      console.log(`Benchmark: 200 transactions in 200 contexts.`)
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

      console.log('Benchmark complete.')

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
    let enqueueL2GasPrepaid
    let data
    beforeEach(async () => {
      CanonicalTransactionChain = CanonicalTransactionChain.connect(sequencer)
      enqueueL2GasPrepaid =
        await CanonicalTransactionChain.enqueueL2GasPrepaid()
      data = '0x' + '12'.repeat(1234)
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

      console.log('Benchmark complete.')

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

      console.log('Benchmark complete.')

      expectApprox(gasUsed, 134_100, {
        absoluteUpperDeviation: 500,
        // Assert a lower bound of 1% reduction on gas cost. If your tests are breaking because your
        // contracts are too efficient, consider updating the target value!
        percentLowerDeviation: 1,
      })
    })
  })
})
