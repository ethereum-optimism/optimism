import { expect } from '../../../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract, ContractFactory } from 'ethers'

/* Internal Imports */
import { getProxyManager, MockContract, getMockContract, DUMMY_ACCOUNTS, setProxyTarget, ZERO_ADDRESS, fromHexString, toHexString, makeHexString, NULL_BYTES32, DUMMY_BYTES32, encodeRevertData, REVERT_FLAGS, NON_ZERO_ADDRESS } from '../../../../../helpers'


describe('OVM_ExecutionManager:opcodes:storage', () => {
  let Proxy_Manager: Contract
  before(async () => {
    Proxy_Manager = await getProxyManager()
  })

  let Mock__OVM_StateManager: MockContract
  before(async () => {
    Mock__OVM_StateManager = await getMockContract('OVM_StateManager')

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

  let Helper_RevertDataViewer: Contract
  beforeEach(async () => {
    const Factory__Helper_RevertDataViewer = await ethers.getContractFactory(
      'Helper_RevertDataViewer'
    )

    Helper_RevertDataViewer = await Factory__Helper_RevertDataViewer.deploy(
      OVM_ExecutionManager.address
    )
  })

  const DUMMY_SLOT_KEY = DUMMY_BYTES32[0]
  const DUMMY_SLOT_VALUE = DUMMY_BYTES32[1]

  describe('ovmSLOAD', () => {
    before(() => {
      Mock__OVM_StateManager.setReturnValues('getContractStorage', () => {
        return [
          DUMMY_SLOT_VALUE
        ]
      })
    })

    describe('when the OVM_StateManager has the corresponding storage slot', () => {
      before(() => {
        Mock__OVM_StateManager.setReturnValues('hasContractStorage', [true])
      })

      describe('when the OVM_StateManager has already loaded the storage slot', () => {
        before(() => {
          Mock__OVM_StateManager.setReturnValues('testAndSetContractStorageLoaded', [true])
        })

        it('should return the value of the storage slot', async () => {
          expect(
            await OVM_ExecutionManager.callStatic.ovmSLOAD(
              DUMMY_SLOT_KEY
            )
          ).to.equal(DUMMY_SLOT_VALUE)
        })
      })

      describe('when the OVM_StateManager has not already loaded the storage slot', () => {
        before(() => {
          Mock__OVM_StateManager.setReturnValues('testAndSetContractStorageLoaded', [false])
        })

        it('should revert with the EXCEEDS_NUISANCE_GAS flag', async () => {
          const calldata = OVM_ExecutionManager.interface.encodeFunctionData(
            'ovmSLOAD',
            [
              DUMMY_SLOT_KEY
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

    describe('when the OVM_StateManager does not have the corresponding storage slot', () => {
      before(() => {
        Mock__OVM_StateManager.setReturnValues('hasContractStorage', [false])
      })

      it('should revert with the INVALID_STATE_ACCESS flag', async () => {
        const calldata = OVM_ExecutionManager.interface.encodeFunctionData(
          'ovmSLOAD',
          [
            DUMMY_SLOT_KEY
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

  describe('ovmSSTORE', () => {
    describe('when the OVM_StateManager has already changed the storage slot', () => {
      before(() => {
        Mock__OVM_StateManager.setReturnValues('testAndSetContractStorageChanged', [true])
      })

      it('should modify the storage slot value', async () => {
        await expect(
          OVM_ExecutionManager.ovmSSTORE(
            DUMMY_SLOT_KEY,
            DUMMY_SLOT_VALUE
          )
        ).to.not.be.reverted

        expect(
          Mock__OVM_StateManager.getCallData('putContractStorage', 0)
        ).to.deep.equal(
          [
            ZERO_ADDRESS,
            DUMMY_SLOT_KEY,
            DUMMY_SLOT_VALUE
          ]
        )
      })
    })

    describe('when the OVM_StateManager has not already changed the storage slot', () => {
      before(() => {
        Mock__OVM_StateManager.setReturnValues('testAndSetContractStorageChanged', [false])
      })

      it('should revert with the EXCEEDS_NUISANCE_GAS flag', async () => {
        const calldata = OVM_ExecutionManager.interface.encodeFunctionData(
          'ovmSSTORE',
          [
            DUMMY_SLOT_KEY,
            DUMMY_SLOT_VALUE
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
})
