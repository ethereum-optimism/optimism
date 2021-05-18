import { expect } from '../../../setup'

import hre from 'hardhat'
import { Contract, ethers } from 'ethers'

describe('Lib_Buffer', () => {
  let Lib_Buffer: Contract
  beforeEach(async () => {
    const Factory__Lib_Buffer = await hre.ethers.getContractFactory(
      'TestLib_Buffer'
    )
    Lib_Buffer = await Factory__Lib_Buffer.deploy()
  })

  describe('push', () => {
    for (const len of [1, 2, 4, 8, 32]) {
      it(`it should be able to add ${len} element(s) to the array`, async () => {
        for (let i = 0; i < len; i++) {
          await expect(
            Lib_Buffer.push(
              ethers.utils.keccak256(`0x${i.toString(16).padStart(16, '0')}`),
              `0x${'00'.repeat(27)}`
            )
          ).to.not.be.reverted
        }
      })
    }
  })

  describe('get', () => {
    for (const len of [1, 2, 4, 8, 32]) {
      describe(`when the array has ${len} element(s)`, () => {
        let values = []
        beforeEach(async () => {
          for (let i = 0; i < len; i++) {
            const value = ethers.utils.keccak256(
              `0x${i.toString(16).padStart(16, '0')}`
            )
            values.push(value)
            await Lib_Buffer.push(value, `0x${'00'.repeat(27)}`)
          }
        })

        for (let i = 0; i < len; i += Math.max(1, len / 4)) {
          it(`should be able to get the ${i}th/st/rd/whatever value`, async () => {
            expect(await Lib_Buffer.get(i)).to.equal(values[i])
          })
        }

        it('should throw if attempting to access an element that does not exist', async () => {
          await expect(Lib_Buffer.get(len + 1)).to.be.reverted
        })
      })
    }
  })

  describe('getLength', () => {
    it('should return zero by default', async () => {
      expect(await Lib_Buffer.getLength()).to.equal(0)
    })

    for (const len of [1, 2, 4, 8, 32]) {
      describe(`when the array has ${len} element(s)`, () => {
        let values = []
        beforeEach(async () => {
          for (let i = 0; i < len; i++) {
            const value = ethers.utils.keccak256(
              `0x${i.toString(16).padStart(16, '0')}`
            )
            values.push(value)
            await Lib_Buffer.push(value, `0x${'00'.repeat(27)}`)
          }
        })

        it(`should return a value of ${len}`, async () => {
          expect(await Lib_Buffer.getLength()).to.equal(len)
        })
      })
    }
  })

  describe('getExtraData', () => {
    it('should be bytes27(0) by default', async () => {
      expect(await Lib_Buffer.getExtraData()).to.equal(`0x${'00'.repeat(27)}`)
    })

    it('should change if set by a call to push()', async () => {
      const extraData = `0x${'11'.repeat(27)}`
      await Lib_Buffer.push(ethers.utils.keccak256('0x00'), extraData)

      expect(await Lib_Buffer.getExtraData()).to.equal(extraData)
    })

    it('should change if set multiple times', async () => {
      await Lib_Buffer.push(
        ethers.utils.keccak256('0x00'),
        `0x${'11'.repeat(27)}`
      )

      const extraData = `0x${'22'.repeat(27)}`

      await Lib_Buffer.push(ethers.utils.keccak256('0x00'), extraData)

      expect(await Lib_Buffer.getExtraData()).to.equal(extraData)
    })
  })

  describe('deleteElementsAfterInclusive', () => {
    it('should revert when the array is empty', async () => {
      await expect(
        Lib_Buffer.deleteElementsAfterInclusive(0, `0x${'00'.repeat(27)}`)
      ).to.be.reverted
    })

    for (const len of [1, 2, 4, 8, 32]) {
      describe(`when the array has ${len} element(s)`, () => {
        let values = []
        beforeEach(async () => {
          for (let i = 0; i < len; i++) {
            const value = ethers.utils.keccak256(
              `0x${i.toString(16).padStart(16, '0')}`
            )
            values.push(value)
            await Lib_Buffer.push(value, `0x${'00'.repeat(27)}`)
          }
        })

        for (let i = len - 1; i > 0; i -= Math.max(1, len / 4)) {
          it(`should be able to delete everything after and including the ${i}th/st/rd/whatever element`, async () => {
            await expect(
              Lib_Buffer.deleteElementsAfterInclusive(i, `0x${'00'.repeat(27)}`)
            ).to.not.be.reverted

            expect(await Lib_Buffer.getLength()).to.equal(i)

            await expect(Lib_Buffer.get(i)).to.be.reverted
          })
        }

        it(`should be able to modify the extra data`, async () => {
          const extraData = `0x${'11'.repeat(27)}`
          await Lib_Buffer.deleteElementsAfterInclusive(
            Math.floor(len / 2),
            extraData
          )

          expect(await Lib_Buffer.getExtraData()).to.equal(extraData)
        })
      })
    }
  })
})
