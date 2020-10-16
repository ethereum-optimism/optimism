import { expect } from '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { ContractFactory, Contract } from 'ethers'
import { MockContract, smockit } from '@eth-optimism/smock'
import { NON_ZERO_ADDRESS } from '../../../helpers/constants'

const callPrecompileStatic = async (
  Helper_PrecompileCaller: Contract,
  precompile: Contract,
  functionName: string,
  functionParams?: any[]
): Promise<any> => {
  return Helper_PrecompileCaller.callStatic[functionName](
    precompile.address,
    precompile.interface.encodeFunctionData(functionName, functionParams || [])
  )
}

describe('OVM_L1MessageSender', () => {
  let Mock__OVM_ExecutionManager: MockContract
  before(async () => {
    Mock__OVM_ExecutionManager = smockit(
      await ethers.getContractFactory('OVM_ExecutionManager')
    )
  })

  let Helper_PrecompileCaller: Contract
  before(async () => {
    Helper_PrecompileCaller = await (
      await ethers.getContractFactory('Helper_PrecompileCaller')
    ).deploy()

    Helper_PrecompileCaller.setTarget(Mock__OVM_ExecutionManager.address)
  })

  let Factory__OVM_L1MessageSender: ContractFactory
  before(async () => {
    Factory__OVM_L1MessageSender = await ethers.getContractFactory(
      'OVM_L1MessageSender'
    )
  })

  let OVM_L1MessageSender: Contract
  beforeEach(async () => {
    OVM_L1MessageSender = await Factory__OVM_L1MessageSender.deploy()
  })

  describe('getL1MessageSender', () => {
    before(async () => {
      Mock__OVM_ExecutionManager.smocked.ovmL1TXORIGIN.will.return.with(
        NON_ZERO_ADDRESS
      )
    })

    it('should return the L1 message sender', async () => {
      expect(
        await callPrecompileStatic(
          Helper_PrecompileCaller,
          OVM_L1MessageSender,
          'getL1MessageSender'
        )
      ).to.equal(NON_ZERO_ADDRESS)
    })
  })
})
