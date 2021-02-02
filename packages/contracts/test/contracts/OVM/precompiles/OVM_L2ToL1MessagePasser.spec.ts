import { expect } from '../../../setup'

/* External Imports */
import { ethers } from 'hardhat'
import { ContractFactory, Contract } from 'ethers'
import { MockContract, smockit } from '@eth-optimism/smock'
import { NON_ZERO_ADDRESS } from '../../../helpers/constants'
import { keccak256 } from 'ethers/lib/utils'
import { remove0x } from '../../../helpers'

const ELEMENT_TEST_SIZES = [1, 2, 4, 8, 16]

const callPrecompile = async (
  Helper_PrecompileCaller: Contract,
  precompile: Contract,
  functionName: string,
  functionParams?: any[]
): Promise<any> => {
  return Helper_PrecompileCaller.callPrecompile(
    precompile.address,
    precompile.interface.encodeFunctionData(functionName, functionParams || [])
  )
}

describe('OVM_L2ToL1MessagePasser', () => {
  let Mock__OVM_ExecutionManager: MockContract
  before(async () => {
    Mock__OVM_ExecutionManager = await smockit(
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

  let Factory__OVM_L2ToL1MessagePasser: ContractFactory
  before(async () => {
    Factory__OVM_L2ToL1MessagePasser = await ethers.getContractFactory(
      'OVM_L2ToL1MessagePasser'
    )
  })

  let OVM_L2ToL1MessagePasser: Contract
  beforeEach(async () => {
    OVM_L2ToL1MessagePasser = await Factory__OVM_L2ToL1MessagePasser.deploy()
  })

  describe('passMessageToL1', () => {
    before(async () => {
      Mock__OVM_ExecutionManager.smocked.ovmCALLER.will.return.with(
        NON_ZERO_ADDRESS
      )
    })

    for (const size of ELEMENT_TEST_SIZES) {
      it(`should be able to pass ${size} messages`, async () => {
        for (let i = 0; i < size; i++) {
          const message = '0x' + '12' + '34'.repeat(i)

          await callPrecompile(
            Helper_PrecompileCaller,
            OVM_L2ToL1MessagePasser,
            'passMessageToL1',
            [message]
          )

          expect(
            await OVM_L2ToL1MessagePasser.sentMessages(
              keccak256(message + remove0x(Helper_PrecompileCaller.address))
            )
          ).to.equal(true)
        }
      })
    }
  })
})
