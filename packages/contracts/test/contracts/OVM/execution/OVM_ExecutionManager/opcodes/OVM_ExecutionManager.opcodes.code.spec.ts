import { expect } from '../../../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract, ContractFactory } from 'ethers'

/* Internal Imports */
import { getProxyManager, MockContract, getMockContract, DUMMY_ACCOUNTS, setProxyTarget, ZERO_ADDRESS, fromHexString, toHexString, makeHexString, NULL_BYTES32, DUMMY_BYTES32, encodeRevertData, REVERT_FLAGS } from '../../../../../helpers'

describe('OVM_ExecutionManager:opcodes:code', () => {
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

  let Dummy_Contract: Contract
  before(async () => {
    // We need some contract to query code for, might as well reuse an existing object.
    Dummy_Contract = Mock__OVM_StateManager
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

  let Helper_RevertDataViewer: Contract
  beforeEach(async () => {
    const Factory__Helper_RevertDataViewer = await ethers.getContractFactory(
      'Helper_RevertDataViewer'
    )

    Helper_RevertDataViewer = await Factory__Helper_RevertDataViewer.deploy(
      OVM_ExecutionManager.address
    )
  })

  describe('ovmEXTCODECOPY()', () => {
    describe('when the OVM_StateManager has the corresponding account', () => {
      before(() => {
        Mock__OVM_StateManager.setReturnValues('hasAccount', [true])
      })

      describe('when the OVM_StateManager has already loaded the corresponding account', () => {
        before(() => {
          Mock__OVM_StateManager.setReturnValues('testAndSetAccountLoaded', [true])
        })

        it('should return the code for a given account', async () => {
          const expectedCode = await ethers.provider.getCode(Dummy_Contract.address)
          const expectedCodeSize = fromHexString(expectedCode).length

          expect(
            await OVM_ExecutionManager.callStatic.ovmEXTCODECOPY(Dummy_Contract.address, 0, expectedCodeSize)
          ).to.equal(expectedCode)
        })

        it('should return empty if the provided length is zero', async () => {
          const expectedCode = '0x'

          expect(
            await OVM_ExecutionManager.callStatic.ovmEXTCODECOPY(Dummy_Contract.address, 0, 0)
          ).to.equal(expectedCode)
        })

        it('should return offset code when offset is less than total length', async () => {
          const fullCode = await ethers.provider.getCode(Dummy_Contract.address)
          const fullCodeSize = fromHexString(fullCode).length

          const codeOffset = Math.floor(fullCodeSize / 2)
          const codeLength = fullCodeSize - codeOffset
          const expectedCode = toHexString(fromHexString(fullCode).slice(codeOffset, codeOffset + codeLength))

          expect(
            await OVM_ExecutionManager.callStatic.ovmEXTCODECOPY(Dummy_Contract.address, codeOffset, codeLength)
          ).to.equal(expectedCode)
        })

        it('should return less code when length is less than total length', async () => {
          const fullCode = await ethers.provider.getCode(Dummy_Contract.address)
          const fullCodeSize = fromHexString(fullCode).length

          const codeLength = Math.floor(fullCodeSize / 2)
          const expectedCode = toHexString(fromHexString(fullCode).slice(0, codeLength))

          expect(
            await OVM_ExecutionManager.callStatic.ovmEXTCODECOPY(Dummy_Contract.address, 0, codeLength)
          ).to.equal(expectedCode)
        })

        it('should return extra code when length is greater than total length', async () => {
          const fullCode = await ethers.provider.getCode(Dummy_Contract.address)
          const fullCodeSize = fromHexString(fullCode).length

          const extraLength = fullCodeSize
          const codeLength = fullCodeSize + extraLength
          const expectedCode = toHexString(Buffer.concat([
            fromHexString(fullCode),
            fromHexString(makeHexString('00', extraLength))
          ]))

          expect(
            await OVM_ExecutionManager.callStatic.ovmEXTCODECOPY(Dummy_Contract.address, 0, codeLength)
          ).to.equal(expectedCode)
        })

        it('should return extra code when offset is less than total length and length is greater than total length', async () => {
          const fullCode = await ethers.provider.getCode(Dummy_Contract.address)
          const fullCodeSize = fromHexString(fullCode).length

          const extraLength = fullCodeSize
          const codeOffset = Math.floor(fullCodeSize / 2)
          const codeLength = fullCodeSize - codeOffset + extraLength
          const expectedCode = toHexString(Buffer.concat([
            fromHexString(fullCode).slice(codeOffset, codeOffset + codeLength),
            fromHexString(makeHexString('00', extraLength))
          ]))

          expect(
            await OVM_ExecutionManager.callStatic.ovmEXTCODECOPY(Dummy_Contract.address, codeOffset, codeLength)
          ).to.equal(expectedCode)
        })

        it('should return empty bytes when both offset and length exceed total length', async () => {
          const fullCode = await ethers.provider.getCode(Dummy_Contract.address)
          const fullCodeSize = fromHexString(fullCode).length

          const extraLength = fullCodeSize
          const codeOffset = fullCodeSize
          const codeLength = fullCodeSize + extraLength
          const expectedCode = toHexString(Buffer.concat([
            fromHexString(makeHexString('00', codeLength))
          ]))

          expect(
            await OVM_ExecutionManager.callStatic.ovmEXTCODECOPY(Dummy_Contract.address, codeOffset, codeLength)
          ).to.equal(expectedCode)
        })
      })

      describe('when the OVM_StateManager has not already loaded the corresponding account', () => {
        before(() => {
          Mock__OVM_StateManager.setReturnValues('testAndSetAccountLoaded', [false])
        })

        it('should revert with the EXCEEDS_NUISANCE_GAS flag', async () => {
          const calldata = OVM_ExecutionManager.interface.encodeFunctionData(
            'ovmEXTCODECOPY',
            [
              Dummy_Contract.address,
              0,
              0
            ]
          )

          await Helper_RevertDataViewer.fallback({
            data: calldata
          })
    
          expect(
            await Helper_RevertDataViewer.revertdata()  
          ).to.equal(encodeRevertData(
            REVERT_FLAGS.EXCEEDS_NUISANCE_GAS
          ))
        })
      })
    })

    describe('when the OVM_StateManager does not have the corresponding account', () => {
      before(() => {
        Mock__OVM_StateManager.setReturnValues('hasAccount', [false])
      })

      it('should revert with the INVALID_STATE_ACCESS flag', async () => {
        const calldata = OVM_ExecutionManager.interface.encodeFunctionData(
          'ovmEXTCODECOPY',
          [
            Dummy_Contract.address,
            0,
            0
          ]
        )
  
        await Helper_RevertDataViewer.fallback({
          data: calldata
        })
  
        expect(
          await Helper_RevertDataViewer.revertdata()  
        ).to.equal(encodeRevertData(
          REVERT_FLAGS.INVALID_STATE_ACCESS
        ))
      })
    })
  })

  describe('ovmEXTCODESIZE()', () => {
    describe('when the OVM_StateManager has the corresponding account', () => {
      before(() => {
        Mock__OVM_StateManager.setReturnValues('hasAccount', [true])
      })

      describe('when the OVM_StateManager has already loaded the corresponding account', () => {
        before(() => {
          Mock__OVM_StateManager.setReturnValues('testAndSetAccountLoaded', [true])
        })
  
        it('should return the code size for a given account', async () => {
          const expectedCode = await ethers.provider.getCode(Dummy_Contract.address)
          const expectedCodeSize = fromHexString(expectedCode).length

          expect(
            await OVM_ExecutionManager.callStatic.ovmEXTCODESIZE(Dummy_Contract.address)
          ).to.equal(expectedCodeSize)
        })

        it('should return zero if the account has no code', async () => {
          const expectedCodeSize = 0
          
          expect(
            await OVM_ExecutionManager.callStatic.ovmEXTCODESIZE(ZERO_ADDRESS)
          ).to.equal(expectedCodeSize)
        })
      })

      describe('when the OVM_StateManager has not already loaded the corresponding account', () => {
        before(() => {
          Mock__OVM_StateManager.setReturnValues('testAndSetAccountLoaded', [false])
        })

        it('should revert with the EXCEEDS_NUISANCE_GAS flag', async () => {
          const calldata = OVM_ExecutionManager.interface.encodeFunctionData(
            'ovmEXTCODESIZE',
            [
              Dummy_Contract.address,
            ]
          )

          await Helper_RevertDataViewer.fallback({
            data: calldata
          })
    
          expect(
            await Helper_RevertDataViewer.revertdata()  
          ).to.equal(encodeRevertData(
            REVERT_FLAGS.EXCEEDS_NUISANCE_GAS
          ))
        })
      })
    })

    describe('when the OVM_StateManager does not have the corresponding account', () => {
      before(() => {
        Mock__OVM_StateManager.setReturnValues('hasAccount', [false])
      })

      it('should revert with the INVALID_STATE_ACCESS flag', async () => {
        const calldata = OVM_ExecutionManager.interface.encodeFunctionData(
          'ovmEXTCODESIZE',
          [
            Dummy_Contract.address
          ]
        )
  
        await Helper_RevertDataViewer.fallback({
          data: calldata
        })
  
        expect(
          await Helper_RevertDataViewer.revertdata()  
        ).to.equal(encodeRevertData(
          REVERT_FLAGS.INVALID_STATE_ACCESS
        ))
      })
    })
  })

  describe('ovmEXTCODEHASH()', () => {
    describe('when the OVM_StateManager has the corresponding account', () => {
      before(() => {
        Mock__OVM_StateManager.setReturnValues('hasAccount', [true])
      })

      describe('when the OVM_StateManager has already loaded the corresponding account', () => {
        before(() => {
          Mock__OVM_StateManager.setReturnValues('testAndSetAccountLoaded', [true])
        })
  
        it('should return the code hash for a given account', async () => {
          const expectedCode = await ethers.provider.getCode(Dummy_Contract.address)
          const expectedCodeHash = ethers.utils.keccak256(expectedCode)

          expect(
            await OVM_ExecutionManager.callStatic.ovmEXTCODEHASH(Dummy_Contract.address)
          ).to.equal(expectedCodeHash)
        })

        it('should return zero if the account does not exist', async () => {
          const expectedCodeHash = NULL_BYTES32

          expect(
            await OVM_ExecutionManager.callStatic.ovmEXTCODEHASH(ZERO_ADDRESS)
          ).to.equal(expectedCodeHash)
        })
      })

      describe('when the OVM_StateManager has not already loaded the corresponding account', () => {
        before(() => {
          Mock__OVM_StateManager.setReturnValues('testAndSetAccountLoaded', [false])
        })

        it('should revert with the EXCEEDS_NUISANCE_GAS flag', async () => {
          const calldata = OVM_ExecutionManager.interface.encodeFunctionData(
            'ovmEXTCODEHASH',
            [
              Dummy_Contract.address,
            ]
          )

          await Helper_RevertDataViewer.fallback({
            data: calldata
          })
    
          expect(
            await Helper_RevertDataViewer.revertdata()  
          ).to.equal(encodeRevertData(
            REVERT_FLAGS.EXCEEDS_NUISANCE_GAS
          ))
        })
      })
    })

    describe('when the OVM_StateManager does not have the corresponding account', () => {
      before(() => {
        Mock__OVM_StateManager.setReturnValues('hasAccount', [false])
      })

      it('should revert with the INVALID_STATE_ACCESS flag', async () => {
        const calldata = OVM_ExecutionManager.interface.encodeFunctionData(
          'ovmEXTCODEHASH',
          [
            Dummy_Contract.address
          ]
        )
  
        await Helper_RevertDataViewer.fallback({
          data: calldata
        })
  
        expect(
          await Helper_RevertDataViewer.revertdata()  
        ).to.equal(encodeRevertData(
          REVERT_FLAGS.INVALID_STATE_ACCESS
        ))
      })
    })
  })
})
