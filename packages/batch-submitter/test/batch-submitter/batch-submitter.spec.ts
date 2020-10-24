import {expect} from '../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getContractInterface } from '@eth-optimism/contracts'
import { remove0x } from '@eth-optimism/core-utils'
import { smockit, MockContract } from '@eth-optimism/smock'
import { Signer, ContractFactory, Contract, BigNumber } from 'ethers'

/* Internal Imports */
import { BatchSubmitter } from '../../src/batch-submitter'
import { Signature } from '../../src'
import { MockchainProvider } from './mockchain-provider'
import {
  makeAddressManager,
  setProxyTarget,
  FORCE_INCLUSION_PERIOD_SECONDS,
  getContractFactory,
} from '../helpers'
import { CanonicalTransactionChainContract, QueueOrigin, TxType, ctcCoder } from '../../src'

const DECOMPRESSION_ADDRESS = '0x4200000000000000000000000000000000000008'
const MAX_GAS_LIMIT = 8_000_000
const MAX_TX_SIZE = 100_000

// Helper functions
interface QueueElement {
  queueRoot: string
  timestamp: number
  blockNumber: number
}
const getNextQueueElement = async (ctcContract: Contract): Promise<QueueElement> => {
  const nextQueueIndex = await ctcContract.getNextQueueIndex()
  const nextQueueElement = await ctcContract.getQueueElement(nextQueueIndex)
  return nextQueueElement
}
const DUMMY_SIG: Signature = {
  r: '11'.repeat(32),
  s: '22'.repeat(32),
  v: '01'
}


describe('BatchSubmitter', () => {
  let signer: Signer
  let sequencer: Signer
  let l2Provider: MockchainProvider
  before(async () => {
    ;[signer, sequencer] = await ethers.getSigners()
    l2Provider = new MockchainProvider()
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
  beforeEach(async () => {
    const unwrapped_OVM_CanonicalTransactionChain = await Factory__OVM_CanonicalTransactionChain.deploy(
      AddressManager.address,
      FORCE_INCLUSION_PERIOD_SECONDS
    )
    OVM_CanonicalTransactionChain = new CanonicalTransactionChainContract(
      unwrapped_OVM_CanonicalTransactionChain.address,
      getContractInterface('OVM_CanonicalTransactionChain'),
      sequencer
    )
  })

  describe.only('Submit', () => {
    let enqueuedElements: {blockNumber: number, timestamp: number}[] = []

    beforeEach(async () => {
      for (let i = 1; i < 15; i++) {
        await OVM_CanonicalTransactionChain.enqueue(
          '0x' + '01'.repeat(20),
          50_000,
          '0x' + i.toString().repeat(64),
          {
            gasLimit: 1_000_000
          }
        )
      }
    })

    it('should execute without reverting', async () => {
      const batchSubmitter = new BatchSubmitter(
        OVM_CanonicalTransactionChain,
        sequencer,
        l2Provider as any,
        l2Provider.chainId(),
        MAX_TX_SIZE,
        10,
        1
      )
      l2Provider.setNumBlocksToReturn(5)
      const nextQueueElement = await getNextQueueElement(OVM_CanonicalTransactionChain)
      const data = ctcCoder.createEOATxData.encode({
            sig: DUMMY_SIG,
            messageHash: '66'.repeat(32)
      })
      l2Provider.setL2BlockData(
        {
          data,
          meta: {
            l1BlockNumber: nextQueueElement.blockNumber - 1,
            txType: TxType.createEOA,
            queueOrigin: QueueOrigin.Sequencer,
            l1TxOrigin: '0x' + '12'.repeat(20),
          }
        } as any,
        nextQueueElement.timestamp - 1,
      )
      let receipt = await batchSubmitter.submitNextBatch()
      let logData = remove0x(receipt.logs[0].data)
      expect(parseInt(logData.slice(64*0, 64*1))).to.equal(0) // _startingQueueIndex
      expect(parseInt(logData.slice(64*1, 64*2))).to.equal(0) // _numQueueElements
      expect(parseInt(logData.slice(64*2, 64*3))).to.equal(5) // _totalElements
      receipt = await batchSubmitter.submitNextBatch()
      logData = remove0x(receipt.logs[0].data)
      expect(parseInt(logData.slice(64*0, 64*1), 16)).to.equal(0) // _startingQueueIndex
      expect(parseInt(logData.slice(64*1, 64*2), 16)).to.equal(0) // _numQueueElements
      expect(parseInt(logData.slice(64*2, 64*3), 16)).to.equal(10) // _totalElements
    })
  })
})
