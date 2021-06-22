/* External Imports */
import { ethers } from 'hardhat'
import { Signer, ContractFactory, Contract } from 'ethers'
import { smockit, MockContract } from '@eth-optimism/smock'
import {
  AppendSequencerBatchParams,
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
  FORCE_INCLUSION_PERIOD_BLOCKS,
  getEthTime,
  getNextBlockNumber,
} from '../../../helpers'

import { predeploys } from '../../../../src'

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
  let Mock__OVM_ExecutionManager: MockContract
  let Mock__OVM_StateCommitmentChain: MockContract
  before(async () => {
    AddressManager = await makeAddressManager()
    await AddressManager.setAddress(
      'OVM_Sequencer',
      await sequencer.getAddress()
    )
    await AddressManager.setAddress(
      'OVM_DecompressionPrecompileAddress',
      predeploys.OVM_SequencerEntrypoint
    )

    Mock__OVM_ExecutionManager = await smockit(
      await ethers.getContractFactory('OVM_ExecutionManager')
    )

    Mock__OVM_StateCommitmentChain = await smockit(
      await ethers.getContractFactory('OVM_StateCommitmentChain')
    )

    await setProxyTarget(
      AddressManager,
      'OVM_ExecutionManager',
      Mock__OVM_ExecutionManager
    )

    await setProxyTarget(
      AddressManager,
      'OVM_StateCommitmentChain',
      Mock__OVM_StateCommitmentChain
    )

    Mock__OVM_ExecutionManager.smocked.getMaxTransactionGasLimit.will.return.with(
      MAX_GAS_LIMIT
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
    OVM_CanonicalTransactionChain = await Factory__OVM_CanonicalTransactionChain.deploy(
      AddressManager.address,
      FORCE_INCLUSION_PERIOD_SECONDS,
      FORCE_INCLUSION_PERIOD_BLOCKS,
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
      OVM_CanonicalTransactionChain = OVM_CanonicalTransactionChain.connect(
        sequencer
      )
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

      console.log('Benchmark complete.')
      console.log('Gas used:', receipt.gasUsed.toNumber())
      console.log('Fixed calldata cost:', fixedCalldataCost)
      console.log(
        'Non-calldata overhead gas cost per transaction:',
        (receipt.gasUsed.toNumber() - fixedCalldataCost) / numTxs
      )
    }).timeout(100000000)

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

      console.log('Benchmark complete.')
      console.log('Gas used:', receipt.gasUsed.toNumber())
      console.log('Fixed calldata cost:', fixedCalldataCost)
      console.log(
        'Non-calldata overhead gas cost per transaction:',
        (receipt.gasUsed.toNumber() - fixedCalldataCost) / numTxs
      )
    }).timeout(100000000)
  })
})
