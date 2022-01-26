/* Internal Imports */
import { ethers } from 'hardhat'
import { Contract } from 'ethers'

/* External Imports */
import { Lib_BytesUtils_TEST_JSON } from '../../../data'
import { runJsonTest } from '../../../helpers'
import { expect } from '../../../setup'

describe('Lib_BytesUtils', () => {
  describe('JSON tests', () => {
    runJsonTest('TestLib_BytesUtils', Lib_BytesUtils_TEST_JSON)
  })

  describe('Use of library with other memory-modifying operations', () => {
    let TestLib_BytesUtils: Contract
    before(async () => {
      TestLib_BytesUtils = await (
        await ethers.getContractFactory('TestLib_BytesUtils')
      ).deploy()
    })

    it('should allow creation of a contract beforehand and still work', async () => {
      const slice = await TestLib_BytesUtils.callStatic.sliceWithTaintedMemory(
        '0x123412341234',
        0,
        0
      )
      expect(slice).to.eq('0x')
    })
  })
})
