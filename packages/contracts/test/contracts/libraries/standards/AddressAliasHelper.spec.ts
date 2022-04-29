import { Contract } from 'ethers'
import { applyL1ToL2Alias, undoL1ToL2Alias } from '@eth-optimism/core-utils'

import { expect } from '../../../setup'
import { deploy } from '../../../helpers'

describe('AddressAliasHelper', () => {
  let AddressAliasHelper: Contract
  before(async () => {
    AddressAliasHelper = await deploy('TestLib_AddressAliasHelper')
  })

  describe('applyL1ToL2Alias', () => {
    it('should be able to apply the alias to a valid address', async () => {
      expect(
        await AddressAliasHelper.applyL1ToL2Alias(
          '0x0000000000000000000000000000000000000000'
        )
      ).to.equal(applyL1ToL2Alias('0x0000000000000000000000000000000000000000'))
    })

    it('should be able to apply the alias even if the operation overflows', async () => {
      expect(
        await AddressAliasHelper.applyL1ToL2Alias(
          '0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF'
        )
      ).to.equal(applyL1ToL2Alias('0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF'))
    })
  })

  describe('undoL1ToL2Alias', () => {
    it('should be able to undo the alias from a valid address', async () => {
      expect(
        await AddressAliasHelper.undoL1ToL2Alias(
          '0x1111000000000000000000000000000000001111'
        )
      ).to.equal(undoL1ToL2Alias('0x1111000000000000000000000000000000001111'))
    })

    it('should be able to undo the alias even if the operation underflows', async () => {
      expect(
        await AddressAliasHelper.undoL1ToL2Alias(
          '0x1111000000000000000000000000000000001110'
        )
      ).to.equal(undoL1ToL2Alias('0x1111000000000000000000000000000000001110'))
    })
  })
})
