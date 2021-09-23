/* External Imports */
import { ethers } from 'hardhat'
import { Signer, ContractFactory, Contract, constants } from 'ethers'
import { smockit, MockContract } from '@eth-optimism/smock'
import {
  AppendSequencerBatchParams,
  BatchContext,
  encodeAppendSequencerBatch,
} from '@eth-optimism/core-utils'
import { TransactionResponse } from '@ethersproject/abstract-provider'
import { keccak256 } from 'ethers/lib/utils'
import _ from 'lodash'

/* Internal Imports */
import {
  makeAddressManager,
  setProxyTarget,
  FORCE_INCLUSION_PERIOD_SECONDS,
  getEthTime,
  getNextBlockNumber,
  NON_ZERO_ADDRESS,
  expectApprox,
} from '../../../helpers'

// Still have some duplication from OVM_CanonicalTransactionChain.spec.ts, but it's so minimal that
// this is probably cleaner for now. Particularly since we're planning to move all of this out into
// core-utils soon anyway.
const MAX_GAS_LIMIT = 8_000_000

const appendSequencerBatch = async (
  OVM_CanonicalTransactionChain: Contract,
  batch: AppendSequencerBatchParams
): Promise<TransactionResponse> => {
  const methodId = keccak256(Buffer.from('appendSequencerBatch()')).slice(2, 10)
  const calldata = encodeAppendSequencerBatch(batch)
  return OVM_CanonicalTransactionChain.signer.sendTransaction({
    to: OVM_CanonicalTransactionChain.address,
    data: '0x' + methodId + calldata,
  })
}

describe('[GAS BENCHMARK] OVM_CanonicalTransactionChain', () => {
  let sequencer: Signer
  before(async () => {
    ;[sequencer] = await ethers.getSigners()
  })

  let AddressManager: Contract
  let Mock__OVM_StateCommitmentChain: MockContract
  before(async () => {
    AddressManager = await makeAddressManager()
    await AddressManager.setAddress(
      'OVM_Sequencer',
      await sequencer.getAddress()
    )

    Mock__OVM_StateCommitmentChain = await smockit(
      await ethers.getContractFactory('OVM_StateCommitmentChain')
    )

    await setProxyTarget(
      AddressManager,
      'OVM_StateCommitmentChain',
      Mock__OVM_StateCommitmentChain
    )
  })

  let Factory__OVM_CanonicalTransactionChain: ContractFactory
  let Factory__OVM_ChainStorageContainer: ContractFactory
  before(async () => {
    Factory__OVM_CanonicalTransactionChain = await ethers.getContractFactory(
      'OVM_CanonicalTransactionChain'
    )

    Factory__OVM_ChainStorageContainer = await ethers.getContractFactory(
      'OVM_ChainStorageContainer'
    )
  })

  let OVM_CanonicalTransactionChain: Contract
  beforeEach(async () => {
    // Use a larger FIP for blocks so that we can send a large number of
    // enqueue() transactions without having to manipulate the block number.
    const forceInclusionPeriodBlocks = 101
    OVM_CanonicalTransactionChain =
      await Factory__OVM_CanonicalTransactionChain.deploy(
        AddressManager.address,
        FORCE_INCLUSION_PERIOD_SECONDS,
        forceInclusionPeriodBlocks,
        MAX_GAS_LIMIT
      )

    const batches = await Factory__OVM_ChainStorageContainer.deploy(
      AddressManager.address,
      'OVM_CanonicalTransactionChain'
    )
    const queue = await Factory__OVM_ChainStorageContainer.deploy(
      AddressManager.address,
      'OVM_CanonicalTransactionChain'
    )

    await AddressManager.setAddress(
      'OVM_ChainStorageContainer-CTC-batches',
      batches.address
    )

    await AddressManager.setAddress(
      'OVM_ChainStorageContainer-CTC-queue',
      queue.address
    )

    await AddressManager.setAddress(
      'OVM_CanonicalTransactionChain',
      OVM_CanonicalTransactionChain.address
    )
  })

  describe('appendSequencerBatch [ @skip-on-coverage ]', () => {
    beforeEach(() => {
      OVM_CanonicalTransactionChain =
        OVM_CanonicalTransactionChain.connect(sequencer)
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

      const res = await appendSequencerBatch(OVM_CanonicalTransactionChain, {
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
      console.log('Gas used:', gasUsed)

      console.log('Fixed calldata cost:', fixedCalldataCost)
      console.log(
        'Non-calldata overhead gas cost per transaction:',
        (gasUsed - fixedCalldataCost) / numTxs
      )
      expectApprox(gasUsed, 1_605_971, { upperPercentDeviation: 0 })
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

      const res = await appendSequencerBatch(OVM_CanonicalTransactionChain, {
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
      console.log('Gas used:', gasUsed)

      console.log('Fixed calldata cost:', fixedCalldataCost)
      console.log(
        'Non-calldata overhead gas cost per transaction:',
        (gasUsed - fixedCalldataCost) / numTxs
      )
      expectApprox(gasUsed, 1_739_992, { upperPercentDeviation: 0 })
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
        await OVM_CanonicalTransactionChain.enqueue(target, gasLimit, data)

        queueContexts.push({
          blockNumber: (await getNextBlockNumber(ethers.provider)) - 1,
          timestamp: await getEthTime(ethers.provider),
          numSequencedTransactions: 1,
          numSubsequentQueueTransactions: 1,
        })
      }

      const fixedCalldataCost =
        (transactionTemplate.slice(2).length / 2) * 16 * numTxs

      const res = await appendSequencerBatch(OVM_CanonicalTransactionChain, {
        shouldStartAtElement: 0,
        totalElementsToAppend: numTxs + numEnqueues,
        contexts: queueContexts,
        transactions,
      })

      const receipt = await res.wait()
      const gasUsed = receipt.gasUsed.toNumber()

      console.log('Benchmark complete.')
      console.log('Gas used:', gasUsed)

      console.log('Fixed calldata cost:', fixedCalldataCost)
      console.log(
        'Non-calldata overhead gas cost per transaction:',
        (gasUsed - fixedCalldataCost) / numTxs
      )
      expectApprox(gasUsed, 1_125_554, { upperPercentDeviation: 0 })
    }).timeout(10_000_000)
  })

  describe('enqueue [ @skip-on-coverage ]', () => {
    let ENQUEUE_L2_GAS_PREPAID
    let data
    beforeEach(async () => {
      OVM_CanonicalTransactionChain =
        OVM_CanonicalTransactionChain.connect(sequencer)
      ENQUEUE_L2_GAS_PREPAID =
        await OVM_CanonicalTransactionChain.ENQUEUE_L2_GAS_PREPAID()
      data = '0x' + '12'.repeat(1234)
    })

    it('cost to enqueue a transaction above the prepaid threshold', async () => {
      const l2GasLimit = 2 * ENQUEUE_L2_GAS_PREPAID

      const res = await OVM_CanonicalTransactionChain.enqueue(
        NON_ZERO_ADDRESS,
        l2GasLimit,
        data
      )
      const receipt = await res.wait()
      const gasUsed = receipt.gasUsed.toNumber()

      console.log('Benchmark complete.')
      console.log('Gas used:', gasUsed)

      expectApprox(gasUsed, 217_789, { upperPercentDeviation: 0 })
    })

    it('cost to enqueue a transaction below the prepaid threshold', async () => {
      const l2GasLimit = ENQUEUE_L2_GAS_PREPAID - 1

      const res = await OVM_CanonicalTransactionChain.enqueue(
        NON_ZERO_ADDRESS,
        l2GasLimit,
        data
      )
      const receipt = await res.wait()
      const gasUsed = receipt.gasUsed.toNumber()

      console.log('Benchmark complete.')
      console.log('Gas used:', gasUsed)

      expectApprox(gasUsed, 156_885, { upperPercentDeviation: 0 })
    })
  })
})
