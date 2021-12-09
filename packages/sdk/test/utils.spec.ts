import { expect } from './setup'
import { Provider } from '@ethersproject/abstract-provider'
import { Contract, Signer } from 'ethers'
import { ethers } from 'hardhat'
import { getContractFactory } from '@eth-optimism/contracts'
import {
  toProvider,
  toTransactionHash,
  CrossChainMessage,
  MessageDirection,
  encodeCrossChainMessage,
  hashCrossChainMessage,
} from '../src'

describe('utils', () => {
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
      const message: CrossChainMessage = {
        direction: MessageDirection.L1_TO_L2,
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
      const message: CrossChainMessage = {
        direction: MessageDirection.L1_TO_L2,
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

  describe('toProvider', () => {
    it('should convert a string to a JsonRpcProvider', () => {
      const provider = toProvider('http://localhost:8545')
      expect(Provider.isProvider(provider)).to.be.true
    })

    it('should not do anything with a provider', () => {
      const provider = toProvider(ethers.provider)
      expect(provider).to.deep.equal(ethers.provider)
    })
  })

  describe('toTransactionHash', () => {
    describe('string inputs', () => {
      it('should return the input if the input is a valid transaction hash', () => {
        const input = '0x' + '11'.repeat(32)
        expect(toTransactionHash(input)).to.equal(input)
      })

      it('should throw an error if the input is a hex string but not a transaction hash', () => {
        const input = '0x' + '11'.repeat(31)
        expect(() => toTransactionHash(input)).to.throw(
          'Invalid transaction hash'
        )
      })

      it('should throw an error if the input is not a hex string', () => {
        const input = 'hi mom look at me go'
        expect(() => toTransactionHash(input)).to.throw(
          'Invalid transaction hash'
        )
      })
    })

    describe('transaction inputs', () => {
      let AbsolutelyNothing: Contract
      before(async () => {
        AbsolutelyNothing = (await (
          await ethers.getContractFactory('AbsolutelyNothing')
        ).deploy()) as any
      })

      it('should return the transaction hash if the input is a transaction response', async () => {
        const tx = await AbsolutelyNothing.doAbsolutelyNothing()
        expect(toTransactionHash(tx)).to.equal(tx.hash)
      })

      it('should return the transaction hash if the input is a transaction receipt', async () => {
        const tx = await AbsolutelyNothing.doAbsolutelyNothing()
        const receipt = await tx.wait()
        expect(toTransactionHash(receipt)).to.equal(receipt.transactionHash)
      })
    })

    describe('other types', () => {
      it('should throw if given a number as an input', () => {
        expect(() => toTransactionHash(1234 as any)).to.throw(
          'Invalid transaction'
        )
      })

      it('should throw if given a function as an input', () => {
        expect(() =>
          toTransactionHash((() => {
            return 1234
          }) as any)
        ).to.throw('Invalid transaction')
      })
    })
  })
})
