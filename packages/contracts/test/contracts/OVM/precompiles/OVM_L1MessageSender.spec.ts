import { expect } from '../../../setup'

/* External Imports */
import { ethers } from 'hardhat'
import { ContractFactory, Contract } from 'ethers'
import { MockContract, smockit } from '@eth-optimism/smock'
import { NON_ZERO_ADDRESS } from '../../../helpers/constants'

const callPredeployStatic = async (
  Helper_PredeployCaller: Contract,
  predeploy: Contract,
  functionName: string,
  functionParams?: any[]
): Promise<any> => {
  return Helper_PredeployCaller.callStatic[functionName](
    predeploy.address,
    predeploy.interface.encodeFunctionData(functionName, functionParams || [])
  )
}

describe('OVM_L1MessageSender', () => {
  let Mock__OVM_ExecutionManager: MockContract
  before(async () => {
    Mock__OVM_ExecutionManager = await smockit(
      await ethers.getContractFactory('OVM_ExecutionManager')
    )
  })

  let Helper_PredeployCaller: Contract
  before(async () => {
    Helper_PredeployCaller = await (
      await ethers.getContractFactory('Helper_PredeployCaller')
    ).deploy()

    Helper_PredeployCaller.setTarget(Mock__OVM_ExecutionManager.address)
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
        await callPredeployStatic(
          Helper_PredeployCaller,
          OVM_L1MessageSender,
          'getL1MessageSender'
        )
      ).to.equal(NON_ZERO_ADDRESS)
    })
  })
})
