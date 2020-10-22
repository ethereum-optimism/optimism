import '../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { RollupDeployConfig, deploy } from '@eth-optimism/contracts'
import { smockit, MockContract } from '@eth-optimism/smock'
import { Signer, ContractFactory, Contract, BigNumber } from 'ethers'

/* Internal Imports */
import { BatchSubmitter } from '../../src/batch-submitter'
import {
  makeAddressManager,
  setProxyTarget,
  FORCE_INCLUSION_PERIOD_SECONDS,
  setEthTime,
  NON_ZERO_ADDRESS,
  remove0x,
  getEthTime,
  getNextBlockNumber,
  increaseEthTime,
  ZERO_ADDRESS,
  getContractFactory
} from '../helpers'

const DECOMPRESSION_ADDRESS = '0x4200000000000000000000000000000000000008'
const MAX_GAS_LIMIT = 8_000_000

describe('BatchSubmitter', () => {
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
      await getContractFactory('OVM_ExecutionManager') as any
    )

    Mock__OVM_StateCommitmentChain = smockit(
      await getContractFactory('OVM_StateCommitmentChain') as any
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

  let OVM_CanonicalTransactionChain: Contract
  beforeEach(async () => {
    OVM_CanonicalTransactionChain = await Factory__OVM_CanonicalTransactionChain.deploy(
      AddressManager.address,
      FORCE_INCLUSION_PERIOD_SECONDS
    )
  })

  describe('Submit', () => {
    it('should execute without reverting', async () => {
      const batchSubmitter = new BatchSubmitter(OVM_CanonicalTransactionChain.address, sequencer, sequencer.provider)
      await batchSubmitter.submitNextBatch()
    })

    it('should allow me to query a bunch of blocks', async () => {
      for (let i = 1; i < 10; i++) {
        console.log('enquing!')
        await OVM_CanonicalTransactionChain.enqueue(
          '0x' + '01'.repeat(20),
          50_000,
          '0x' + i.toString().repeat(64),
          {
            gasLimit: 1_000_000
          }
        )
        console.log('done!')
      }
      const totalBlocks = await (await signer.provider.getBlockNumber())
      for (let i = 0; i < totalBlocks; i++) {
        console.log(await signer.provider.getBlockWithTransactions(i))
      }
    })
  })
})
