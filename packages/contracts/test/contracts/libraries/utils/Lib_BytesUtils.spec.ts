import { Contract } from 'ethers'

import { expect } from '../../../setup'
import { Lib_BytesUtils_TEST_JSON } from '../../../data'
import { deploy, runJsonTest } from '../../../helpers'

describe('Lib_BytesUtils', () => {
  describe('JSON tests', () => {
    runJsonTest('TestLib_BytesUtils', Lib_BytesUtils_TEST_JSON)
  })

  describe('Use of library with other memory-modifying operations', () => {
    let TestLib_BytesUtils: Contract
    before(async () => {
      TestLib_BytesUtils = await deploy('TestLib_BytesUtils')
    })

    it('should allow creation of a contract beforehand and still work', async () => {
      expect(
        await TestLib_BytesUtils.callStatic.sliceWithTaintedMemory(
          '0x123412341234',
          0,
          0
        )
      ).to.eq('0x')
    })
  })
})
