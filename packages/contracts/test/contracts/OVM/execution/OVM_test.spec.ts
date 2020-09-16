import { expect } from '../../../setup'

/* External Imports */
import bre, { ethers } from '@nomiclabs/buidler'
import { Contract } from 'ethers'

/* Internal Imports */
import { getModifiableStorageFactory } from '../../../helpers/storage/contract-storage'
import { runExecutionManagerTest } from '../../../helpers/test-parsing/parse-tests'
import { NON_NULL_BYTES32, GAS_LIMIT } from '../../../helpers'

const getCodeSize = async (address: string): Promise<number> => {
  const code = await ethers.provider.getCode(address)
  return (code.length - 2) / 2
}

describe('OVM_StateManager', () => {
  let OVM_ExecutionManager: Contract
  let OVM_StateManager: Contract
  let Helper_CodeContractForCalls: Contract
  before(async () => {
    const Factory__OVM_ExecutionManager = await getModifiableStorageFactory(
      'OVM_ExecutionManager'
    )
    const Factory__OVM_StateManager = await getModifiableStorageFactory(
      'OVM_StateManager'
    )
    const Factory__Helper_CodeContractForCalls = await getModifiableStorageFactory(
      'Helper_CodeContractForCalls'
    )

    OVM_ExecutionManager = await Factory__OVM_ExecutionManager.deploy(
      '0x5a0b54d5dc17e0aadc383d2db43b0a0d3e029c4c'
    )
    OVM_StateManager = await Factory__OVM_StateManager.deploy()
    Helper_CodeContractForCalls = await Factory__Helper_CodeContractForCalls.deploy()
  })

  const DUMMY_OVM_ADDRESS_1 = '0x' + '12'.repeat(20)
  const DUMMY_OVM_ADDRESS_2 = '0x' + '21'.repeat(20)

  describe('the test suite', () => {
    it('does the test suite', async () => {
      runExecutionManagerTest(
        {
          name: 'Top level test',
          preState: {
            ExecutionManager: {
              ovmStateManager: OVM_StateManager.address,
              messageRecord: {
                nuisanceGasLeft: GAS_LIMIT / 2
              }
            },
            StateManager: {
              accounts: {
                [DUMMY_OVM_ADDRESS_1]: {
                  codeHash: NON_NULL_BYTES32,
                  ethAddress: Helper_CodeContractForCalls.address
                },
                [DUMMY_OVM_ADDRESS_2]: {
                  codeHash: NON_NULL_BYTES32,
                  ethAddress: Helper_CodeContractForCalls.address
                }
              }
            }
          },
          postState: {
            ExecutionManager: {
              messageRecord: {
                nuisanceGasLeft: GAS_LIMIT / 2 - (await getCodeSize(Helper_CodeContractForCalls.address)) * 100 * 2
              }
            }
          },
          parameters: [
            {
              name: 'Do an ovmCALL',
              parameters: [
                {
                  steps: [
                    {
                      functionName: 'ovmCALL',
                      functionParams: [
                        GAS_LIMIT / 2,
                        DUMMY_OVM_ADDRESS_1,
                        [
                          {
                            functionName: 'ovmCALL',
                            functionParams: [
                              GAS_LIMIT / 2,
                              DUMMY_OVM_ADDRESS_2,
                              [
                                {
                                  functionName: 'ovmADDRESS',
                                  functionParams: [],
                                  returnStatus: true,
                                  returnValues: [DUMMY_OVM_ADDRESS_2]
                                },
                                {
                                  functionName: 'ovmCALLER',
                                  functionParams: [],
                                  returnStatus: true,
                                  returnValues: [DUMMY_OVM_ADDRESS_1]
                                }
                              ]
                            ],
                            returnStatus: true,
                            returnValues: []
                          },
                        ]
                      ],
                      returnStatus: true,
                      returnValues: []
                    }
                  ]
                }
              ]
            }
          ]
        },
        OVM_ExecutionManager,
        OVM_StateManager
      )
    })
  })
})
