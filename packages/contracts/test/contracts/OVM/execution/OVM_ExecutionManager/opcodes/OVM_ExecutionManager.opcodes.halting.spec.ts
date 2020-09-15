import { expect } from '../../../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract, ContractFactory } from 'ethers'

/* Internal Imports */
import { getProxyManager, encodeRevertData, REVERT_FLAGS } from '../../../../../helpers'

describe('OVM_ExecutionManager:opcodes:halting', () => {
  let Proxy_Manager: Contract
  before(async () => {
    Proxy_Manager = await getProxyManager()
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

    Helper_RevertDataViewer = await Factory__Helper_RevertDataViewer.deploy(OVM_ExecutionManager.address)
  })

  describe('ovmREVERT', () => {
    it('should revert with the provided data prefixed by the intentional revert flag', async () => {
      const revertdata = '12345678'.repeat(10)
      const calldata = OVM_ExecutionManager.interface.encodeFunctionData(
        'ovmREVERT',
        ['0x' + revertdata]
      )

      await Helper_RevertDataViewer.fallback({
        data: calldata
      })

      expect(
        await Helper_RevertDataViewer.revertdata()  
      ).to.equal(encodeRevertData(
        REVERT_FLAGS.INTENTIONAL_REVERT,
        '0x' + revertdata
      ))
    })

    it('should revert with the intentional revert flag if no data is provided', async () => {
      const calldata = OVM_ExecutionManager.interface.encodeFunctionData(
        'ovmREVERT',
        ['0x']
      )

      await Helper_RevertDataViewer.fallback({
        data: calldata
      })

      expect(
        await Helper_RevertDataViewer.revertdata()  
      ).to.equal(encodeRevertData(
        REVERT_FLAGS.INTENTIONAL_REVERT
      ))
    })
  })
})
