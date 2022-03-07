/* External Imports */
import { ethers } from 'hardhat'
import { Contract, Signer, ContractFactory } from 'ethers'

/* Internal Imports */
import { expect } from '../../../setup'
import { makeAddressManager, NON_NULL_BYTES32 } from '../../../helpers'

describe('ChainStorageContainer', () => {
  let sequencer: Signer
  let otherSigner: Signer
  let signer: Signer
  let signerAddress: string

  let AddressManager: Contract
  let Factory__ChainStorageContainer: ContractFactory
  before(async () => {
    ;[sequencer, otherSigner, signer] = await ethers.getSigners()
    signerAddress = await otherSigner.getAddress()

    AddressManager = await makeAddressManager()
    await AddressManager.setAddress(
      'OVM_Sequencer',
      await sequencer.getAddress()
    )

    Factory__ChainStorageContainer = await ethers.getContractFactory(
      'ChainStorageContainer'
    )
  })

  let ChainStorageContainer: Contract
  beforeEach(async () => {
    ChainStorageContainer = await Factory__ChainStorageContainer.connect(
      otherSigner
    ).deploy(AddressManager.address, signerAddress)

    await AddressManager.setAddress(
      'ChainStorageContainer',
      ChainStorageContainer.address
    )

    await AddressManager.setAddress(signerAddress, signerAddress)
  })

  describe('push', () => {
    for (const len of [1, 2, 4, 8, 32]) {
      it(`it should be able to add ${len} element(s) to the array`, async () => {
        for (let i = 0; i < len; i++) {
          await expect(
            ChainStorageContainer.connect(otherSigner)['push(bytes32)'](
              NON_NULL_BYTES32
            )
          ).to.not.be.reverted
        }
      })
    }
  })

  describe('setGlobalMetadata', () => {
    it('should modify the extra data', async () => {
      const globalMetaData = `0x${'11'.repeat(27)}`
      await ChainStorageContainer.connect(otherSigner).setGlobalMetadata(
        globalMetaData
      )

      expect(await ChainStorageContainer.getGlobalMetadata()).to.equal(
        globalMetaData
      )
    })
  })

  describe('deleteElementsAfterInclusive', () => {
    it('should revert when the array is empty', async () => {
      await expect(
        ChainStorageContainer.connect(otherSigner)[
          'deleteElementsAfterInclusive(uint256)'
        ](0)
      ).to.be.reverted
    })

    it('should revert when called by non-owner', async () => {
      await expect(
        ChainStorageContainer.connect(signer)[
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
            await ChainStorageContainer.connect(otherSigner)['push(bytes32)'](
              value
            )
          }
        })

        for (let i = len - 1; i > 0; i -= Math.max(1, len / 4)) {
          it(`should be able to delete everything after and including the ${i}th/st/rd/whatever element`, async () => {
            await expect(
              ChainStorageContainer.connect(otherSigner)[
                'deleteElementsAfterInclusive(uint256)'
              ](i)
            ).to.not.be.reverted

            expect(await ChainStorageContainer.length()).to.equal(i)
            await expect(ChainStorageContainer.get(i)).to.be.reverted
          })
        }
      })
    }
  })
})
