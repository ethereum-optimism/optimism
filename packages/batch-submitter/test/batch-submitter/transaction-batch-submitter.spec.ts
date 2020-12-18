import { expect } from '../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Signer, ContractFactory, Contract } from 'ethers'
import { getContractInterface } from '@eth-optimism/contracts'
import { smockit, MockContract } from '@eth-optimism/smock'
import { remove0x, getLogger } from '@eth-optimism/core-utils'

/* Internal Imports */
import { MockchainProvider } from './mockchain-provider'
import {
  makeAddressManager,
  setProxyTarget,
  FORCE_INCLUSION_PERIOD_SECONDS,
  getContractFactory,
} from '../helpers'
import {
  CanonicalTransactionChainContract,
  QueueOrigin,
  TxType,
  ctcCoder,
  TransactionBatchSubmitter,
  Signature,
  TX_BATCH_SUBMITTER_LOG_TAG,
} from '../../src'

const DECOMPRESSION_ADDRESS = '0x4200000000000000000000000000000000000008'
const MAX_GAS_LIMIT = 8_000_000
const MAX_TX_SIZE = 100_000
const MIN_TX_SIZE = 1_000

// Helper functions
interface QueueElement {
  queueRoot: string
  timestamp: number
  blockNumber: number
}
const getQueueElement = async (
  ctcContract: Contract,
  nextQueueIndex?: number
): Promise<QueueElement> => {
  if (!nextQueueIndex) {
    nextQueueIndex = await ctcContract.getNextQueueIndex()
  }
  const nextQueueElement = await ctcContract.getQueueElement(nextQueueIndex)
  return nextQueueElement
}
const DUMMY_SIG: Signature = {
  r: '11'.repeat(32),
  s: '22'.repeat(32),
  v: '01',
}

describe('TransactionBatchSubmitter', () => {
  let signer: Signer
  let sequencer: Signer
  before(async () => {
    ;[signer, sequencer] = await ethers.getSigners()
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
      DECOMPRESSION_ADDRESS
    )

    Mock__OVM_ExecutionManager = smockit(
      (await getContractFactory('OVM_ExecutionManager')) as any
    )

    Mock__OVM_StateCommitmentChain = smockit(
      (await getContractFactory('OVM_StateCommitmentChain')) as any
    )

    await setProxyTarget(
      AddressManager,
      'OVM_ExecutionManager',
      Mock__OVM_ExecutionManager as any
    )

    await setProxyTarget(
      AddressManager,
      'OVM_StateCommitmentChain',
      Mock__OVM_StateCommitmentChain as any
    )

    Mock__OVM_StateCommitmentChain.smocked.canOverwrite.will.return.with(false)
    Mock__OVM_ExecutionManager.smocked.getMaxTransactionGasLimit.will.return.with(
      MAX_GAS_LIMIT
    )
  })

  let Factory__OVM_CanonicalTransactionChain: ContractFactory
  before(async () => {
    Factory__OVM_CanonicalTransactionChain = await getContractFactory(
      'OVM_CanonicalTransactionChain'
    )
  })

  let OVM_CanonicalTransactionChain: CanonicalTransactionChainContract
  let l2Provider: MockchainProvider
  beforeEach(async () => {
    const unwrapped_OVM_CanonicalTransactionChain = await Factory__OVM_CanonicalTransactionChain.deploy(
      AddressManager.address,
      FORCE_INCLUSION_PERIOD_SECONDS
    )
    await unwrapped_OVM_CanonicalTransactionChain.init()

    OVM_CanonicalTransactionChain = new CanonicalTransactionChainContract(
      unwrapped_OVM_CanonicalTransactionChain.address,
      getContractInterface('OVM_CanonicalTransactionChain'),
      sequencer
    )
    l2Provider = new MockchainProvider(
      OVM_CanonicalTransactionChain.address,
      '0x' + '00'.repeat(20)
    )
  })

  describe('Submit', () => {
    const enqueuedElements: Array<{
      blockNumber: number
      timestamp: number
    }> = []

    let batchSubmitter
    beforeEach(async () => {
      for (let i = 1; i < 15; i++) {
        await OVM_CanonicalTransactionChain.enqueue(
          '0x' + '01'.repeat(20),
          50_000,
          '0x' + i.toString().repeat(64),
          {
            gasLimit: 1_000_000,
          }
        )
      }
      batchSubmitter = new TransactionBatchSubmitter(
        sequencer,
        l2Provider as any,
        MIN_TX_SIZE,
        MAX_TX_SIZE,
        10,
        1,
        1,
        false,
        getLogger(TX_BATCH_SUBMITTER_LOG_TAG),
        false
      )
    })

    it('should submit a sequencer batch correctly', async () => {
      l2Provider.setNumBlocksToReturn(5)
      const nextQueueElement = await getQueueElement(
        OVM_CanonicalTransactionChain
      )
      const data = ctcCoder.createEOATxData.encode({
        sig: DUMMY_SIG,
        messageHash: '66'.repeat(32),
      })
      l2Provider.setL2BlockData(
        {
          data,
          l1BlockNumber: nextQueueElement.blockNumber - 1,
          txType: TxType.createEOA,
          queueOrigin: QueueOrigin.Sequencer,
          l1TxOrigin: '0x' + '12'.repeat(20),
        } as any,
        nextQueueElement.timestamp - 1
      )
      let receipt = await batchSubmitter.submitNextBatch()
      let logData = remove0x(receipt.logs[1].data)
      expect(parseInt(logData.slice(64 * 0, 64 * 1), 16)).to.equal(0) // _startingQueueIndex
      expect(parseInt(logData.slice(64 * 1, 64 * 2), 16)).to.equal(0) // _numQueueElements
      expect(parseInt(logData.slice(64 * 2, 64 * 3), 16)).to.equal(6) // _totalElements
      receipt = await batchSubmitter.submitNextBatch()
      logData = remove0x(receipt.logs[1].data)
      expect(parseInt(logData.slice(64 * 0, 64 * 1), 16)).to.equal(0) // _startingQueueIndex
      expect(parseInt(logData.slice(64 * 1, 64 * 2), 16)).to.equal(0) // _numQueueElements
      expect(parseInt(logData.slice(64 * 2, 64 * 3), 16)).to.equal(11) // _totalElements
    })

    it('should submit a queue batch correctly', async () => {
      l2Provider.setNumBlocksToReturn(5)
      l2Provider.setL2BlockData({
        queueOrigin: QueueOrigin.L1ToL2,
      } as any)
      let receipt = await batchSubmitter.submitNextBatch()
      let logData = remove0x(receipt.logs[1].data)
      expect(parseInt(logData.slice(64 * 0, 64 * 1), 16)).to.equal(0) // _startingQueueIndex
      expect(parseInt(logData.slice(64 * 1, 64 * 2), 16)).to.equal(6) // _numQueueElements
      expect(parseInt(logData.slice(64 * 2, 64 * 3), 16)).to.equal(6) // _totalElements
      receipt = await batchSubmitter.submitNextBatch()
      logData = remove0x(receipt.logs[1].data)
      expect(parseInt(logData.slice(64 * 0, 64 * 1), 16)).to.equal(6) // _startingQueueIndex
      expect(parseInt(logData.slice(64 * 1, 64 * 2), 16)).to.equal(5) // _numQueueElements
      expect(parseInt(logData.slice(64 * 2, 64 * 3), 16)).to.equal(11) // _totalElements
    })

    it('should submit a batch with both queue and sequencer chain elements', async () => {
      l2Provider.setNumBlocksToReturn(10) // For this batch we'll return 10 elements!
      l2Provider.setL2BlockData({
        queueOrigin: QueueOrigin.L1ToL2,
      } as any)
      // Turn blocks 3-5 into sequencer txs
      const nextQueueElement = await getQueueElement(
        OVM_CanonicalTransactionChain,
        2
      )
      const data = ctcCoder.createEOATxData.encode({
        sig: DUMMY_SIG,
        messageHash: '66'.repeat(32),
      })
      l2Provider.setL2BlockData(
        {
          data,
          l1BlockNumber: nextQueueElement.blockNumber - 1,
          txType: TxType.createEOA,
          queueOrigin: QueueOrigin.Sequencer,
          l1TxOrigin: '0x' + '12'.repeat(20),
        } as any,
        nextQueueElement.timestamp - 1,
        3,
        6
      )
      const receipt = await batchSubmitter.submitNextBatch()
      const logData = remove0x(receipt.logs[1].data)
      expect(parseInt(logData.slice(64 * 0, 64 * 1), 16)).to.equal(0) // _startingQueueIndex
      expect(parseInt(logData.slice(64 * 1, 64 * 2), 16)).to.equal(8) // _numQueueElements
      expect(parseInt(logData.slice(64 * 2, 64 * 3), 16)).to.equal(11) // _totalElements
    })
  })
})
