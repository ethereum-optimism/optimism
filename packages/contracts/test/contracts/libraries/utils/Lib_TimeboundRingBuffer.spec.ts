/* tslint:disable:no-empty */
import { expect } from '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract, Signer } from 'ethers'

/* Internal Imports */
import {
  NON_NULL_BYTES32,
  makeHexString,
  increaseEthTime
} from '../../../helpers'

const numToBytes32 = (num: Number): string => {
  if (num < 0 || num > 255) {
    throw new Error('Unsupported number.')
  }
  const strNum = (num < 16) ? '0' + num.toString(16) : num.toString(16)
  return '0x' + '00'.repeat(31) + strNum
}

describe('Lib_TimeboundRingBuffer', () => {
  let signer: Signer
  before(async () => {
    ;[signer] = await ethers.getSigners()
  })

  let Lib_TimeboundRingBuffer: Contract

  const NON_NULL_BYTES28 = makeHexString('01', 28)
  const pushNum = (num: Number) => Lib_TimeboundRingBuffer.push(numToBytes32(num), NON_NULL_BYTES28)
  const push2Nums = (num1: Number, num2: Number) => Lib_TimeboundRingBuffer.push2(numToBytes32(num1), numToBytes32(num2), NON_NULL_BYTES28)

  describe('push with no timeout', () => {
    beforeEach(async () => {
      Lib_TimeboundRingBuffer = await (
        await ethers.getContractFactory('TestLib_TimeboundRingBuffer')
      ).deploy(4, 1, 0)
      for (let i = 0; i < 4; i++) {
        await Lib_TimeboundRingBuffer.push(numToBytes32(i), NON_NULL_BYTES28)
      }
    })

    it('should push a single value which increases the length', async () => {
      expect(await Lib_TimeboundRingBuffer.getLength()).to.equal(4)
      await Lib_TimeboundRingBuffer.push(NON_NULL_BYTES32, NON_NULL_BYTES28)
      expect(await Lib_TimeboundRingBuffer.getLength()).to.equal(5)
    })

    it('should overwrite old values:[0,1,2,3] -> [4,5,2,3]', async () => {
      expect(await Lib_TimeboundRingBuffer.get(0)).to.equal(numToBytes32(0))
      await Lib_TimeboundRingBuffer.push(numToBytes32(4), NON_NULL_BYTES28)
      expect(await Lib_TimeboundRingBuffer.get(4)).to.equal(numToBytes32(4))
      await Lib_TimeboundRingBuffer.push(numToBytes32(5), NON_NULL_BYTES28)
      expect(await Lib_TimeboundRingBuffer.get(5)).to.equal(numToBytes32(5))
    })
  })

  describe('get()', () => {
    before(async () => {
      Lib_TimeboundRingBuffer = await (
        await ethers.getContractFactory('TestLib_TimeboundRingBuffer')
      ).deploy(2, 1, 10_000)
      await increaseEthTime(ethers.provider, 20_000)
      for (let i = 0; i < 4; i++) {
        await Lib_TimeboundRingBuffer.push(numToBytes32(i), NON_NULL_BYTES28)
      }
    })

    it('should revert if index is too old', async () => {
      await expect(Lib_TimeboundRingBuffer.get(0)).to.be.revertedWith("Index too old & has been overridden.")
    })

    it('should revert if index is greater than length', async () => {
      await expect(Lib_TimeboundRingBuffer.get(5)).to.be.revertedWith("Index too large.")
    })
  })

  describe('push with timeout', () => {
    const startSize = 2

    beforeEach(async () => {
      Lib_TimeboundRingBuffer = await (
        await ethers.getContractFactory('TestLib_TimeboundRingBuffer')
      ).deploy(startSize, 1, 10_000)
      for (let i = 0; i < startSize; i++) {
        await pushNum(i)
      }
    })

    const pushJunk = () => Lib_TimeboundRingBuffer.push(NON_NULL_BYTES32, NON_NULL_BYTES28)

    it('should push a single value which extends the array', async () => {
      await pushNum(2)
      const increasedSize = startSize + 1
      expect(await Lib_TimeboundRingBuffer.getMaxSize()).to.equal(increasedSize)

      await increaseEthTime(ethers.provider, 20_000)
      await pushNum(3)
      expect(await Lib_TimeboundRingBuffer.getMaxSize()).to.equal(increasedSize) // Shouldn't increase the size this time

      expect(await Lib_TimeboundRingBuffer.get(2)).to.equal(numToBytes32(2))
      expect(await Lib_TimeboundRingBuffer.get(3)).to.equal(numToBytes32(3))
    })

    it('should NOT extend the array if the time is not up and extend it when it is', async () => {
      await pushJunk()
      const increasedSize = startSize + 1
      expect(await Lib_TimeboundRingBuffer.getMaxSize()).to.equal(increasedSize)
      await increaseEthTime(ethers.provider, 20_000)
      // Push the time forward and verify that the time doesn't increment
      for (let i = 0; i < increasedSize + 1; i++) {
        await pushJunk()
      }
      expect(await Lib_TimeboundRingBuffer.getMaxSize()).to.equal(increasedSize)
    })
  })

  describe('push2 with timeout', () => {
    const startSize = 2

    beforeEach(async () => {
      Lib_TimeboundRingBuffer = await (
        await ethers.getContractFactory('TestLib_TimeboundRingBuffer')
      ).deploy(startSize, 1, 10_000)
    })

    it('should push a single value which extends the array', async () => {
      await push2Nums(0, 1)
      await push2Nums(2, 3)
      const increasedSize = startSize + 2
      expect(await Lib_TimeboundRingBuffer.getMaxSize()).to.equal(increasedSize)

      await increaseEthTime(ethers.provider, 20_000)
      await push2Nums(4, 5)
      expect(await Lib_TimeboundRingBuffer.getMaxSize()).to.equal(increasedSize) // Shouldn't increase the size this time

      for (let i = 2; i < 6; i++) {
        expect(await Lib_TimeboundRingBuffer.get(i)).to.equal(numToBytes32(i))
      }
    })
  })

  describe('getExtraData', () => {
    beforeEach(async () => {
      Lib_TimeboundRingBuffer = await (
        await ethers.getContractFactory('TestLib_TimeboundRingBuffer')
      ).deploy(2, 1, 10_000)
    })

    it('should return the expected extra data', async () => {
      await Lib_TimeboundRingBuffer.push(NON_NULL_BYTES32, NON_NULL_BYTES28)
      expect(await Lib_TimeboundRingBuffer.getExtraData()).to.equal(NON_NULL_BYTES28)
    })
  })

  describe('deleteElementsAfter', () => {
    // [0,1,2,3] -> [0,1,-,-]
    beforeEach(async () => {
      Lib_TimeboundRingBuffer = await (
        await ethers.getContractFactory('TestLib_TimeboundRingBuffer')
      ).deploy(4, 1, 0)
      for (let i = 0; i < 4; i++) {
        pushNum(i)
      }
    })

    it('should disallow deletions which are too old', async () => {
      push2Nums(4, 5)
      await expect(Lib_TimeboundRingBuffer.deleteElementsAfter(0, NON_NULL_BYTES28)).to.be.revertedWith("Attempting to delete too many elements.")
    })

    it('should not allow get to be called on an old value even after deletion', async () => {
      pushNum(4)
      expect(await Lib_TimeboundRingBuffer.getMaxSize()).to.equal(4)

      await expect(Lib_TimeboundRingBuffer.get(0)).to.be.revertedWith("Index too old & has been overridden.")
      Lib_TimeboundRingBuffer.deleteElementsAfter(3, NON_NULL_BYTES28)
      await expect(Lib_TimeboundRingBuffer.get(0)).to.be.revertedWith("Index too old & has been overridden.")
      await expect(Lib_TimeboundRingBuffer.get(4)).to.be.revertedWith("Index too large.")
      expect(await Lib_TimeboundRingBuffer.get(1)).to.equal(numToBytes32(1))
      expect(await Lib_TimeboundRingBuffer.get(3)).to.equal(numToBytes32(3))
    })

    it('should not reduce the overall size of the buffer', async () => {
      pushNum(4)
      expect(await Lib_TimeboundRingBuffer.get(1)).to.equal(numToBytes32(1))
      // We expect that we can still access `1` because the deletionOffset
      // will have reduced by 1 after we pushed.
      Lib_TimeboundRingBuffer.deleteElementsAfter(3, NON_NULL_BYTES28)
      expect(await Lib_TimeboundRingBuffer.get(1)).to.equal(numToBytes32(1))
    })
  })
})