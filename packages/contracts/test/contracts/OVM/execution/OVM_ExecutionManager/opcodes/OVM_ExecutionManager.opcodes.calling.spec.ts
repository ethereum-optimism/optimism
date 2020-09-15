import { expect } from '../../../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract, ContractFactory, Signer } from 'ethers'

/* Internal Imports */
import { getProxyManager, MockContract, getMockContract, DUMMY_ACCOUNTS, setProxyTarget, ZERO_ADDRESS, fromHexString, toHexString, makeHexString, NULL_BYTES32, DUMMY_BYTES32, encodeRevertData, REVERT_FLAGS, NON_ZERO_ADDRESS, GAS_LIMIT } from '../../../../../helpers'

describe('OVM_ExecutionManager:opcodes:calling', () => {
  let signer: Signer
  before(async () => {
    ;[signer] = await ethers.getSigners()
  })

  let Proxy_Manager: Contract
  before(async () => {
    Proxy_Manager = await getProxyManager()
  })

  let Mock__OVM_StateManager: MockContract
  before(async () => {
    Mock__OVM_StateManager = await getMockContract('OVM_StateManager')

    Mock__OVM_StateManager.setReturnValues('getAccount', (address: string) => {
      return [
        {
          ...DUMMY_ACCOUNTS[0].data,
          ethAddress: address
        }
      ]
    })

    await setProxyTarget(
      Proxy_Manager,
      'OVM_StateManager',
      Mock__OVM_StateManager
    )
  })

  let Factory__OVM_ExecutionManager: ContractFactory
  before(async () => {
    Factory__OVM_ExecutionManager = await ethers.getContractFactory(
      'OVM_ExecutionManager'
    )
  })

  let OVM_ExecutionManager: Contract
  beforeEach(async () => {
    OVM_ExecutionManager = await Factory__OVM_ExecutionManager.deploy(
      Proxy_Manager.address
    )
  })

  let Helper_CallTarget: Contract
  let Helper_RevertDataViewer: Contract
  beforeEach(async () => {
    const Factory__Helper_CallTarget = await ethers.getContractFactory(
      'Helper_CallTarget'
    )
    const Factory__Helper_RevertDataViewer = await ethers.getContractFactory(
      'Helper_RevertDataViewer'
    )

    Helper_CallTarget = await Factory__Helper_CallTarget.deploy()
    Helper_RevertDataViewer = await Factory__Helper_RevertDataViewer.deploy(
      Helper_CallTarget.address
    )
  })

  describe('ovmCALL', () => {
    describe('when the OVM_StateManager has the corresponding account', () => {
      before(() => {
        Mock__OVM_StateManager.setReturnValues('hasAccount', [true])
      })

      describe('when the OVM_StateManager has already loaded the account', () => {
        before(() => {
          Mock__OVM_StateManager.setReturnValues('testAndSetAccountLoaded', [true])
        })

        describe('when the call does not revert', () => {
          it('should return the result provided by the target contract', async () => {
            const returnData = makeHexString('1234', 32)

            expect(
              await OVM_ExecutionManager.callStatic.ovmCALL(
                GAS_LIMIT,
                Helper_CallTarget.address,
                Helper_CallTarget.interface.encodeFunctionData(
                  'doReturn',
                  [returnData]
                )
              )
            ).to.deep.equal([
              true,
              returnData
            ])
          })

          it('should set the ovmADDRESS to the target address', async () => {
            expect(
              await OVM_ExecutionManager.callStatic.ovmCALL(
                GAS_LIMIT,
                Helper_CallTarget.address,
                Helper_CallTarget.interface.encodeFunctionData(
                  'doReturnADDRESS',
                )
              )
            ).to.deep.equal([
              true,
              ethers.utils.defaultAbiCoder.encode(
                ['address'],
                [Helper_CallTarget.address]
              )
            ])
          })
        })

        describe('when the call does revert', () => {
          describe('with no data', () => {
            it('should return false with no data', async () => {
              expect(
                await OVM_ExecutionManager.callStatic.ovmCALL(
                  GAS_LIMIT,
                  Helper_CallTarget.address,
                  Helper_CallTarget.interface.encodeFunctionData(
                    'doRevert',
                    ['0x']
                  )
                )
              ).to.deep.equal([
                false,
                '0x'
              ])
            })
          })

          describe('with the INTENTIONAL_REVERT flag', () => {
            it('should return false with the flag and user-provided data', async () => {
            })
          })

          describe('with the EXCEEDS_NUISANCE_GAS flag', () => {
            it('should return false with the flag', async () => {

            })
          })

          describe('with the INVALID_STATE_ACCESS flag', () => {
            it('should revert with the INVALID_STATE_ACCESS flag', () => {

            })
          })

          describe('with the UNSAFE_BYTECODE flag', () => {
            it('should return false with the flag and no data', async () => {

            })
          })
        })
      })

      describe('when the OVM_StateManager has not already loaded the account', () => {
        before(() => {
          Mock__OVM_StateManager.setReturnValues('testAndSetAccountLoaded', [false])
        })

        describe('when the call parent does not contain enough nuisance gas', () => {
          it('should revert with the EXCEEDS_NUISANCE_GAS flag', () => {

          })
        })
      })
    })

    describe('when the OVM_StateManager does not have the corresponding account', () => {
      before(() => {
        Mock__OVM_StateManager.setReturnValues('hasAccount', [false])
      })

      it('should revert with the INVALID_STATE_ACCESS flag', () => {

      })
    })
  })

  describe('ovmSTATICCALL', () => {
    describe('when the OVM_StateManager has the corresponding account', () => {
      before(() => {
        Mock__OVM_StateManager.setReturnValues('hasAccount', [true])
      })

      describe('when the OVM_StateManager has already loaded the account', () => {
        before(() => {
          Mock__OVM_StateManager.setReturnValues('testAndSetAccountLoaded', [true])
        })

        describe('when the call does not revert', () => {
          it('should return the result provided by the target contract', async () => {

          })

          it('should set the context to static', async () => {

          })
        })

        describe('when the call does revert', () => {
          describe('with no data', () => {
            it('should return false with no data', async () => {

            })
          })

          describe('with the INTENTIONAL_REVERT flag', () => {
            it('should return false with the flag and user-provided data', async () => {

            })
          })

          describe('with the EXCEEDS_NUISANCE_GAS flag', () => {
            it('should return false with the flag', async () => {

            })
          })

          describe('with the INVALID_STATE_ACCESS flag', () => {
            it('should revert with the INVALID_STATE_ACCESS flag', () => {

            })
          })

          describe('with the UNSAFE_BYTECODE flag', () => {
            it('should return false with the flag and no data', async () => {

            })
          })
        })
      })

      describe('when the OVM_StateManager has not already loaded the account', () => {
        before(() => {
          Mock__OVM_StateManager.setReturnValues('testAndSetAccountLoaded', [false])
        })
        
        describe('when the call parent does not contain enough nuisance gas', () => {
          it('should revert with the EXCEEDS_NUISANCE_GAS flag', () => {

          })
        })
      })
    })

    describe('when the OVM_StateManager does not have the corresponding account', () => {
      before(() => {
        Mock__OVM_StateManager.setReturnValues('hasAccount', [false])
      })

      it('should revert with the INVALID_STATE_ACCESS flag', () => {
        
      })
    })
  })

  describe('ovmDELEGATECALL', () => {
    describe('when the OVM_StateManager has the corresponding account', () => {
      before(() => {
        Mock__OVM_StateManager.setReturnValues('hasAccount', [true])
      })

      describe('when the OVM_StateManager has already loaded the account', () => {
        before(() => {
          Mock__OVM_StateManager.setReturnValues('testAndSetAccountLoaded', [true])
        })
      
        describe('when the call does not revert', () => {
          it('should return the result provided by the target contract', async () => {

          })

          it('should retain the previous ovmADDRESS', async () => {

          })
        })

        describe('when the call does revert', () => {
          describe('with no data', () => {
            it('should return false with no data', async () => {

            })
          })

          describe('with the INTENTIONAL_REVERT flag', () => {
            it('should return false with the flag and user-provided data', async () => {

            })
          })

          describe('with the EXCEEDS_NUISANCE_GAS flag', () => {
            it('should return false with the flag', async () => {

            })
          })

          describe('with the INVALID_STATE_ACCESS flag', () => {
            it('should revert with the INVALID_STATE_ACCESS flag', () => {

            })
          })

          describe('with the UNSAFE_BYTECODE flag', () => {
            it('should return false with the flag and no data', async () => {

            })
          })
        })
      })

      describe('when the OVM_StateManager has not already loaded the account', () => {
        before(() => {
          Mock__OVM_StateManager.setReturnValues('testAndSetAccountLoaded', [false])
        })

        describe('when the call parent does not contain enough nuisance gas', () => {
          it('should revert with the EXCEEDS_NUISANCE_GAS flag', () => {

          })
        })
      })
    })

    describe('when the OVM_StateManager does not have the corresponding account', () => {
      before(() => {
        Mock__OVM_StateManager.setReturnValues('hasAccount', [false])
      })

      it('should revert with the INVALID_STATE_ACCESS flag', () => {
        
      })
    })
  })
})
