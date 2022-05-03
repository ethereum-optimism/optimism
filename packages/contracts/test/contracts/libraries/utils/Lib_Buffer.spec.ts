import { Contract, ethers } from 'ethers'

import { expect } from '../../../setup'
import { deploy } from '../../../helpers'

describe('Lib_Buffer', () => {
  let Lib_Buffer: Contract
  beforeEach(async () => {
    Lib_Buffer = await deploy('TestLib_Buffer')
  })

  describe('push(bytes32,bytes27)', () => {
    for (const len of [1, 2, 4, 8, 32]) {
      it(`it should be able to add ${len} element(s) to the array`, async () => {
        for (let i = 0; i < len; i++) {
          await expect(
            Lib_Buffer['push(bytes32,bytes27)'](
              ethers.utils.keccak256(`0x${i.toString(16).padStart(16, '0')}`),
              `0x${'00'.repeat(27)}`
            )
          ).to.not.be.reverted
        }
      })
    }
  })

  describe('push(bytes32)', () => {
    for (const len of [1, 2, 4, 8, 32]) {
      it(`it should be able to add ${len} element(s) to the array`, async () => {
        for (let i = 0; i < len; i++) {
          await expect(
            Lib_Buffer['push(bytes32)'](
              ethers.utils.keccak256(`0x${i.toString(16).padStart(16, '0')}`)
            )
          ).to.not.be.reverted
        }
      })
    }
  })

  describe('get', () => {
    for (const len of [1, 2, 4, 8, 32]) {
      describe(`when the array has ${len} element(s)`, () => {
        const values = []
        beforeEach(async () => {
          for (let i = 0; i < len; i++) {
            const value = ethers.utils.keccak256(
              `0x${i.toString(16).padStart(16, '0')}`
            )
            values.push(value)
            await Lib_Buffer['push(bytes32,bytes27)'](
              value,
              `0x${'00'.repeat(27)}`
            )
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
        const values = []
        beforeEach(async () => {
          for (let i = 0; i < len; i++) {
            const value = ethers.utils.keccak256(
              `0x${i.toString(16).padStart(16, '0')}`
            )
            values.push(value)
            await Lib_Buffer['push(bytes32,bytes27)'](
              value,
              `0x${'00'.repeat(27)}`
            )
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
      await Lib_Buffer['push(bytes32,bytes27)'](
        ethers.utils.keccak256('0x00'),
        extraData
      )

      expect(await Lib_Buffer.getExtraData()).to.equal(extraData)
    })

    it('should change if set multiple times', async () => {
      await Lib_Buffer['push(bytes32,bytes27)'](
        ethers.utils.keccak256('0x00'),
        `0x${'11'.repeat(27)}`
      )

      const extraData = `0x${'22'.repeat(27)}`

      await Lib_Buffer['push(bytes32,bytes27)'](
        ethers.utils.keccak256('0x00'),
        extraData
      )

      expect(await Lib_Buffer.getExtraData()).to.equal(extraData)
    })
  })

  describe('setExtraData', () => {
    it('should modify the extra data', async () => {
      const extraData = `0x${'11'.repeat(27)}`
      await Lib_Buffer.setExtraData(extraData)

      expect(await Lib_Buffer.getExtraData()).to.equal(extraData)
    })

    it('should be able to modify the extra data multiple times', async () => {
      const extraData1 = `0x${'22'.repeat(27)}`
      await Lib_Buffer.setExtraData(extraData1)
      expect(await Lib_Buffer.getExtraData()).to.equal(extraData1)

      const extraData2 = `0x${'11'.repeat(27)}`
      await Lib_Buffer.setExtraData(extraData2)

      expect(await Lib_Buffer.getExtraData()).to.equal(extraData2)
    })
  })

  describe('deleteElementsAfterInclusive', () => {
    it('should revert when the array is empty', async () => {
      await expect(Lib_Buffer['deleteElementsAfterInclusive(uint40)'](0)).to.be
        .reverted
    })

    for (const len of [1, 2, 4, 8, 32]) {
      describe(`when the array has ${len} element(s)`, () => {
        const values = []
        beforeEach(async () => {
          for (let i = 0; i < len; i++) {
            const value = ethers.utils.keccak256(
              `0x${i.toString(16).padStart(16, '0')}`
            )
            values.push(value)
            await Lib_Buffer['push(bytes32,bytes27)'](
              value,
              `0x${'00'.repeat(27)}`
            )
          }
        })

        for (let i = len - 1; i > 0; i -= Math.max(1, len / 4)) {
          it(`should be able to delete everything after and including the ${i}th/st/rd/whatever element`, async () => {
            await expect(Lib_Buffer['deleteElementsAfterInclusive(uint40)'](i))
              .to.not.be.reverted

            expect(await Lib_Buffer.getLength()).to.equal(i)
            await expect(Lib_Buffer.get(i)).to.be.reverted
          })
        }

        for (let i = len - 1; i > 0; i -= Math.max(1, len / 4)) {
          it(`should be able to delete after and incl. ${i}th/st/rd/whatever element while changing extra data`, async () => {
            const extraData = `0x${i.toString(16).padStart(54, '0')}`
            await expect(
              Lib_Buffer['deleteElementsAfterInclusive(uint40,bytes27)'](
                i,
                extraData
              )
            ).to.not.be.reverted

            expect(await Lib_Buffer.getLength()).to.equal(i)
            await expect(Lib_Buffer.get(i)).to.be.reverted
            expect(await Lib_Buffer.getExtraData()).to.equal(extraData)
          })
        }
      })
    }
  })

  describe('setContext', () => {
    it('should modify the context', async () => {
      const length = 20
      const extraData = `0x${'11'.repeat(27)}`
      const cntx = [length, extraData]

      await Lib_Buffer.setContext(length, extraData)

      expect(await Lib_Buffer.getContext()).to.eql(cntx)
    })

    it('should not modify the context', async () => {
      const length = 0
      const extraData = `0x${'00'.repeat(27)}`

      const prevContext = await Lib_Buffer.getContext()
      await Lib_Buffer.setContext(length, extraData)

      expect(await Lib_Buffer.getContext()).to.eql(prevContext)
    })
  })
})
