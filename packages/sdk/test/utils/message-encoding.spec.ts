import { expect } from '../setup'
import { Contract, Signer } from 'ethers'
import { ethers } from 'hardhat'
import { getContractFactory } from '@eth-optimism/contracts'
import {
  CoreCrossChainMessage,
  encodeCrossChainMessage,
  hashCrossChainMessage,
} from '../../src'

describe('message encoding utils', () => {
  let signers: Signer[]
  before(async () => {
    signers = (await ethers.getSigners()) as any
  })

  describe('encodeCrossChainMessage', () => {
    let Lib_CrossDomainUtils: Contract
    before(async () => {
      Lib_CrossDomainUtils = (await getContractFactory(
        'TestLib_CrossDomainUtils',
        signers[0]
      ).deploy()) as any
    })

    it('should properly encode a message', async () => {
      const message: CoreCrossChainMessage = {
        target: '0x' + '11'.repeat(20),
        sender: '0x' + '22'.repeat(20),
        message: '0x' + '1234'.repeat(32),
        messageNonce: 1234,
      }

      const actual = encodeCrossChainMessage(message)
      const expected = await Lib_CrossDomainUtils.encodeXDomainCalldata(
        message.target,
        message.sender,
        message.message,
        message.messageNonce
      )
      expect(actual).to.equal(expected)
    })
  })

  describe('hashCrossChainMessage', () => {
    let MessageEncodingHelper: Contract
    before(async () => {
      MessageEncodingHelper = (await (
        await ethers.getContractFactory('MessageEncodingHelper')
      ).deploy()) as any
    })

    it('should properly hash a message', async () => {
      const message: CoreCrossChainMessage = {
        target: '0x' + '11'.repeat(20),
        sender: '0x' + '22'.repeat(20),
        message: '0x' + '1234'.repeat(32),
        messageNonce: 1234,
      }

      const actual = hashCrossChainMessage(message)
      const expected = await MessageEncodingHelper.hashXDomainCalldata(
        message.target,
        message.sender,
        message.message,
        message.messageNonce
      )
      expect(actual).to.equal(expected)
    })
  })
})
