import { ethers } from 'hardhat'
import { Contract } from 'ethers'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'

import { expect } from '../../../setup'
import { deploy, NON_NULL_BYTES32 } from '../../../helpers'

describe('ChainStorageContainer', () => {
  let signer1: SignerWithAddress
  let signer2: SignerWithAddress
  before(async () => {
    ;[signer1, signer2] = await ethers.getSigners()
  })

  let AddressManager: Contract
  let ChainStorageContainer: Contract
  beforeEach(async () => {
    AddressManager = await deploy('Lib_AddressManager')
    ChainStorageContainer = await deploy('ChainStorageContainer', {
      signer: signer1,
      args: [AddressManager.address, signer1.address],
    })

    // ChainStorageContainer uses name resolution to check the owner address.
    await AddressManager.setAddress(signer1.address, signer1.address)
  })

  describe('push', () => {
    for (const len of [1, 2, 4, 8, 32]) {
      it(`it should be able to add ${len} element(s) to the array`, async () => {
        for (let i = 0; i < len; i++) {
          await expect(ChainStorageContainer['push(bytes32)'](NON_NULL_BYTES32))
            .to.not.be.reverted
        }
      })
    }
  })

  describe('setGlobalMetadata', () => {
    it('should modify the extra data', async () => {
      const globalMetaData = `0x${'11'.repeat(27)}`
      await ChainStorageContainer.setGlobalMetadata(globalMetaData)

      expect(await ChainStorageContainer.getGlobalMetadata()).to.equal(
        globalMetaData
      )
    })
  })

  describe('deleteElementsAfterInclusive', () => {
    it('should revert when the array is empty', async () => {
      await expect(
        ChainStorageContainer['deleteElementsAfterInclusive(uint256)'](0)
      ).to.be.reverted
    })

    it('should revert when called by non-owner', async () => {
      await expect(
        ChainStorageContainer.connect(signer2)[
          'deleteElementsAfterInclusive(uint256)'
        ](0)
      ).to.be.revertedWith(
        'ChainStorageContainer: Function can only be called by the owner.'
      )
    })

    for (const len of [1, 2, 4, 8, 32]) {
      describe(`when the array has ${len} element(s)`, () => {
        const values = []
        beforeEach(async () => {
          for (let i = 0; i < len; i++) {
            const value = NON_NULL_BYTES32
            values.push(value)
            await ChainStorageContainer['push(bytes32)'](value)
          }
        })

        for (let i = len - 1; i > 0; i -= Math.max(1, len / 4)) {
          it(`should be able to delete everything after and including the ${i}th/st/rd/whatever element`, async () => {
            await expect(
              ChainStorageContainer['deleteElementsAfterInclusive(uint256)'](i)
            ).to.not.be.reverted

            expect(await ChainStorageContainer.length()).to.equal(i)
            await expect(ChainStorageContainer.get(i)).to.be.reverted
          })
        }
      })
    }
  })
})
