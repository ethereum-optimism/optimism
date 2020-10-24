import '../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getContractInterface } from '@eth-optimism/contracts'
import { smockit, MockContract } from '@eth-optimism/smock'
import { Signer, ContractFactory, Contract, BigNumber } from 'ethers'

/* Internal Imports */
import { BatchSubmitter } from '../../src/batch-submitter'
import { MockchainProvider } from './mockchain-provider'
import {
  makeAddressManager,
  setProxyTarget,
  FORCE_INCLUSION_PERIOD_SECONDS,
  getContractFactory,
} from '../helpers'
import { CanonicalTransactionChainContract } from '../../src'

const DECOMPRESSION_ADDRESS = '0x4200000000000000000000000000000000000008'
const MAX_GAS_LIMIT = 8_000_000
const MAX_TX_SIZE = 100_000

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

  describe('Submit', () => {
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
      await batchSubmitter.submitNextBatch()
    })
  })
})
