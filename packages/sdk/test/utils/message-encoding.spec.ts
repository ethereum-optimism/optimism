import { ethers, Contract } from 'ethers'
import hre from 'hardhat'

import { expect } from '../setup'
import {
  CoreCrossChainMessage,
  encodeCrossChainMessage,
  hashCrossChainMessage,
  encodeV0,
  encodeV1,
  hashWithdrawal,
} from '../../src'
import { DUMMY_MESSAGE } from '../helpers'

const addVersionToNonce = (
  nonce: ethers.BigNumber,
  version: number
): ethers.BigNumber => {
  return ethers.BigNumber.from(version).shl(240).or(nonce)
}

describe('message encoding utils', () => {
  let MessageEncodingHelper: Contract
  before(async () => {
    MessageEncodingHelper = (await (
      await hre.ethers.getContractFactory('MessageEncodingHelper')
    ).deploy()) as any
  })

  describe('encodeV0', () => {
    it('should properly encode a v0 message', async () => {
      const message: CoreCrossChainMessage = DUMMY_MESSAGE

      const actual = encodeV0(message)
      const expected = await MessageEncodingHelper.getVersionedEncoding(
        message.messageNonce,
        message.sender,
        message.target,
        message.value,
        message.minGasLimit,
        message.message
      )

      expect(actual).to.equal(expected)
    })
  })

  describe('encodeV1', () => {
    it('should properly encode a v1 message', async () => {
      const message: CoreCrossChainMessage = {
        ...DUMMY_MESSAGE,
        messageNonce: addVersionToNonce(DUMMY_MESSAGE.messageNonce, 1),
      }

      const actual = encodeV1(message)
      const expected = await MessageEncodingHelper.getVersionedEncoding(
        message.messageNonce,
        message.sender,
        message.target,
        message.value,
        message.minGasLimit,
        message.message
      )

      expect(actual).to.equal(expected)
    })
  })

  describe('encodeCrossChainMessage', () => {
    it('should return the v0 encoding for a v0 message', async () => {
      const message: CoreCrossChainMessage = DUMMY_MESSAGE

      const actual = encodeCrossChainMessage(message)
      const expected = encodeV0(message)

      expect(actual).to.equal(expected)
    })

    it('should return the v1 encoding for a v1 message', async () => {
      const message: CoreCrossChainMessage = {
        ...DUMMY_MESSAGE,
        messageNonce: addVersionToNonce(DUMMY_MESSAGE.messageNonce, 1),
      }

      const actual = encodeCrossChainMessage(message)
      const expected = encodeV1(message)

      expect(actual).to.equal(expected)
    })
  })

  describe('hashCrossChainMessage', () => {
    it('should properly hash a v0 message', async () => {
      const message: CoreCrossChainMessage = DUMMY_MESSAGE

      const actual = hashCrossChainMessage(message)
      const expected = await MessageEncodingHelper.getVersionedHash(
        message.messageNonce,
        message.sender,
        message.target,
        message.value,
        message.minGasLimit,
        message.message
      )

      expect(actual).to.equal(expected)
    })

    it('should properly hash a v1 message', async () => {
      const message: CoreCrossChainMessage = {
        ...DUMMY_MESSAGE,
        messageNonce: addVersionToNonce(DUMMY_MESSAGE.messageNonce, 1),
      }

      const actual = hashCrossChainMessage(message)
      const expected = await MessageEncodingHelper.getVersionedHash(
        message.messageNonce,
        message.sender,
        message.target,
        message.value,
        message.minGasLimit,
        message.message
      )

      expect(actual).to.equal(expected)
    })
  })

  describe('hashWithdrawal', () => {
    it('should properly hash a withdrawal message', async () => {
      const message: CoreCrossChainMessage = DUMMY_MESSAGE

      const actual = hashWithdrawal(message)
      const expected = await MessageEncodingHelper.withdrawalHash(
        message.messageNonce,
        message.sender,
        message.target,
        message.value,
        message.minGasLimit,
        message.message
      )

      expect(actual).to.equal(expected)
    })
  })
})
