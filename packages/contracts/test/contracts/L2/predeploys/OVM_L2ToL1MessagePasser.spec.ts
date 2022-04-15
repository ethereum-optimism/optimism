/* External Imports */
import { ethers } from 'hardhat'
import { ContractFactory, Contract } from 'ethers'
import { smock, FakeContract } from '@defi-wonderland/smock'
import { remove0x } from '@eth-optimism/core-utils'
import { keccak256 } from 'ethers/lib/utils'

/* Internal Imports */
import { expect } from '../../../setup'
import { NON_ZERO_ADDRESS } from '../../../helpers/constants'

const ELEMENT_TEST_SIZES = [1, 2, 4, 8, 16]

const callPredeploy = async (
  Helper_PredeployCaller: Contract,
  predeploy: Contract,
  functionName: string,
  functionParams?: any[]
): Promise<any> => {
  return Helper_PredeployCaller.callPredeploy(
    predeploy.address,
    predeploy.interface.encodeFunctionData(functionName, functionParams || [])
  )
}

// TODO: rewrite this test to bypass the execution manager
describe.skip('OVM_L2ToL1MessagePasser', () => {
  let Fake__OVM_ExecutionManager: FakeContract
  before(async () => {
    Fake__OVM_ExecutionManager = await smock.fake<Contract>(
      'OVM_ExecutionManager'
    )
  })

  let Helper_PredeployCaller: Contract
  before(async () => {
    Helper_PredeployCaller = await (
      await ethers.getContractFactory('Helper_PredeployCaller')
    ).deploy()

    Helper_PredeployCaller.setTarget(Fake__OVM_ExecutionManager.address)
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
      Fake__OVM_ExecutionManager.ovmCALLER.returns(NON_ZERO_ADDRESS)
    })

    for (const size of ELEMENT_TEST_SIZES) {
      it(`should be able to pass ${size} messages`, async () => {
        for (let i = 0; i < size; i++) {
          const message = '0x' + '12' + '34'.repeat(i)

          await callPredeploy(
            Helper_PredeployCaller,
            OVM_L2ToL1MessagePasser,
            'passMessageToL1',
            [message]
          )

          expect(
            await OVM_L2ToL1MessagePasser.sentMessages(
              keccak256(message + remove0x(Helper_PredeployCaller.address))
            )
          ).to.equal(true)
        }
      })
    }
  })
})
