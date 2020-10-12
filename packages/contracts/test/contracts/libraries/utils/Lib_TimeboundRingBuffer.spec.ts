/* tslint:disable:no-empty */
import { expect } from '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract, Signer } from 'ethers'

/* Internal Imports */
import {
  NON_NULL_BYTES32,
  makeHexString,
  fromHexString,
  getHexSlice,
  increaseEthTime
} from '../../../helpers'

const numToBytes32 = (num: Number) => {
  if (num < 0 || num > 255) {
    throw new Error('Unsupported number.')
  }
  const strNum = (num < 16) ? '0' + num.toString(16) : num.toString(16)
  return '0x' + '00'.repeat(31) + strNum
}

describe.only('Lib_TimeboundRingBuffer', () => {
  let signer: Signer
  before(async () => {
    ;[signer] = await ethers.getSigners()
  })

  let Lib_TimeboundRingBuffer: Contract

  const NON_NULL_BYTES28 = makeHexString('01', 28)

  describe('[0,1,2,3] with no timeout', () => {
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

  describe('push with timeout', () => {
    const startSize = 2
    beforeEach(async () => {
      Lib_TimeboundRingBuffer = await (
        await ethers.getContractFactory('TestLib_TimeboundRingBuffer')
      ).deploy(startSize, 1, 10_000)
      for (let i = 0; i < startSize; i++) {
        await Lib_TimeboundRingBuffer.push(numToBytes32(i), NON_NULL_BYTES28)
      }
    })

    const pushNum = (num: Number) => Lib_TimeboundRingBuffer.push(numToBytes32(num), NON_NULL_BYTES28)
    const pushJunk = () => Lib_TimeboundRingBuffer.push(NON_NULL_BYTES32, NON_NULL_BYTES28)

    it('should push a single value which extends the array', async () => {
      await Lib_TimeboundRingBuffer.push(numToBytes32(2), NON_NULL_BYTES28)
      const increasedSize = startSize + 1
      expect(await Lib_TimeboundRingBuffer.getMaxSize()).to.equal(increasedSize)

      await increaseEthTime(ethers.provider, 20_000)
      await Lib_TimeboundRingBuffer.push(numToBytes32(3), NON_NULL_BYTES28)
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
})