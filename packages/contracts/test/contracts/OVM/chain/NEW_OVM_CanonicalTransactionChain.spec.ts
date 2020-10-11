import { expect } from '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Signer, ContractFactory, Contract } from 'ethers'
import { smockit, MockContract } from '@eth-optimism/smock'

/* Internal Imports */
import {
  makeAddressManager,
  setProxyTarget,
  FORCE_INCLUSION_PERIOD_SECONDS,
  // getEthTime,
  // setEthTime,
  // NON_NULL_BYTES32,
  // ZERO_ADDRESS,
} from '../../../helpers'

interface sequencerBatchContext {
  numSequencedTransactions: Number
  numSubsequentQueueTransactions: Number
  timestamp: Number
  blocknumber: Number
}

describe('NEW_OVM_CanonicalTransactionChain', () => {
  let signer: Signer
  before(async () => {
    ;[signer] = await ethers.getSigners()
  })

  let AddressManager: Contract
  before(async () => {
    AddressManager = await makeAddressManager()
    await AddressManager.setAddress('OVM_Sequencer', await signer.getAddress())
  })

  let Mock__OVM_L1ToL2TransactionQueue: MockContract
  before(async () => {
    Mock__OVM_L1ToL2TransactionQueue = smockit(
      await ethers.getContractFactory('OVM_L1ToL2TransactionQueue')
    )

    await setProxyTarget(
      AddressManager,
      'OVM_L1ToL2TransactionQueue',
      Mock__OVM_L1ToL2TransactionQueue
    )
  })

  let Factory__OVM_CanonicalTransactionChain: ContractFactory
  before(async () => {
    Factory__OVM_CanonicalTransactionChain = await ethers.getContractFactory(
      'NEW_OVM_CanonicalTransactionChain'
    )
  })

  let OVM_CanonicalTransactionChain: Contract
  beforeEach(async () => {
    OVM_CanonicalTransactionChain = await Factory__OVM_CanonicalTransactionChain.deploy(
      AddressManager.address,
      FORCE_INCLUSION_PERIOD_SECONDS
    )
  })

  describe('appendSequencerMultiBatch', () => {
    before(() => {
      Mock__OVM_L1ToL2TransactionQueue.smocked.size.will.return.with(0)
      Mock__OVM_L1ToL2TransactionQueue.smocked.dequeue.will.return()
    })
    it('should append a multi-batch with just one batch', async () =>{
      // Try out appending 
      const testBatchContext: sequencerBatchContext = {
        numSequencedTransactions: 1,
        numSubsequentQueueTransactions: 0,
        timestamp: 0,
        blocknumber: 0
      }

     await OVM_CanonicalTransactionChain.appendSequencerMultiBatch(
        ['0x1212'],
        [testBatchContext],
        0,
        1
    )
    })
  })
})
