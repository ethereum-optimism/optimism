import { Contract } from 'ethers'

import { expect } from '../../../../setup'
import { deploy } from '../../../../helpers'

describe('CheckTrue', () => {
  let CheckTrue: Contract
  before(async () => {
    CheckTrue = await deploy('CheckTrue')
  })

  describe('check', () => {
    it('should return true', async () => {
      expect(await CheckTrue.check('0x')).to.equal(true)
    })
  })
})
